package server

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"

	"github.com/stts-se/weblib/userdb"
)

type invitations struct {
	mutex  *sync.RWMutex
	tokens map[string]time.Time
	maxAge float64 // max age in seconds
}

// NB! not thread safe -- lock mutex before calling
func (i *invitations) purge() {
	for token, created := range i.tokens {
		if time.Since(created).Seconds() > i.maxAge {
			delete(i.tokens, token)
			log.Printf("Expired token: %s", token)
		}
	}
	log.Printf("Purged invitation db")
}

// Auth holds an authorization handler using a user database along with sessions and cookies
type Auth struct {
	sessionName string
	userDB      *userdb.UserDB
	cookieStore *sessions.CookieStore
	invitations invitations
}

// NewAuth create a new Auth instance
func NewAuth(sessionName string, userDB *userdb.UserDB, cookieStore *sessions.CookieStore) Auth {
	return Auth{
		sessionName: sessionName,
		userDB:      userDB,
		cookieStore: cookieStore,
		invitations: invitations{
			mutex:  &sync.RWMutex{},
			tokens: make(map[string]time.Time),
			maxAge: 86400 * 7, // one week in seconds
		},
	}
}

// Login : log in a user with the specified username and password, creating a new auth session for the user
func (a *Auth) Login(w http.ResponseWriter, r *http.Request, userName, password string) error {

	ok, err := a.userDB.Authorized(userName, password)
	if err != nil {
		return fmt.Errorf("login failed : %v", err)
	}
	if ok {
		session, err := a.cookieStore.Get(r, a.sessionName)
		if err != nil {
			return fmt.Errorf("couldn't get session : %v", err)
		}

		session.Options = &sessions.Options{
			Path:     "/",
			MaxAge:   86400 * 7, // one week
			HttpOnly: true,
		}
		//log.Printf("Session %#v", session)

		// Set user as authenticated
		session.Values["authenticated-user"] = userName
		session.Save(r, w)
		//log.Printf("User %s logged in successfully", a.GetLoggedInUserName(r))
		return nil
	}
	return fmt.Errorf("login failed")
}

// CreateSingleUseToken creates a single use token, useful for signup invitations
func (a *Auth) CreateSingleUseToken() (string, error) {
	token := uuid.New().String()
	a.invitations.mutex.RLock()
	defer a.invitations.mutex.RUnlock()

	a.invitations.purge()

	if _, ok := a.invitations.tokens[token]; ok {
		err := fmt.Errorf("token already exists: %s", token)
		log.Println(err)
		//http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", err
	}
	a.invitations.tokens[token] = time.Now()
	return token, nil
}

// SignupUser creates a new user with the specified userName and password, using the specified token. If the signup is successful, the token will be consumed.
func (a *Auth) SignupUser(userName, password, singleUseToken string) error {
	a.invitations.mutex.RLock()
	defer a.invitations.mutex.RUnlock()

	a.invitations.purge()

	// verify token
	created, tokenExists := a.invitations.tokens[singleUseToken]
	if !tokenExists {
		err := fmt.Errorf("invalid token : %s", singleUseToken)
		log.Println(err)
		return err
	}
	if time.Since(created).Seconds() > a.invitations.maxAge {
		delete(a.invitations.tokens, singleUseToken)
		err := fmt.Errorf("expired token: %s", singleUseToken)
		log.Println(err)
		return err
	}

	err := a.userDB.InsertUser(userName, password)
	if err != nil {
		err := fmt.Errorf("signup failed : %v", err)
		log.Println(err)
		return err
	}
	delete(a.invitations.tokens, singleUseToken)
	//log.Printf("Created used %s", userName)
	return nil
}

// Logout : logout current user
func (a *Auth) Logout(w http.ResponseWriter, r *http.Request) (string, error) {
	session, err := a.cookieStore.Get(r, a.sessionName)
	if err != nil {
		return "", fmt.Errorf("couldn't get session : %v", err)
	}
	userName := session.Values["authenticated-user"].(string)

	// Revoke users authentication
	session.Values["authenticated-user"] = ""
	session.Options.MaxAge = -1
	session.Save(r, w)
	//log.Printf("User %s logged out successfully", userName)
	return userName, nil
}

func (a *Auth) GetLoggedInUserName(r *http.Request) string {
	session, err := a.cookieStore.Get(r, a.sessionName)
	if err != nil {
		//log.Printf("Couldn't get user session : %v", err)
		return ""
	}
	if auth, ok := session.Values["authenticated-user"].(string); ok {
		return auth
	}
	return ""
}

func (a *Auth) IsLoggedIn(r *http.Request) bool {
	session, err := a.cookieStore.Get(r, a.sessionName)
	//log.Printf("Session %#v\n", session)
	if err != nil {
		//log.Printf("Couldn't get user session : %v", err)
		return false
	}
	if auth, ok := session.Values["authenticated-user"].(string); ok && auth != "" {
		return true
	}
	return false
}

// RequireAuthUser is used as middle ware to protect a path, e.g., router.Use(auth.RequireAuthUser)
func (a *Auth) RequireAuthUser(authFunc http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.IsLoggedIn(r) {
			authFunc.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}

// ServeAuthUser will call authFunc if there is an authorized user; else unauthFunc will be called
func (a *Auth) ServeAuthUser(authFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if a.IsLoggedIn(r) {
			// TODO: Check if user has access rights to the specified level?
			authFunc(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// ServeAuthUserOrElse will call authFunc if there is an authorized user; else unauthFunc will be called
func (a *Auth) ServeAuthUserOrElse(authFunc http.HandlerFunc, unauthFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if a.IsLoggedIn(r) {
			// TODO: Check if user has access rights to the specified level?
			authFunc(w, r)
		} else {
			unauthFunc(w, r)
		}
	}
}

// ListUsers list users (user names) in the database
func (a *Auth) ListUsers() []string {
	return a.userDB.GetUsers()
}

// SaveUserDB : save user database to disk
func (a *Auth) SaveUserDB() error {
	return a.userDB.SaveFile()
}

// UTILITY FUNCTIONS
func genRandomString(length int) string {
	chars := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ" + "abcdefghijkmnopqrstuvwxyz" + "123456789_")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}
