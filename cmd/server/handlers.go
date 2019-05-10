package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
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

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %v", r)
		next.ServeHTTP(w, r)
	})
}

func simpleDoc(router *mux.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		walkedURLs := []string{}
		printedURLs := make(map[string]bool)
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			u, err := route.GetPathTemplate()
			if err != nil {
				return err
			}
			if len(u) > 1 {
				u = strings.TrimSuffix(u, "/")
			}
			doc := u
			// if info, ok := docs[t]; ok {
			// doc = fmt.Sprintf("%s - %s", u, info)
			// }
			if _, printed := printedURLs[u]; !printed {
				printedURLs[u] = true
				walkedURLs = append(walkedURLs, doc)
			}
			return nil
		})

		url := strings.Join(walkedURLs, "\n")
		fmt.Fprintf(w, "%s\n", url)
	}
}
