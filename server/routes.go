package main

import (
	"fmt"
	"net/http"
)

func protected(w http.ResponseWriter, r *http.Request) {
	// Print secret message
	fmt.Fprintln(w, "Your identity is verified, and you have access to all the secret stuff!")
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Users")
	for _, uName := range userDB.GetUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}

}
