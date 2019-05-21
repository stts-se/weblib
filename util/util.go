package util

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gorilla/mux"
)

// GetParam retrieves the value of the specified parameter (looking for URL parameters and mux vars). If it's not defined, GetParam returns the empty string.
func GetParam(r *http.Request, paramName string) string {
	res := r.FormValue(paramName)
	if res != "" {
		return res
	}
	vars := mux.Vars(r)
	return vars[paramName]
}

// ParseForm is used to parse a request form, and insert required params to a key-value map. An error is returned if the call to r.ParseForms generates an error, or if any of the required parameters are unset.
func ParseForm(r *http.Request, requiredParams []string) (map[string]string, error) {
	res := make(map[string]string)
	missing := []string{}
	if err := r.ParseForm(); err != nil {
		return res, fmt.Errorf("couldn't parse form : %v", err)
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
		return res, fmt.Errorf("missing param%s : %v", pluralS, missing)
	}
	return res, nil
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

//ReadLines read a file into a slice of lines
func ReadLines(fileName string) ([]string, error) {
	var res []string
	var scanner *bufio.Scanner
	fh, err := os.Open(fileName)
	if err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fileName, err)
	}

	if strings.HasSuffix(fileName, ".gz") {
		gz, err := gzip.NewReader(fh)
		if err != nil {
			return res, fmt.Errorf("failed to read '%s' : %v", fileName, err)
		}
		scanner = bufio.NewScanner(gz)
	} else {
		scanner = bufio.NewScanner(fh)
	}
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fileName, err)
	}
	return res, nil
}

//ReadFile read a file into a single string
func ReadFile(fileName string) (string, error) {
	lines, err := ReadLines(fileName)
	if err != nil {
		return "", err
	}
	return strings.Join(lines, "\n"), nil
}

//FileExists returns true if the file exists on disk (without checking what type it is)
func FileExists(fileName string) bool {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false
	}
	return true
}
