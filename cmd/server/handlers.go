package main

import (
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"github.com/stts-se/weblib"
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
		log.Printf("Request URL: %s", weblib.GetRequestURL(r))
		next.ServeHTTP(w, r)
	})
}

func simpleDoc(router *mux.Router, docInfo map[string]string) http.HandlerFunc {
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
			if info, ok := docInfo[u]; ok {
				doc = fmt.Sprintf("%s - %s", u, info)
			}
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

func sortedKeys(m map[string]string) []string {
	res := []string{}
	for k := range m {
		res = append(res, k)
	}

	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

func about(w http.ResponseWriter, r *http.Request) {
	lines := []string{}
	for _, key := range sortedKeys(appInfo) {
		lines = append(lines, fmt.Sprintf("<tr><td>%s:</td><td>%s</td></tr>", key, appInfo[key]))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<table>%s</table>\n", strings.Join(lines, "\n"))
}
