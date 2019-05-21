// Package i18n contains locale/i18n utilities for web applications
package i18n

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/stts-se/weblib/util"
)

// I18N a key-value dictionary container for a certain locale
type I18N map[string]string

// S is used to look up the localized version of the input string (s). It will also fill in the arguments (args) using fmt.Sprintf.
func (i *I18N) S(s string, args ...interface{}) string {
	//log.Printf("I18N.S debug\t%s\t%#v\t%v\t%s", s, args, len(args), reflect.TypeOf(args))
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
	log.Printf("No i18n for locale %s, using default locale %s", locale, DefaultLocale)
	return Default()
}

// ReadI18NPropFiles read and cache all i18n property files in the specified dir
func ReadI18NPropFiles(dir string) error {
	res := make(map[string]*I18N)

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
			fs := strings.Split(l, "\t")
			if len(fs) == 2 {
				loc[fs[0]] = fs[1]
			}
		}
		res[locName] = &loc
		log.Printf("Read locale %s from file %s", locName, fPath)
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

const stripLocaleRegion = true

// GetLocaleFromRequest retrieve locale from http.Request (reads (1) URL params, (2) cookies, (3) request header)
func GetLocaleFromRequest(r *http.Request) *I18N {
	locName := util.GetParam(r, "locale")
	if locName == "" {
		cookie, err := r.Cookie("locale")
		log.Printf("Locale cookie from request: %#v", cookie)
		if err == nil {
			locName = cookie.Value
		}
	}
	if locName == "" {
		acceptLangs := r.Header["Accept-Language"]
		if len(acceptLangs) > 0 {
			locName = strings.Split(acceptLangs[0], ",")[0]
		}
	}
	log.Printf("Requested locale: %s", locName)
	if locName != "" {
		if stripLocaleRegion {
			locName = strings.Split(locName, "-")[0]
		}
		return GetOrDefault(locName)
	}
	return Default()
}
