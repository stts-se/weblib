package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/stts-se/weblib/auth"
	"github.com/stts-se/weblib/util"
)

type authHandlers struct {
	Auth *auth.Auth
}

func (a *authHandlers) helloWorld(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	var msg string
	if ok, userName := a.Auth.IsLoggedIn(r); ok {
		msg = cli18n.S("Hello, you are logged in as user %s!", userName)
	} else {
		msg = cli18n.S("Hello, you are not logged in.")
	}
	fmt.Fprintf(w, "%s\n", msg)
}

func (a *authHandlers) message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli18n := i18nCache.GetI18NFromRequest(r)
		msgLoc := cli18n.S(msg)
		if ok, uName := a.Auth.IsLoggedIn(r); ok {
			msgLoc = strings.Replace(msgLoc, "${username}", uName, -1)
		}
		fmt.Fprintf(w, "%s\n", msgLoc)
	}
}

func (a *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	switch r.Method {
	case "GET":
		err := templates.ExecuteTemplate(w, "login.html", TemplateData{Loc: cli18n})
		if err != nil {
			log.Printf("Couldn't execute template : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	case "POST":
		form, err := util.ParseForm(r, []string{"username", "password"})
		if err != nil {
			log.Printf("Couldn't parse form : %v", err)
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}
		userName := form["username"]
		password := form["password"]

		err = a.Auth.Login(w, r, userName, password)
		if err != nil {
			log.Printf("Login failed : %v", err)
			http.Error(w, "Login failed", http.StatusUnauthorized)
			return
		}
		log.Printf("User %s logged in", userName)
		msg := cli18n.S("Logged in as user %s", userName) + "\n"
		fmt.Fprint(w, msg)
		return

	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) invite(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	switch r.Method {
	case "GET":
		err := templates.ExecuteTemplate(w, "invite.html", TemplateData{Loc: cli18n})
		if err != nil {
			log.Printf("Couldn't execute template : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	case "POST":
		token, err := a.Auth.CreateSingleUseToken()
		if err != nil {
			log.Printf("Couldn't create invitation token : %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		link := fmt.Sprintf("%s/auth/signup?token=%s", util.GetServerURL(r), url.PathEscape(token))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("Created invitation link: %s", link)
		msg := fmt.Sprintf("%s: <a href='%s'>%s</a>\n", cli18n.S("Invitation link"), link, link)
		fmt.Fprint(w, msg)
	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) signup(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	switch r.Method {
	case "GET":
		token, err := url.PathUnescape(util.GetParam(r, "token"))
		if err != nil {
			log.Printf("Couldn't unescape param token : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		if len(token) == 0 {
			log.Printf("Empty token")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		if len(token) < 10 {
			log.Printf("Invalid token")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
		data := struct{ Token string }{Token: token}
		err = templates.ExecuteTemplate(w, "signup.html", TemplateData{Loc: cli18n, Data: data})
		if err != nil {
			log.Printf("Couldn't execute template : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	case "POST":
		form, err := util.ParseForm(r, []string{"username", "password", "token"})
		if err != nil {
			log.Printf("Couldn't parse form : %v", err)
			http.Error(w, "Incomplete credentials", http.StatusUnauthorized)
		}
		userName := form["username"]
		password := form["password"]
		token := form["token"]

		err = a.Auth.SignupUser(userName, password, token)
		if err != nil {
			log.Printf("Couldn't create user : %s", err)
			http.Error(w, "Internal server error", http.StatusUnauthorized)
			return
		}
		log.Printf("Created used %s", userName)
		msg := cli18n.S("Created user %s", userName) + "\n"
		fmt.Fprint(w, msg)
		return

	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	switch r.Method {
	case "GET":
		err := templates.ExecuteTemplate(w, "logout.html", TemplateData{Loc: cli18n})
		if err != nil {
			log.Printf("Couldn't execute template : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	case "POST":
		userName, err := a.Auth.Logout(w, r)
		if err != nil {
			log.Printf("Couldn't logout : %s", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		log.Printf("User %s logged out", userName)
		msg := cli18n.S("Logged out user %s", userName) + "\n"
		fmt.Fprint(w, msg)
	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	msg := cli18n.S("Users") + "\n"
	fmt.Fprint(w, msg)
	for _, uName := range a.Auth.ListUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}
