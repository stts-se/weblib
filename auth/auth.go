// Package auth is a library for user authentication over http.
package auth

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/stts-se/weblib/userdb"
)

type singleUseTokens struct {
	mutex  *sync.RWMutex
	tokens map[string]time.Time
	maxAge float64 // max age in seconds
}

// NB! not thread safe -- mutex should be locked before calling
func (sit *singleUseTokens) purge() {
	for token, created := range sit.tokens {
		if time.Since(created).Seconds() > sit.maxAge {
			delete(sit.tokens, token)
			log.Printf("Expired token: %s", token)
		}
	}
	log.Printf("Purged single use tokens db")
}

// Auth : authentication management, using a user database along with sessions and cookies
type Auth struct {
	sessionName     string
	userDB          *userdb.UserDB
	roleDB          *userdb.RoleDB
	cookieStore     *sessions.CookieStore
	singleUseTokens singleUseTokens
}

// NewAuth create a new Auth instance
func NewAuth(sessionName string, userDB *userdb.UserDB, roleDB *userdb.RoleDB, cookieStore *sessions.CookieStore) (*Auth, error) {
	res := &Auth{
		sessionName: sessionName,
		userDB:      userDB,
		roleDB:      roleDB,
		cookieStore: cookieStore,
		singleUseTokens: singleUseTokens{
			mutex:  &sync.RWMutex{},
			tokens: make(map[string]time.Time),
			maxAge: 86400 * 7, // one week in seconds
		},
	}
	err := userdb.Validate(res.userDB, res.roleDB)
	if err != nil {
		return res, fmt.Errorf("userdb/roledb validation failed : %v", err)
	}
	return res, nil
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
		return nil
	}
	return fmt.Errorf("login failed")
}

// CreateSingleUseToken creates a single use token, useful for signup invitations
func (a *Auth) CreateSingleUseToken() (string, error) {
	token := uuid.New().String()
	a.singleUseTokens.mutex.RLock()
	defer a.singleUseTokens.mutex.RUnlock()

	a.singleUseTokens.purge()

	if _, ok := a.singleUseTokens.tokens[token]; ok {
		err := fmt.Errorf("token already exists: %s", token)
		log.Println(err)
		//http.Error(w, "Internal server error", http.StatusInternalServerError)
		return "", err
	}
	a.singleUseTokens.tokens[token] = time.Now()
	return token, nil
}

// SignupUser creates a new user with the specified userName and password, using the specified token. If the signup is successful, the token will be consumed.
func (a *Auth) SignupUser(userName, password, singleUseToken string) error {
	a.singleUseTokens.mutex.RLock()
	defer a.singleUseTokens.mutex.RUnlock()

	a.singleUseTokens.purge()

	// verify token
	created, tokenExists := a.singleUseTokens.tokens[singleUseToken]
	if !tokenExists {
		err := fmt.Errorf("invalid token : %s", singleUseToken)
		log.Println(err)
		return err
	}
	if time.Since(created).Seconds() > a.singleUseTokens.maxAge {
		delete(a.singleUseTokens.tokens, singleUseToken)
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
	delete(a.singleUseTokens.tokens, singleUseToken)
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
	return userName, nil
}

// IsLoggedIn : Check if there the user is logged in. Second return value is the user name.
func (a *Auth) IsLoggedIn(r *http.Request) (bool, string) {
	session, err := a.cookieStore.Get(r, a.sessionName)
	if err != nil {
		return false, ""
	}
	if auth, ok := session.Values["authenticated-user"].(string); ok && auth != "" {
		return a.userDB.UserExists(auth)
	}
	return false, ""
}

// IsLoggedInWithRole : check if a user is logged in with the specified role.  Second return value is the user name.
func (a *Auth) IsLoggedInWithRole(r *http.Request, roleName string) (bool, string) {
	if ok, authUser := a.IsLoggedIn(r); ok && authUser != "" {
		if a.roleDB.Authorized(roleName, authUser) {
			return true, authUser
		}
		return false, ""
	}
	return false, ""
}

// RequireAuthUser is used as middle ware to protect a path
func (a *Auth) RequireAuthUser(route *mux.Router) {
	var f = func(authFunc http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok, _ := a.IsLoggedIn(r); ok {
				authFunc.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}
	route.Use(f)
}

// RequireAuthRole is used as middle ware to protect a path
func (a *Auth) RequireAuthRole(route *mux.Router, roleName string) {
	var f = func(authFunc http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if ok, _ := a.IsLoggedInWithRole(r, roleName); ok {
				authFunc.ServeHTTP(w, r)
			} else {
				http.NotFound(w, r)
			}
		})
	}
	route.Use(f)
}

// ServeAuthUser will call authFunc if there is an authorized user; else unauthFunc will be called
func (a *Auth) ServeAuthUser(authFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if ok, _ := a.IsLoggedIn(r); ok {
			authFunc(w, r)
		} else {
			http.NotFound(w, r)
		}
	}
}

// ServeAuthUserOrElse will call authFunc if there is an authorized user; else unauthFunc will be called
func (a *Auth) ServeAuthUserOrElse(authFunc http.HandlerFunc, unauthFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if ok, _ := a.IsLoggedIn(r); ok {
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
