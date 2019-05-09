package main

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
)

var sessionName = "auth-user-weblib"

func parseForm(r *http.Request, requiredParams []string) (map[string]string, bool) {
	res := make(map[string]string)
	missing := []string{}
	if err := r.ParseForm(); err != nil {
		log.Printf("Couldn't parse form : %v", err)
		return res, false
	}
	for _, param := range requiredParams {
		value := r.FormValue(param)
		if value != "" {
			res[param] = value
		} else {
			missing = append(missing, param)
		}
	}
	if len(missing) > 0 {
		pluralS := "s"
		if len(missing) == 1 {
			pluralS = ""
		}
		log.Printf("Couldn't parse form : missing param%s : %v", pluralS, missing)
	}
	return res, true
}

func login(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/login.html")
		return
	case "POST":
		form, ok := parseForm(r, []string{"username", "password"})
		userName := form["username"]
		password := form["password"]
		if !ok {
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}

		ok, err := userDB.Authorized(userName, password)
		if err != nil {
			log.Printf("Login failed : %v", err)
			http.Error(w, "Login failed", http.StatusUnauthorized)
			return
		}
		if ok {
			session, err := cookieStore.Get(r, sessionName)
			if err != nil {
				log.Printf("Couldn't get session : %v", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}

			session.Options = &sessions.Options{
				Path:     "/",
				MaxAge:   86400 * 7, // one week
				HttpOnly: true,
			}

			log.Printf("Session %v", session)

			// Set user as authenticated
			session.Values["authenticated-user"] = userName
			session.Save(r, w)
			log.Printf("User %s logged in successfully", getLoggedInUserName(r))
			fmt.Fprintf(w, "Logged in successfully as user %s\n", getLoggedInUserName(r))
			return
		}
		http.Error(w, "Login failed", http.StatusUnauthorized)
		return

	default:
		http.NotFound(w, r)
	}

}

type invitationHolder struct {
	mutex  *sync.RWMutex
	tokens map[string]time.Time
	maxAge float64 // max age in seconds
}

var invitations = invitationHolder{
	mutex:  &sync.RWMutex{},
	tokens: make(map[string]time.Time),
	maxAge: 86400 * 7, // one week in seconds
}

func genRandomString(length int) string {
	chars := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ" + "abcdefghijkmnopqrstuvwxyz" + "123456789_")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func invite(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/invite.html")
		return
	case "POST":
		token := uuid.New().String()
		invitations.mutex.RLock()
		defer invitations.mutex.RUnlock()

		purgeInvitations()

		if _, ok := invitations.tokens[token]; ok {
			log.Printf("Token already exists : %s", token)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		invitations.tokens[token] = time.Now()

		link := fmt.Sprintf("%s://%s/auth/signup?token=%s", serverProtocol, serverAddress, token)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("Created invitation link: %s", link)
		fmt.Fprintf(w, "Invitation link: <a href='%s'>%s</a>\n", link, link)
	default:
		http.NotFound(w, r)
	}

}

// NB! not thread safe -- lock mutex before calling
func purgeInvitations() {
	for token, created := range invitations.tokens {
		if time.Since(created).Seconds() > invitations.maxAge {
			delete(invitations.tokens, token)
			log.Printf("Expired token: %s", token)
		}
	}
	log.Printf("Purged invitation db")
}

func signup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/signup.html")
		return
	case "POST":
		form, ok := parseForm(r, []string{"username", "password", "token"})
		userName := form["username"]
		password := form["password"]
		token := form["token"]
		if !ok {
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}

		invitations.mutex.RLock()
		defer invitations.mutex.RUnlock()

		purgeInvitations()

		// verify token
		created, tokenExists := invitations.tokens[token]
		if !tokenExists {
			log.Printf("Invalid token : %s", token)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		if time.Since(created).Seconds() > invitations.maxAge {
			delete(invitations.tokens, token)
			log.Printf("Expired token: %s", token)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		err := userDB.InsertUser(userName, password)
		if err != nil {
			log.Printf("Signup failed : %v", err)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}
		delete(invitations.tokens, token)
		log.Printf("Created used %s", userName)
		fmt.Fprintf(w, "Created user %s\n", userName)
		return

	default:
		http.NotFound(w, r)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/logout.html")
		return
	case "POST":
		session, err := cookieStore.Get(r, sessionName)
		if err != nil {
			log.Printf("Couldn't get session : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		userName := session.Values["authenticated-user"]

		// Revoke users authentication
		session.Values["authenticated-user"] = ""
		session.Options.MaxAge = -1
		session.Save(r, w)
		log.Printf("User %s logged out successfully", userName)
		fmt.Fprintf(w, "Logged out user %s successfully\n", userName)
	default:
		http.NotFound(w, r)
	}
}

func getLoggedInUserName(r *http.Request) string {
	session, err := cookieStore.Get(r, sessionName)
	if err != nil {
		log.Printf("Couldn't get session : %v", err)
		return ""
	}
	if auth, ok := session.Values["authenticated-user"].(string); ok {
		return auth
	}
	return ""
}

func isLoggedIn(r *http.Request) bool {
	session, err := cookieStore.Get(r, sessionName)
	if err != nil {
		log.Printf("Couldn't get session : %v", err)
		return false
	}
	if auth, ok := session.Values["authenticated-user"].(string); ok && auth != "" {
		return true
	}
	return false
}

func loggedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isLoggedIn(r) {
			next.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}

// authUser will call authFunc if there is an authorized user; else unauthFunc will be called
func authUser(authFunc http.HandlerFunc, unauthFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		if isLoggedIn(r) {
			// TODO: Check if user has access rights to the specified level
			authFunc(w, r)
		} else {
			unauthFunc(w, r)
		}
	}
}
