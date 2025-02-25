package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/gorilla/mux"

	"github.com/stts-se/weblib/i18n"
	"github.com/stts-se/weblib/util"
)

func message(msg string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cli18n := i18nCache.GetI18NFromRequest(r)
		fmt.Fprintf(w, cli18n.S("%s")+"\n", msg)
	}
}

func httpError(httpStatusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, http.StatusText(httpStatusCode), httpStatusCode)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %#v", r)
		log.Printf("Request URL: %s", util.GetRequestURL(r))
		loc, locSource := i18n.GetLocaleFromRequest(r)
		log.Printf("Requested locale: %s (from %s)", loc, locSource)
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
	for _, pair := range appInfo {
		lines = append(lines, fmt.Sprintf("<tr><td>%s:</td><td>%v</td></tr>", pair.v1, pair.v2))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, "<table>%s</table>\n", strings.Join(lines, "\n"))
}

func listLocales(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	msg := cli18n.S("Locales") + "\n"
	fmt.Fprint(w, msg)
	for _, loc := range i18nCache.ListLocales() {
		if loc == i18nCache.DefaultLocale {
			fmt.Fprintf(w, "- %s (default)\n", loc)
		} else {
			fmt.Fprintf(w, "- %s\n", loc)
		}
	}
}

func translate(w http.ResponseWriter, r *http.Request) {
	cli18n := i18nCache.GetI18NFromRequest(r)
	//input, err := url.PathUnescape(util.GetParam(r, "input"))
	// if err != nil {
	// 	log.Printf("Couldn't unescape param input : %v", err)
	// 	http.Error(w, "Internal server error", http.StatusInternalServerError)
	// 	return
	// }
	input := util.GetParam(r, "input")
	if input == "" {
		log.Printf("Missing param input")
		http.Error(w, "Missing param input", http.StatusPartialContent)
		return
	}
	argsParam, err := url.PathUnescape(util.GetParam(r, "args"))
	if err != nil {
		log.Printf("Couldn't unescape param args : %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	var translated string
	if argsParam == "" {
		translated = cli18n.S(input)
	} else {
		// // Flatten  variadic (moved to the I28N.S method for now)
		// args := []interface{}{}
		// for _, s := range strings.Split(argsParam, ",") {
		// 	args = append(args, s)
		// }

		args := strings.Split(argsParam, ",")
		translated = cli18n.S(input, args)
	}
	msg := strings.TrimSpace(translated) + "\n"
	fmt.Fprint(w, msg)

}
