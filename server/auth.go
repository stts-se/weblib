package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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

func invite(w http.ResponseWriter, r *http.Request) {

	// TODO: secure params
	//email, _ := r.BasicAuth()
	email := getParam("email", r)

	if email != "" {
		// TODO
		uuid, err := genUUID(32)
		if err != nil {
			log.Printf("genUUID failed : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		invitationDB.mutex.RLock()
		defer invitationDB.mutex.RUnlock()

		if _, ok := invitationDB.invitations[uuid]; ok {
			log.Printf("uuid already exists : %s", uuid)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		invitationDB.invitations[uuid] = time.Now()

		// userName := email
		// password := "12345678"
		// link := fmt.Sprintf("%s://%s/auth/signup?uuid=%s&username=%s&password=%s", serverProtocol, serverAddress, uuid, userName, password)
		link := fmt.Sprintf("%s://%s/auth/signup?uuid=%s", serverProtocol, serverAddress, uuid)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `Invitation link: <a href="%s">%s</a>`, link, link)
		return
	}
	http.Error(w, "No invitation credentials provided", http.StatusBadRequest)
}

func signup(w http.ResponseWriter, r *http.Request) {

	// TODO: secure params
	//userName, password, hash, _ := r.BasicAuth()
	uuid := getParam("uuid", r)
	userName := getParam("username", r)
	password := getParam("password", r)
	if uuid != "" && userName != "" && password != "" {

		invitationDB.mutex.RLock()
		defer invitationDB.mutex.RUnlock()

		// verify uuid
		created, uuidExists := invitationDB.invitations[uuid]
		if !uuidExists {
			log.Printf("Unknown uuid : %s", uuid)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if time.Since(created).Seconds() > invitationDB.maxAge {
			delete(invitationDB.invitations, uuid)
			log.Printf("Expired uuid created at %v: %s", created, uuid)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		err := userDB.InsertUser(userName, password)
		if err != nil {
			log.Printf("Signup failed : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
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
