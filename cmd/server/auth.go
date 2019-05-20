package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/stts-se/weblib"
	"github.com/stts-se/weblib/auth"
	"github.com/stts-se/weblib/i18n"
)

type authHandlers struct {
	Auth *auth.Auth
}

func (a *authHandlers) helloWorld(w http.ResponseWriter, r *http.Request) {
	var msg string
	if ok, userName := a.Auth.IsLoggedIn(r); ok {
		msg = fmt.Sprintf("Hello, you are logged in as user %s!", userName)
	} else {
		msg = "Hello, you are not logged in."
	}
	fmt.Fprintf(w, "%s\n", msg)
}

func (a *authHandlers) message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if ok, uName := a.Auth.IsLoggedIn(r); ok {
			msg = strings.Replace(msg, "${username}", uName, -1)
		}
		fmt.Fprintf(w, "%s\n", msg)
	}
}

func (a *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	cli18n := getLocaleFromRequest(r)
	switch r.Method {
	case "GET":
		err := templates.ExecuteTemplate(w, "login.html", TemplateData{Loc: cli18n})
		if err != nil {
			log.Printf("Couldn't execute template : %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
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
		fmt.Fprintf(w, cli18n.S("Logged in successfully as user %s")+"\n", userName)
		return

	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) invite(w http.ResponseWriter, r *http.Request) {
	cli18n := getLocaleFromRequest(r)
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

		link := fmt.Sprintf("%s/auth/signup?token=%s", weblib.GetServerURL(r), url.PathEscape(token))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		log.Printf("Created invitation link: %s", link)
		fmt.Fprintf(w, fmt.Sprintf("%s: <a href='%s'>%s</a>\n", cli18n.S("Invitation link"), link, link))
	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) signup(w http.ResponseWriter, r *http.Request) {
	cli18n := getLocaleFromRequest(r)
	switch r.Method {
	case "GET":
		token, err := url.PathUnescape(weblib.GetParam("token", r))
		if err != nil {
			log.Printf("Couldn't unescape token : %v", err)
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
		fmt.Fprintf(w, cli18n.S("Created user %s")+"\n", userName)
		return

	default:
		http.NotFound(w, r)
	}
}

const stripLocaleRegion = true

func getLocaleFromRequest(r *http.Request) *i18n.I18N {
	locName := weblib.GetParam("locale", r)
	if locName == "" {
		cookie, err := r.Cookie("locale")
		log.Printf("Locale cookie from request: %#v", cookie)
		if err == nil {
			locName = cookie.Value
		}
	}
	if locName == "" {
		acceptLangs := r.Header["Accept-Language"]
		if len(acceptLangs) > 0 {
			locName = strings.Split(acceptLangs[0], ",")[0]
		}
	}
	log.Printf("Locale from request: %s", locName)
	if locName != "" {
		if stripLocaleRegion {
			locName = strings.Split(locName, "-")[0]
		}
		return i18n.GetOrCreate(locName)
	}
	return i18n.Default()
}

func (a *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	cli18n := getLocaleFromRequest(r)
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

		log.Printf("User %s logged out successfully", userName)
		fmt.Fprintf(w, cli18n.S("Logged out user %s successfully")+"\n", userName)
	default:
		http.NotFound(w, r)
	}
}

func (a *authHandlers) listUsers(w http.ResponseWriter, r *http.Request) {
	cli18n := getLocaleFromRequest(r)
	fmt.Fprintf(w, cli18n.S("Users")+"\n")
	for _, uName := range a.Auth.ListUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}
