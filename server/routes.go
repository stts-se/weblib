package main

import (
	"fmt"
	"net/http"
)

func protected(w http.ResponseWriter, r *http.Request) {
	// Print secret message
	fmt.Fprintln(w, "Your identity is verified, and you have access to all the secret stuff!")
}

func httpError(httpErrorCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, httpErrorCode)
		return
	}
}

func pageNotFound() http.HandlerFunc {
	return httpError(http.StatusNotFound, "404 page not found")
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, world!")
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Users")
	for _, uName := range userDB.GetUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}
