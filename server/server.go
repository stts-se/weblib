package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"

	"github.com/stts-se/weblib/userdb"
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

var userDB userdb.UserDB

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

	r.HandleFunc("/login", notLoggedIn(login))
	r.HandleFunc("/login", login)
	r.HandleFunc("/protected", requireAccessRights(protected))
	r.HandleFunc("/list_users", requireAccessRights(listUsers))
	r.HandleFunc("/logout", requireAccessRights(logout))

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
