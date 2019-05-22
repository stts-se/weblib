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

type dict map[string]string

// I18N a key-value dictionary container for a certain locale
type I18N struct {
	dict   dict
	locale string
}

// S is used to look up the localized version of the input string (s). It will also fill in the arguments (args) using fmt.Sprintf.
func (i *I18N) S(s string, args ...interface{}) string {
	if LogToTemplate {
		templateLog.mutex.Lock()
		defer templateLog.mutex.Unlock()
		if _, ok := templateLog.data[i.locale]; !ok {
			templateLog.data[i.locale] = make(map[string]bool)
		}
		templateLog.data[i.locale][s] = true
	}

	res := s
	if r, ok := (*i).dict[s]; ok {
		res = r
	} else {
		log.Printf("Missing %s localization for input string %s", i.locale, s)
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

// NewI18N returns a new (empty) I18N dictionary for the specified locale
func NewI18N(locale string) *I18N {
	return &I18N{dict: make(map[string]string), locale: locale}
}

type i18nDB struct {
	mutex *sync.RWMutex
	data  map[string]*I18N
}

var i18ns = i18nDB{
	mutex: &sync.RWMutex{},
	data:  make(map[string]*I18N),
}

type set map[string]bool

type templateLogger struct {
	mutex *sync.RWMutex
	data  map[string]set
}

var templateLog = templateLogger{
	mutex: &sync.RWMutex{},
	data:  make(map[string]set),
}

// DefaultLocale holds the name of the default locale (used when no locale is provided by the user/client)
const DefaultLocale = "en"

const i18nExtension = ".properties"

// Default I18N instance (used when no locale is provided by the user/client)
func Default() *I18N {
	return GetOrDefault(DefaultLocale)
}

func sortedKeysString2I18N(m map[string]*I18N) []string {
	res := []string{}
	for k := range m {
		res = append(res, k)
	}

	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

// ListLocales list all locale (names)
func ListLocales() []string {
	return sortedKeysString2I18N(i18ns.data)
}

// GetOrDefault returns the I18N instance for the locale. If it doesn't exist, the default I18N will be returned.
func GetOrDefault(locale string) *I18N {
	if loc, ok := i18ns.data[locale]; ok {
		return loc
	}
	log.Printf("No i18n defined for locale %s, using default locale %s", locale, DefaultLocale)
	return Default()
}

// GetOrCreate returns the I18N instance for the locale. If it doesn't exist, a new, empty locale dictionary will be created (but not saved to cache)
func GetOrCreate(locale string) *I18N {
	if loc, ok := i18ns.data[locale]; ok {
		return loc
	}
	log.Printf("No i18n defined for locale %s, creating a new instance on the fly", locale)
	return NewI18N(locale)
}

// ReadI18NPropFiles read and cache all i18n property files in the specified dir
func ReadI18NPropFiles(dir string) error {
	res := make(map[string]*I18N)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("Couldn't list files in folder %s : %v", dir, err)
	}
	for _, f := range files {
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

		loc := NewI18N(locName)
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "#") {
				continue
			}
			fs := strings.Split(l, "\t")
			if len(fs) == 2 {
				loc.dict[fs[0]] = fs[1]
			}
		}
		res[locName] = loc
		log.Printf("Read locale %s from %s", locName, fPath)
	}
	if _, ok := res[DefaultLocale]; !ok {
		loc := NewI18N(DefaultLocale)
		res[DefaultLocale] = loc
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
		if LogToTemplate {
			return GetOrCreate(locName)
		} else {
			return GetOrDefault(locName)
		}
	}
	return Default()
}

// Close i18n nicely. If LogToTemplate is enabled, a template file (template.properties) will be created.
// TODO: In the future, maybe also write cached translations to file (and append undefined translations to existing i18n files).
func Close(saveDir string) error {
	if LogToTemplate {
		templateLog.mutex.Lock()
		defer templateLog.mutex.Unlock()

		if len(templateLog.data) == 0 {
			return nil
		}

		if saveDir == "" {
			return fmt.Errorf("empty output dir")
		}
		var sortedKeysString2Set = func(m map[string]set) []string {
			res := []string{}
			for k := range m {
				res = append(res, k)
			}

			sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
			return res
		}
		var sortedKeysSet = func(m set) []string {
			res := []string{}
			for k := range m {
				res = append(res, k)
			}

			sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
			return res
		}

		for _, locale := range sortedKeysString2Set(templateLog.data) {
			utts := templateLog.data[locale]
			templateFileName := path.Join(saveDir, fmt.Sprintf("%s.template", locale))

			fh, err := os.Create(templateFileName)
			if err != nil {
				return fmt.Errorf("failed to open file : %v", err)
			}
			defer fh.Close()

			fmt.Fprintf(fh, "# i18n template for %s generated on %v\n", locale, time.Now().Format("2006-01-02 15:04:05 MST"))
			for _, from := range sortedKeysSet(utts) {
				fmt.Fprintf(fh, "%s\n", from)
			}
			log.Printf("Saved i18n template to file %s", templateFileName)
		}
	}
	return nil
}
