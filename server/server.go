package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"reflect"
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

var cookieStore *sessions.CookieStore

func getParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

func protected(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "auth-user-session")

	// Check if user is authenticated
	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Error(w, "Not logged in", http.StatusForbidden)
		return
	}

	// Print secret message
	fmt.Fprintln(w, "The cake is a lie and you are logged in!")
}

func login(w http.ResponseWriter, r *http.Request) {
	session, _ := cookieStore.Get(r, "auth-user-session")

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
	session, _ := cookieStore.Get(r, "auth-user-session")

	// Revoke users authentication
	session.Values["authenticated"] = false
	session.Save(r, w)
}

var userDB userdb.UserDB

func fileExists(fileName string) bool {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false
	}
	return true
}

func initCookieStore(keyFile string) (*sessions.CookieStore, error) {
	var cs *sessions.CookieStore
	var key []byte
	var err error
	if !fileExists(keyFile) {
		rand.Seed(time.Now().UnixNano())
		// Note: Don't store your key in your source code. Pass it via an
		// environmental variable, or flag (or both), and don't accidentally commit it
		// alongside your code. Ensure your key is sufficiently random - i.e. use Go's
		// crypto/rand or securecookie.GenerateRandomKey(32) and persist the result.
		//var cookieStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

		// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)

		fmt.Printf("No server key defined. Create new server key? (Ctrl-c to exit) [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		var r string
		r, err = reader.ReadString('\n')
		if err != nil {
			return cs, err
		}
		r = strings.ToLower(strings.TrimSpace(r))
		if len(r) > 0 && !strings.HasPrefix(r, "y") {
			fmt.Fprintf(os.Stderr, "BYE!\n")
			os.Exit(0)
		}
		key = make([]byte, 32)
		_, err = rand.Read(key)
		if err != nil {
			return cs, err
		}
		err = ioutil.WriteFile(keyFile, key, 0644)
		if err != nil {
			return cs, err
		}
		keyCheck, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return cs, fmt.Errorf("couldn't re-read key file")
		}
		if !reflect.DeepEqual(key, keyCheck) {
			return cs, fmt.Errorf("session key mismatch")
		}
		log.Printf("New key saved to file %s", keyFile)

	} else {
		key, err = ioutil.ReadFile(keyFile)
		if err != nil {
			return cs, err
		}
		if len(key) != 32 {
			return cs, fmt.Errorf("Invalid key length: %d", len(key))
		}
	}
	cs = sessions.NewCookieStore([]byte(key))
	return cs, nil
}

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
		fmt.Println("Empty user db. Create new user? (Ctrl-c to exit)")
		for {
			fmt.Printf("Username: ")
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
			fmt.Printf("Create another user? [Y/n] ")
			r, err := reader.ReadString('\n')
			if err != nil {
				return userDB, err
			}
			r = strings.ToLower(strings.TrimSpace(r))
			if len(r) > 0 && !strings.HasPrefix(r, "y") {
				break
			}
		}
	}
	return userDB, nil
}

func main() {
	var err error

	args := os.Args[1:]
	if len(args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: server <port> <serverkeyfile> <userdb>\n")
		os.Exit(0)
	}
	port := args[0]

	cookieStore, err = initCookieStore(args[1])
	if err != nil {
		log.Fatalf("%v", err)
	}

	userDB, err = initDB(args[2])
	if err != nil {
		log.Fatalf("%v", err)
	}

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
		Handler: r,
		// Addr:         "127.0.0.1:" + port,
		Addr:         ":" + port,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Server started on localhost:%s", port)

	log.Fatal(srv.ListenAndServe(), nil)
	// ListenAndServeTLS(addr, certFile, keyFile string, handler Handler) error

	fmt.Println("No fun")
}
