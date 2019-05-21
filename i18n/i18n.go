// Package i18n contains locale/i18n utilities for web applications
package i18n

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/stts-se/weblib/util"
)

// I18N a key-value dictionary container for a certain locale
type I18N map[string]string

// S is used to look up the localized version of the input string (s). It will also fill in the arguments (args) using fmt.Sprintf.
func (i *I18N) S(s string, args ...interface{}) string {
	if LogToTemplate {
		templateLog.mutex.Lock()
		defer templateLog.mutex.Unlock()
		templateLog.data[s] = true
	}

	res := s
	if r, ok := (*i)[s]; ok {
		res = r
	}
	if len(args) == 0 {
		return res
	}

	// Flatten incorrectly organized variadic args -- an []interface{} slice with a
	// single []string slice element is probably intended as a flattened variadic
	if len(args) == 1 && reflect.TypeOf(args[0]) == reflect.TypeOf([]string{}) {
		argsI := []interface{}{}
		for _, s := range args[0].([]string) {
			argsI = append(argsI, s)
		}
		args = argsI
	}

	return fmt.Sprintf(res, args...)
}

type i18nDB struct {
	mutex *sync.RWMutex
	data  map[string]*I18N
}

var i18ns = i18nDB{
	mutex: &sync.RWMutex{},
	data:  make(map[string]*I18N),
}

var i18nDir = ""

type templateLogger struct {
	mutex *sync.RWMutex
	data  map[string]bool
}

var templateLog = templateLogger{
	mutex: &sync.RWMutex{},
	data:  make(map[string]bool),
}

// DefaultLocale holds the name of the default locale (used when no locale is provided by the user/client)
const DefaultLocale = "en"

const i18nExtension = ".properties"

// Default I18N instance (used when no locale is provided by the user/client)
func Default() *I18N {
	return GetOrDefault(DefaultLocale)
}

func sortedKeys(m map[string]*I18N) []string {
	res := []string{}
	for k := range m {
		res = append(res, k)
	}

	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

// ListLocales list all locale (names)
func ListLocales() []string {
	return sortedKeys(i18ns.data)
}

// GetOrDefault returns the I18N instance for the locale. If it doesn't exist, the default I18N will be returned.
func GetOrDefault(locale string) *I18N {
	if loc, ok := i18ns.data[locale]; ok {
		return loc
	}
	log.Printf("No i18n defined for locale %s, using default locale %s", locale, DefaultLocale)
	return Default()
}

// ReadI18NPropFiles read and cache all i18n property files in the specified dir
func ReadI18NPropFiles(dir string) error {
	res := make(map[string]*I18N)

	if i18nDir == "" {
		i18nDir = dir
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Couldn't list files in folder %s : %v", dir, err)
	}
	for _, f := range files {
		loc := I18N(make(map[string]string))
		fn := f.Name()
		fPath := filepath.Join(dir, f.Name())
		ext := path.Ext(path.Base(fn))
		if ext != i18nExtension {
			continue
		}
		locName := strings.TrimSuffix(fn, ext)
		lines, err := util.ReadLines(fPath)
		if err != nil {
			return err
		}

		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "#") {
				continue
			}
			fs := strings.Split(l, "\t")
			if len(fs) == 2 {
				loc[fs[0]] = fs[1]
			}
		}
		res[locName] = &loc
		log.Printf("Read locale %s from %s", locName, fPath)
	}
	if _, ok := res[DefaultLocale]; !ok {
		loc := I18N(make(map[string]string))
		res[DefaultLocale] = &loc
	}
	i18ns.mutex.Lock()
	defer i18ns.mutex.Unlock()
	i18ns.data = res
	return nil
}

// StripLocaleRegion set to true will ignore everything after the first dash (-) of a locale string
var StripLocaleRegion = true

// LogToTemplate  set to true to log all calls to I18N.S to a template file (template.properties)
var LogToTemplate = false

// GetLocaleFromRequest retrieves the requested locale from the http.Request. The first return value is the locale name, the second value is the source from which the locale was retrieved (param, cookie or header).
func GetLocaleFromRequest(r *http.Request) (string, string) {
	// check params
	locName := util.GetParam(r, "locale")
	if locName != "" {
		return locName, "param"
	}

	// check cookies
	cookie, err := r.Cookie("locale")
	//log.Printf("Locale cookie from request: %#v", cookie)
	if err == nil {
		return cookie.Value, "cookie"
	}

	// check header Accept-Language
	acceptLangs := r.Header["Accept-Language"]
	if len(acceptLangs) > 0 {
		return strings.Split(acceptLangs[0], ",")[0], "header"
	}
	return "", ""
}

// GetI18NFromRequest will lookup the requested locale in the cache, and return the corresponding I18N instance. If the requested locale doesn't exist, the default locale will be returned instead.
func GetI18NFromRequest(r *http.Request) *I18N {
	locName, _ := GetLocaleFromRequest(r)
	if locName != "" {
		if StripLocaleRegion {
			locName = strings.Split(locName, "-")[0]
		}
		return GetOrDefault(locName)
	}
	return Default()
}

// Close i18n nicely. If LogToTemplate is enabled, a template file (template.properties) will be created.
// TODO: In the future, maybe also write cached translations to file (and append undefined translations to existing i18n files).
func Close() error {
	if LogToTemplate {
		templateLog.mutex.Lock()
		defer templateLog.mutex.Unlock()

		if len(templateLog.data) == 0 {
			return nil
		}

		if i18nDir == "" {
			return fmt.Errorf("i18n is not initialised properly (no output dir)")
		}
		templateFileName := path.Join(i18nDir, fmt.Sprintf("template%s", i18nExtension))

		fh, err := os.Create(templateFileName)
		if err != nil {
			return fmt.Errorf("failed to open file : %v", err)
		}
		defer fh.Close()

		var sortedKeys = func(m map[string]bool) []string {
			res := []string{}
			for k := range m {
				res = append(res, k)
			}

			sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
			return res
		}

		fmt.Fprintf(fh, "# i18n template generated on %v\n", time.Now().Format("2006-01-02 15:04:05 MST"))
		for _, s := range sortedKeys(templateLog.data) {
			fmt.Fprintf(fh, "%s\n", s)
		}
		log.Printf("Saved i18n template to file %s", templateFileName)
	}
	return nil
}
