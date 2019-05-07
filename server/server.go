package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/stts-se/weblib/userdb"
	"golang.org/x/crypto/ssh/terminal"
)

// This is filled in by main, listing the URLs handled by the router,
// so that these can be shown in the generated docs.
var walkedURLs []string

func generateDoc(w http.ResponseWriter, r *http.Request) {
	s := strings.Join(walkedURLs, "\n")
	fmt.Fprintf(w, "%s\n", s)
}

// Note: Don't store your key in your source code. Pass it via an
// environmental variable, or flag (or both), and don't accidentally commit it
// alongside your code. Ensure your key is sufficiently random - i.e. use Go's
// crypto/rand or securecookie.GenerateRandomKey(32) and persist the result.
//var store = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

var (
	// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)
	key   = []byte("super-secret-key-xxxx")
	store = sessions.NewCookieStore(key)
)

func getParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

func protected(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-user-session")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Print secret message
	fmt.Fprintln(w, "The cake is a lie!")
}

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-user-session")

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // one week
		HttpOnly: true,
	}
	// Authentication goes here
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
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "auth-user-session")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
}

var userDB userdb.UserDB

func promptPassword() (string, error) {
	bytePassword, err := terminal.ReadPassword(0)
	fmt.Println()
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	return password, nil
}

func initDB(dbFile string) (userdb.UserDB, error) {
	userDB, err := userdb.ReadUserDB(dbFile)
	if err != nil {
		return userDB, err
	}
	userDB.Constraints = func(userName, password string) (bool, string) {
		if len(userName) == 0 {
			return false, "empty user name"
		}
		if len(userName) < 4 {
			return false, "username must have min 4 chars"
		}
		if len(password) == 0 {
			return false, "empty password"
		}
		if len(password) < 4 {
			return false, "password must have min 4 chars"
		}
		return true, ""
	}

	if len(userDB.GetUsers()) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Printf("Empty user db. Create new user (or Ctrl-c to exit)\nUsername: ")
		userName, err := reader.ReadString('\n')
		if err != nil {
			return userDB, err
		}

		fmt.Printf("Password: ")
		password, err := promptPassword()
		if err != nil {
			return userDB, err
		}
		fmt.Printf("Repeat password: ")
		passwordCheck, err := promptPassword()
		if err != nil {
			return userDB, err
		}
		if password != passwordCheck {
			return userDB, fmt.Errorf("Passwords do not match")
		}
		err = userDB.InsertUser(userName, password)
		if err != nil {
			return userDB, err
		}
		log.Printf("Created user %s", userName)
	}
	return userDB, nil
}

func main() {
	var err error

	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: server <port> <userdb>\n")
		os.Exit(0)
	}

	userDB, err = initDB(os.Args[2])
	if err != nil {
		log.Fatalf("%v", err)
	}

	p := os.Args[1]
	r := mux.NewRouter()
	r.StrictSlash(true)

	r.HandleFunc("/doc/", generateDoc).Methods("GET")

	r.HandleFunc("/protected", protected)
	r.HandleFunc("/login", login)
	r.HandleFunc("/logout", logout)

	// List route URLs to use as simple on-line documentation
	docs := make(map[string]string)
	r.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		t, err := route.GetPathTemplate()
		if err != nil {
			return err
		}
		if info, ok := docs[t]; ok {
			t = fmt.Sprintf("%s - %s", t, info)
		}
		walkedURLs = append(walkedURLs, t)
		return nil
	})
	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("static/"))))

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:" + p,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Server started on localhost:%s", p)

	log.Fatal(srv.ListenAndServe(), nil)
	// ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error

	fmt.Println("No fun")
}
