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

// UTILITIES

func getParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

func parseForm(r *http.Request, requiredParams []string) (map[string]string, bool) {
	res := make(map[string]string)
	missing := []string{}
	if err := r.ParseForm(); err != nil {
		log.Printf("Couldn't parse form : %v", err)
		return res, false
	}
	for _, param := range requiredParams {
		value := r.FormValue(param)
		if value != "" {
			res[param] = value
		} else {
			missing = append(missing, param)
		}
	}
	if len(missing) > 0 {
		pluralS := "s"
		if len(missing) == 1 {
			pluralS = ""
		}
		log.Printf("Couldn't parse form : missing param%s : %v", pluralS, missing)
	}
	return res, true
}
