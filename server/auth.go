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

var sessionName = "auth-user-session"

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, sessionName)

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // one week
		HttpOnly: true,
	}

	// TODO: secure params
	//userName, password, _ := r.BasicAuth()
	userName := getParam("username", r)
	password := getParam("password", r)

	if userName != "" && password != "" {
		ok, err := userDB.Authorized(userName, password)
		if err != nil {
			log.Printf("Login failed : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if ok {
			// Set user as authenticated
			session.Values["authenticated"] = true
			session.Save(r, w)
			log.Printf("Logged in successfully")
			fmt.Fprintf(w, "Logged in successfully\n")
			return
		}
		http.Error(w, "Login failed", http.StatusForbidden)
		return
	}
	http.Error(w, "No login credentials provided", http.StatusBadRequest)
}

type invitationsHolder struct {
	mutex       *sync.RWMutex
	invitations map[string]time.Time
	maxAge      float64 // max age in seconds
}

var invitationDB = invitationsHolder{
	mutex:       &sync.RWMutex{},
	invitations: make(map[string]time.Time),
	maxAge:      86400 * 7, // one week in seconds
}

func genPassword(length int) string {
	chars := []rune("ABCDEFGHJKLMNPQRSTUVWXYZ" +
		"abcdefghijkmnopqrstuvwxyz" +
		"123456789()_")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

func invite(w http.ResponseWriter, r *http.Request) {

	// TODO: secure params
	//email, _ := r.BasicAuth()
	email := getParam("email", r)

	if email != "" {
		token := uuid.New().String()
		invitationDB.mutex.RLock()
		defer invitationDB.mutex.RUnlock()

		purgeInvitations()

		if _, ok := invitationDB.invitations[token]; ok {
			log.Printf("Token already exists : %s", token)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		invitationDB.invitations[token] = time.Now()

		userName := email
		password := genPassword(10)
		link := fmt.Sprintf("%s://%s/auth/signup?token=%s&username=%s&password=%s", serverProtocol, serverAddress, token, userName, password)
		// link := fmt.Sprintf("%s://%s/auth/signup?token=%s", serverProtocol, serverAddress, token)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("Created invitation link: %s", link)
		fmt.Fprintf(w, `Invitation link: <a href="%s">%s</a>`, link, link)
		return
	}
	http.Error(w, "No invitation credentials provided", http.StatusBadRequest)
}

// not thread safe -- lock mutex before calling
func purgeInvitations() {
	for token, created := range invitationDB.invitations {
		if time.Since(created).Seconds() > invitationDB.maxAge {
			delete(invitationDB.invitations, token)
			log.Printf("Token expired: %s", token)
		}
	}
	log.Printf("Purged invitation db")
}

func signup(w http.ResponseWriter, r *http.Request) {
	// TODO: secure params
	//userName, password, hash, _ := r.BasicAuth()
	token := getParam("token", r)
	userName := getParam("username", r)
	password := getParam("password", r)
	if token != "" && userName != "" && password != "" {

		invitationDB.mutex.RLock()
		defer invitationDB.mutex.RUnlock()

		purgeInvitations()

		// verify token
		created, tokenExists := invitationDB.invitations[token]
		if !tokenExists {
			log.Printf("Unknown token : %s", token)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if time.Since(created).Seconds() > invitationDB.maxAge {
			delete(invitationDB.invitations, token)
			log.Printf("Expired token: %s", token)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err := userDB.InsertUser(userName, password)
		if err != nil {
			log.Printf("Signup failed : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		delete(invitationDB.invitations, token)
		log.Printf("Created used %s", userName)
		fmt.Fprintf(w, "Created user %s\n", userName)
		return
	}
	http.Error(w, "No signup credentials provided", http.StatusBadRequest)
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, sessionName)

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
	log.Printf("Logged out successfully")
	fmt.Fprintf(w, "Logged out successfully\n")
}

// authUser will call authFunc if there is an authorized user; else unauthFunc will be called
func authUser(authFunc http.HandlerFunc, unauthFunc http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		session, _ := cookieStore.Get(r, sessionName)
		if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
			unauthFunc(w, r)
		} else {
			// TODO: Check if user has access rights to the specified level
			authFunc(w, r)
		}
	}
}
