package main

import (
	"fmt"
	"net/http"
)

func protected(w http.ResponseWriter, r *http.Request) {
	// Print secret message
	fmt.Fprintln(w, "The cake is a lie and you are logged in!")
}

func listUsers(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Users")
	for _, uName := range userDB.GetUsers() {
		fmt.Fprintf(w, "- %s\n", uName)
	}

}
