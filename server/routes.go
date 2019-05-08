package main

import (
	"fmt"
	"net/http"
)

func message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s\n", msg)
	}
}

func httpError(httpStatusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(httpStatusCode), httpStatusCode)
	}
}

func pageNotFound() http.HandlerFunc {
	return httpError(http.StatusNotFound)
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Users\n")
	for _, uName := range userDB.GetUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}
}
