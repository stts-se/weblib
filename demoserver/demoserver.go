// A simple demo server, using the available libraries in this repository. It is also meant to show some design patterns, not suitable for library implementation, but still useful as templates for a server implementation.
package main

import (
	flaglib "flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/mux"

	"github.com/stts-se/weblib/auth"
	"github.com/stts-se/weblib/i18n"
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
	var err error
	err = s.auth.SaveUserDB()
	if err != nil {
		return fmt.Errorf("couldn't save user db : %v", err)
	}
	err = s.auth.SaveRoleDB()
	if err != nil {
		return fmt.Errorf("couldn't save user db : %v", err)
	}
	err = i18nCache.Close()
	if err != nil {
		log.Printf("Couldn't close i18n cache : %v", err)
	}
	return nil
}

type pair struct {
	v1 string
	v2 interface{}
}

var appInfo = []pair{
	{"App name", "demoserver"},
	{"Version", "0.1"},
	{"Release date", "unknown"},
	{"Build timestamp", time.Now().Format("2006-01-02 15:04:05 MST")},
}

const i18nDir = "i18n"

var i18nCache *i18n.I18NDB

func main() {

	rand.Seed(time.Now().UnixNano())

	var err error

	// FLAGS
	cmdName := os.Args[0]
	flags := flaglib.NewFlagSet(cmdName, flaglib.ExitOnError)
	host := flags.String("host", "127.0.0.1", "server host")
	port := flags.Int("port", 7932, "server port")
	serverKeyFile := flags.String("key", "server_config/serverkey", "server key file for session cookies")
	userDBFile := flags.String("u", "", "user database (required)")
	roleDBFile := flags.String("r", "", "role database (required)")
	logI18NToTemplate := flags.Bool("t", false, "generate a template from all strings processed by i18n (template file is generated on server shutdown)")
	help := flags.Bool("h", false, "print usage and exit")

	// go run /usr/local/go/src/crypto/tls/generate_cert.go
	tlsCert := flags.String("tlsCert", "", "server_config/cert.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)")
	tlsKey := flags.String("tlsKey", "", "server_config/key.pem (generate with golang's crypto/tls/generate_cert.go) (default disabled)")

	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <options>\n", cmdName)
		flags.PrintDefaults()
	}

	flags.Parse(os.Args[1:])
	args := flags.Args()
	if *help {
		flags.Usage()
		os.Exit(1)
	}

	requiredFlags := map[string]string{"u": "user database", "r": "role database"}
	// cache what flags have been used
	usedFlags := make(map[string]bool)
	flags.Visit(func(f *flaglib.Flag) { usedFlags[f.Name] = true })

	// check for missing, required flags
	missingRequiredFlags := false
	for req, desc := range requiredFlags {
		if !usedFlags[req] {
			fmt.Fprintf(os.Stderr, "missing required flag -%s %s\n", req, desc)
			missingRequiredFlags = true
		}
	}
	if missingRequiredFlags {
		flags.Usage()
		os.Exit(2) // the same exit code used by flag.Parse
	}

	// check remaining args (should be empty)
	if len(args) != 0 {
		flags.Usage()
		os.Exit(1)
	}

	i18n.LogToTemplate = *logI18NToTemplate
	err = initI18NPropFiles(i18nDir)
	if err != nil {
		log.Fatalf("Couldn't read i18n properties : %v", err)
	}

	tlsEnabled := false
	protocol := "http"
	if *tlsCert != "" && *tlsKey != "" {
		tlsEnabled = true
	}
	if !tlsEnabled && (*tlsCert != "" || *tlsKey != "") {
		flags.Usage()
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

	localeR := r.PathPrefix("/locale").Subrouter()
	localeR.HandleFunc("/list", listLocales)
	localeR.HandleFunc("/translate/{input}", translate)

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

	//signal.Notify(stop, os.Interrupt)
	signal.Notify(stop) // will exit nicely on Ctrl-C and kill signals
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

	// This happens after Ctrl-C/kill signals
	fmt.Fprintf(os.Stderr, "\n")
	log.Printf("Received stop signal")

	err = srv.close()
	if err != nil {
		log.Fatalf("Server stopped with an error on close : %v", err)
	}
	log.Println("Server stopped")
}
