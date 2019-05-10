package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func helloWorld(w http.ResponseWriter, r *http.Request) {
	var msg string
	if isLoggedIn(r) {
		msg = fmt.Sprintf("Hello, you are logged in as user %s!", getLoggedInUserName(r))
	} else {
		msg = "Hello, you are not logged in."
	}
	fmt.Fprintf(w, "%s\n", msg)
}

func message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		msg = strings.Replace(msg, "${username}", getLoggedInUserName(r), -1)
		fmt.Fprintf(w, "%s\n", msg)
	}
}

func httpError(httpStatusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(httpStatusCode), httpStatusCode)
	}
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Users\n")
	for _, uName := range userDB.GetUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %v", r)
		next.ServeHTTP(w, r)
	})
}
