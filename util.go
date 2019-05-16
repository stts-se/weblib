package weblib

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// GetParam : get request params
func GetParam(paramName string, r *http.Request) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

// ParseForm : parse input form, and add required params to a key-value map
func ParseForm(r *http.Request, requiredParams []string) (map[string]string, bool) {
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

func getProtocol(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	return "http"
}

//GetRequestURL get the full request URL from the request protocol://host:port/path
func GetRequestURL(r *http.Request) string {
	return fmt.Sprintf("%s://%s%s\n", getProtocol(r), r.Host, r.RequestURI)
}

//GetServerURL get the server URL protocol://host:port
func GetServerURL(r *http.Request) string {
	return fmt.Sprintf("%s://%s\n", getProtocol(r), r.Host)
}
