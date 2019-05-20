package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"

	"github.com/stts-se/weblib/auth"
)

// Server container
type Server struct {
	httpServer *http.Server
	auth       *auth.Auth
	protocol   string
	tlsEnabled bool
}

func (s *Server) url() string {
	return fmt.Sprintf("%s://%s", s.protocol, s.httpServer.Addr)
}

func (s *Server) close() error {
	err := s.auth.SaveUserDB()
	if err != nil {
		return fmt.Errorf("couldn't save user db : %v", err)
	}
	return nil
}

var appInfo = map[string]string{
	"AppName":         "demo server",
	"Version":         "0.1",
	"Build timestamp": "unknown",
}

func main() {

	rand.Seed(time.Now().UnixNano())

	var err error

	// OPTIONS
	host := flag.String("host", "127.0.0.1", "server host")
	port := flag.Int("port", 7932, "server port")
	serverKeyFile := flag.String("key", "server_config/serverkey", "server key file for session cookies")
	userDBFile := flag.String("u", "", "user database")
	roleDBFile := flag.String("r", "", "role database")
	help := flag.Bool("h", false, "print usage and exit")

	// go run /usr/local/go/src/crypto/tls/generate_cert.go
	tlsCert := flag.String("tlsCert", "", "server_config/cert.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)")
	tlsKey := flag.String("tlsKey", "", "server_config/key.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)")

	// parse check for missing required flags
	required := map[string]string{"u": "user database", "r": "role database"}
	flag.Parse()
	seen := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) { seen[f.Name] = true })
	for req, desc := range required {
		if !seen[req] {
			// or possibly use `log.Fatalf` instead of:
			fmt.Fprintf(os.Stderr, "missing required flag -%s %s\n", req, desc)
			flag.PrintDefaults()
			os.Exit(2) // the same exit code flag.Parse uses
		}
	}

	args := flag.Args()
	if len(args) != 0 {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *help {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	tlsEnabled := false
	protocol := "http"
	if *tlsCert != "" && *tlsKey != "" {
		tlsEnabled = true
	}
	if !tlsEnabled && (*tlsCert != "" || *tlsKey != "") {
		fmt.Fprintf(os.Stderr, "Usage: server <options>\n")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if tlsEnabled {
		protocol = "https"
	}
	address := fmt.Sprintf("%s:%v", *host, *port)

	cookieStore, err := initCookieStore(*serverKeyFile)
	if err != nil {
		log.Fatalf("Cookie store init failed : %v", err)
	}

	userDB, err := initUserDB(*userDBFile)
	roleDB, err := initRoleDB(*roleDBFile)

	if err != nil {
		log.Fatalf("UserDB init failed : %v", err)
	}

	auth, err := auth.NewAuth("auth-user-weblib", userDB, roleDB, cookieStore)
	if err != nil {
		log.Fatalf("Auth init failed : %v", err)
	}
	authHandlers := authHandlers{Auth: auth}

	r := mux.NewRouter()
	r.StrictSlash(true)
	r.Use(logging)

	r.HandleFunc("/", authHandlers.helloWorld)
	r.HandleFunc("/doc/", simpleDoc(r, make(map[string]string)))
	r.HandleFunc("/about/", about)

	authR := r.PathPrefix("/auth").Subrouter()
	authR.HandleFunc("/", authHandlers.message("User authorization"))
	authR.HandleFunc("/login", auth.ServeAuthUserOrElse(authHandlers.message("You are already logged in as user ${username}"), authHandlers.login))
	authR.HandleFunc("/logout", auth.ServeAuthUser(authHandlers.logout))
	authR.HandleFunc("/signup", authHandlers.signup)

	protectedR := r.PathPrefix("/protected").Subrouter()
	auth.RequireAuthUser(protectedR)
	protectedR.HandleFunc("/", authHandlers.message("Protected area (open to all logged-in users)"))

	adminR := r.PathPrefix("/admin").Subrouter()
	auth.RequireAuthRole(adminR, "admin")
	adminR.HandleFunc("/", authHandlers.message("Admin area (open for admin users)"))
	adminR.HandleFunc("/invite", authHandlers.invite)
	adminR.HandleFunc("/list_users", authHandlers.listUsers)

	r.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("static/"))))

	httpSrv := &http.Server{
		Handler:      r,
		Addr:         address,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	srv := Server{
		auth:       auth,
		protocol:   protocol,
		tlsEnabled: tlsEnabled,
		httpServer: httpSrv,
	}

	log.Printf("Getting ready to start server on %s://%s", srv.protocol, httpSrv.Addr)

	stop := make(chan os.Signal, 1)

	signal.Notify(stop, os.Interrupt)
	go func() {
		if tlsEnabled {
			err = httpSrv.ListenAndServeTLS(*tlsCert, *tlsKey)
		} else {
			err = httpSrv.ListenAndServe()
		}
		if err != nil {
			log.Fatalf("Couldn't start server on port %v : %v", port, err)
		}
	}()
	log.Printf("Server up and running on %s://%s", srv.protocol, httpSrv.Addr)

	<-stop

	// This happens after Ctrl-C
	fmt.Fprintf(os.Stderr, "\n")
	err = srv.close()
	if err != nil {
		log.Fatalf("Server stopped with an error on close : %v", err)
	}
	log.Println("Server stopped")
}
