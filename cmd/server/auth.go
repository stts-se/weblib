package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/stts-se/weblib"
	"github.com/stts-se/weblib/auth"
)

type AuthHandlers struct {
	ServerURL string
	Auth      *auth.Auth
}

func (a *AuthHandlers) helloWorld(w http.ResponseWriter, r *http.Request) {
	var msg string
	if userName, ok := a.Auth.IsLoggedIn(r); ok {
		msg = fmt.Sprintf("Hello, you are logged in as user %s!", userName)
	} else {
		msg = "Hello, you are not logged in."
	}
	fmt.Fprintf(w, "%s\n", msg)
}

func (a *AuthHandlers) message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if uName, ok := a.Auth.IsLoggedIn(r); ok {
			msg = strings.Replace(msg, "${username}", uName, -1)
		}
		fmt.Fprintf(w, "%s\n", msg)
	}
}

func (a *AuthHandlers) login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/login.html")
		return
	case "POST":
		form, ok := weblib.ParseForm(r, []string{"username", "password"})
		userName := form["username"]
		password := form["password"]
		if !ok {
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}

		err := a.Auth.Login(w, r, userName, password)
		if err != nil {
			log.Printf("Login failed : %v", err)
			http.Error(w, "Login failed", http.StatusUnauthorized)
			return
		}
		log.Printf("User %s logged in successfully", userName)
		fmt.Fprintf(w, "Logged in successfully as user %s\n", userName)
		return

	default:
		http.NotFound(w, r)
	}
}

func (a *AuthHandlers) invite(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/invite.html")
		return
	case "POST":
		token, err := a.Auth.CreateSingleUseToken()
		if err != nil {
			log.Printf("Couldn't create invitation token : %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		link := fmt.Sprintf("%s/auth/signup?token=%s", a.ServerURL, token)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("Created invitation link: %s", link)
		fmt.Fprintf(w, "Invitation link: <a href='%s'>%s</a>\n", link, link)
	default:
		http.NotFound(w, r)
	}
}

func (a *AuthHandlers) signup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/signup.html")
		return
	case "POST":
		form, ok := weblib.ParseForm(r, []string{"username", "password", "token"})
		userName := form["username"]
		password := form["password"]
		token := form["token"]
		if !ok {
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}

		err := a.Auth.SignupUser(userName, password, token)
		if err != nil {
			log.Printf("Couldn't create user : %s", err)
			http.Error(w, "Internal server error", http.StatusUnauthorized)
			return
		}
		log.Printf("Created used %s", userName)
		fmt.Fprintf(w, "Created user %s\n", userName)
		return

	default:
		http.NotFound(w, r)
	}
}

func (a *AuthHandlers) logout(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		http.ServeFile(w, r, "static/auth/logout.html")
		return
	case "POST":
		userName, err := a.Auth.Logout(w, r)
		if err != nil {
			log.Printf("Couldn't logout : %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("User %s logged out successfully", userName)
		fmt.Fprintf(w, "Logged out user %s successfully\n", userName)
	default:
		http.NotFound(w, r)
	}
}

func (a *AuthHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Users\n")
	for _, uName := range a.Auth.ListUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}
