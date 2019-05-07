package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
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

	// OPTIONS
	host := flag.String("host", "127.0.0.1", "server host")
	port := flag.String("port", "7932", "server port")
	serverKeyFile := flag.String("key", "server_config/serverkey", "server key file (for session cookies)")
	userDBFile := flag.String("db", "userdb.txt", "user database")
	help := flag.Bool("h", false, "print usage and exit")

	// go run /usr/local/go/src/crypto/tls/generate_cert.go
	tlsCert := flag.String("tlsCert", "", "server_config/cert.pem (default disabled)")
	tlsKey := flag.String("tlsKey", "", "server_config/key.pem (default disabled)")

	flag.Parse()

	args := flag.Args()
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "TLS cert/key files can be generated using go utility generate_cert.go\n")
		os.Exit(1)
	}
	if *help {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "TLS cert/key files can be generated using go utility generate_cert.go\n")
		os.Exit(1)
	}

	tlsEnabled := false
	if *tlsCert != "" && *tlsKey != "" {
		tlsEnabled = true
	}
	if !tlsEnabled && (*tlsCert != "" || *tlsKey != "") {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "TLS cert/key files can be generated using go utility generate_cert.go\n")
		os.Exit(1)
	}

	cookieStore, err = initCookieStore(*serverKeyFile)
	if err != nil {
		log.Fatalf("%v", err)
	}

	userDB, err = initDB(*userDBFile)
	if err != nil {
		log.Fatalf("%v", err)
	}

	r := mux.NewRouter() // http.NewServeMux()
	r.StrictSlash(true)

	r.HandleFunc("/doc/", generateDoc)

	r.HandleFunc("/login", notLoggedIn(login))
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

	protocol := "http"
	if tlsEnabled {
		protocol = "https"
	}
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%s", *host, *port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Starting server on %s://%s", protocol, srv.Addr)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)
	go func() {
		if tlsEnabled {
			err = srv.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			err = srv.ListenAndServe()
		}
		if err != nil {
			log.Fatalf("Couldn't start server on port %s : %v", port, err)
		}
	}()
	log.Printf("Server up and running on %s://%s", protocol, srv.Addr)

	<-stop

	// This happens after Ctrl-C
	fmt.Fprintf(os.Stderr, "\n")
	log.Println("Server stopped")
}
