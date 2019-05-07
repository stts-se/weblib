package main

import (
	"fmt"
	"log"
	"net"
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
	runHTTPS := true
	deriveIP := true

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

	r := mux.NewRouter() // http.NewServeMux()
	r.StrictSlash(true)

	r.HandleFunc("/doc/", generateDoc)

	r.HandleFunc("/login", notLoggedIn(login))
	r.HandleFunc("/protected", requireAccessRights(protected))
	//r.HandleFunc("/protected", requireAccessRights(generateDoc))
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

	serverIP := "127.0.0.1"
	if deriveIP {
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			log.Fatalf("%v", err)
		}
		for _, addr := range addrs {
			ip := strings.Split(addr.String(), "/")[0]
			if strings.HasPrefix(ip, "192.168") {
				serverIP = ip
			}
		}
	}

	protocol := "http"
	if runHTTPS {
		protocol = "https"
	}
	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("%s:%s", serverIP, port),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("Starting server on %s://%s", protocol, srv.Addr)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)
	go func() {
		if runHTTPS {
			// go run /usr/local/go/src/crypto/tls/generate_cert.go
			err = srv.ListenAndServeTLS("server_config/cert.pem", "server_config/key.pem")
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
