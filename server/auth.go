package main

import (
	"log"
	"net/http"

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
	//userName, password, _ := r.BasicAuth()
	userName := getParam("username", r)
	password := getParam("password", r)

	if userName != "" && password != "" {
		ok, err := userDB.Authorized(userName, password)
		if err != nil {
			log.Printf("%v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		if ok {
			// Set user as authenticated
			session.Values["authenticated"] = true
			session.Save(r, w)
			http.Error(w, "Logged in", http.StatusOK)
			return
		}
		http.Error(w, "Login failed", http.StatusForbidden)
		return
	} else {
		http.Error(w, "No login credentials provided", http.StatusForbidden)
		return
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, sessionName)

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
}

func userLoggedIn(r *http.Request) bool {
	session, _ := cookieStore.Get(r, sessionName)

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		return false
	}
	return true
}

func notLoggedIn(pass http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if userLoggedIn(r) {
			http.Error(w, "Already logged in", http.StatusForbidden)
			return
		}
		pass(w, r)
	}
}

func requireAccessRights(pass http.HandlerFunc) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if userLoggedIn(r) {
			pass(w, r)
		}

		// if !userLoggedIn(r) {
		// 	http.Error(w, "Not logged in", http.StatusForbidden)
		// 	return
		// }

		// // TODO: Check if user has access rights to the specified level

		// pass(w, r)
	}
}
