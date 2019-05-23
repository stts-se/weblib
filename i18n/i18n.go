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
type set map[string]bool

// I18N a key-value dictionary container for a certain locale
type I18N struct {
	dict   dict
	locale string
}

// S is used to look up the localized version of the input string (s). It will also fill in the arguments (args) using fmt.Sprintf. If LogToTemplate is set to true, any unknown translations will be logged to a template file.
func (i *I18N) S(s string, args ...interface{}) string {
	if LogToTemplate {
		templateLog.mutex.RLock()
		defer templateLog.mutex.RUnlock()
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

// newI18N returns a new (empty) I18N dictionary for the specified locale
func newI18N(locale string) *I18N {
	return &I18N{dict: make(map[string]string), locale: locale}
}

type templateLogger struct {
	mutex *sync.RWMutex
	data  map[string]set
}

var templateLog = templateLogger{
	mutex: &sync.RWMutex{},
	data:  make(map[string]set),
}

// I18NDB a mutexed database of I18N instances
type I18NDB struct {
	mutex         *sync.RWMutex
	data          map[string]*I18N
	DefaultLocale string
	Dir           string
}

func newI18NDB(dir, defaultLocale string) *I18NDB {
	return &I18NDB{
		mutex:         &sync.RWMutex{},
		data:          make(map[string]*I18N),
		DefaultLocale: defaultLocale,
		Dir:           dir,
	}
}

const i18nExtension = ".properties"

// Default I18N instance (used when no locale is provided by the user/client)
func (db *I18NDB) Default() *I18N {
	return db.GetOrDefault(db.DefaultLocale)
}

func sortedKeysString2I18N(m map[string]*I18N) []string {
	res := []string{}
	for k := range m {
		res = append(res, k)
	}

	sort.Slice(res, func(i, j int) bool { return res[i] < res[j] })
	return res
}

// ListLocales list all locale (names) in the db
func (db *I18NDB) ListLocales() []string {
	return sortedKeysString2I18N(db.data)
}

// GetOrDefault returns the I18N instance for the locale. If it doesn't exist, the default I18N will be returned.
func (db *I18NDB) GetOrDefault(locale string) *I18N {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if loc, ok := db.data[locale]; ok {
		return loc
	}
	log.Printf("No i18n defined for locale %s, using default locale %s", locale, db.DefaultLocale)
	return db.Default()
}

// GetOrCreate returns the I18N instance for the locale. If it doesn't exist, a new, empty locale dictionary will be created (but not saved to cache)
func (db *I18NDB) GetOrCreate(locale string) *I18N {
	db.mutex.Lock()
	defer db.mutex.Unlock()
	if loc, ok := db.data[locale]; ok {
		return loc
	}
	log.Printf("No i18n defined for locale %s, creating a new instance on the fly", locale)
	return newI18N(locale)
}

func readI18NPropFiles(dir string) (map[string]*I18N, error) {
	res := make(map[string]*I18N)

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return res, fmt.Errorf("couldn't list files in folder %s : %v", dir, err)
	}
	for _, f := range files {
		fn := f.Name()
		fPath := filepath.Join(dir, f.Name())
		ext := path.Ext(path.Base(fn))
		if ext != i18nExtension {
			continue
		}
		locName := strings.TrimSuffix(fn, ext)
		loc, err := readI18NPropFile(locName, fPath)
		if err != nil {
			return res, err
		}
		res[locName] = loc
	}
	return res, nil
}

func readI18NPropFile(locName, fName string) (*I18N, error) {
	res := newI18N(locName)
	lines, err := util.ReadLines(fName)
	if err != nil {
		return res, err
	}
	for _, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		fs := strings.Split(l, "\t")
		if len(fs) == 2 {
			res.dict[fs[0]] = fs[1]
		}
	}
	log.Printf("Read locale %s from %s", locName, fName)
	return res, nil
}

// ReadI18NPropFiles read all i18n property files in the specified folder
func ReadI18NPropFiles(dir, defaultLocale string) (*I18NDB, error) {
	res := newI18NDB(dir, defaultLocale)

	i18ns, err := readI18NPropFiles(dir)
	if err != nil {
		return res, err
	}

	res.mutex.RLock()
	defer res.mutex.RUnlock()
	res.data = i18ns

	return res, nil
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

// GetI18NFromRequest will lookup the requested locale, and return the corresponding I18N instance. If the requested locale doesn't exist, the default locale will be returned instead.
func (db *I18NDB) GetI18NFromRequest(r *http.Request) *I18N {
	locName, _ := GetLocaleFromRequest(r)
	if locName != "" {
		if StripLocaleRegion {
			locName = strings.Split(locName, "-")[0]
		}
		if LogToTemplate {
			return db.GetOrCreate(locName)
		}
		return db.GetOrDefault(locName)
	}
	return db.Default()
}

// Close i18n nicely. If LogToTemplate is enabled, and the saveDir is non-empty, a template file (template.properties) will be created.
func (db *I18NDB) Close() error {
	if LogToTemplate {
		templateLog.mutex.Lock()
		defer templateLog.mutex.Unlock()

		if len(templateLog.data) == 0 {
			return nil
		}

		if db.Dir == "" {
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
			templateFileName := path.Join(db.Dir, fmt.Sprintf("%s_template.log", locale))
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

// CrossValidate will return true if the files are validated without errors. The second return value is a slice of error messages, if any.
func (db *I18NDB) CrossValidate() ([]string, error) {

	res := []string{}

	// 1. Compare loaded I18Ns with pre-cached translation maps (order not preserved)
	if len(db.data) == 0 {
		return res, fmt.Errorf("I18N data cache is empty. You need to run ReadI18NPropFile before validating.")
	}
	if len(db.data) == 1 {
		return res, nil
	}

	locs := sortedKeysString2I18N(db.data)
	ref := db.data[locs[0]]
	refLoc := ref.locale
	for _, loc := range locs[1:] {
		this := db.data[loc]
		thisLoc := this.locale

		if rL, tL := len(ref.dict), len(this.dict); rL != tL {
			res = append(res, fmt.Sprintf("mismatching number of items; %s:%d vs. %s:%d", refLoc, rL, thisLoc, tL))
		}

		for refKey := range ref.dict {
			if _, ok := this.dict[refKey]; !ok {
				res = append(res, fmt.Sprintf("key in %s is not present in %s\t%s", refLoc, thisLoc, refKey))
			}
		}
		for thisKey := range this.dict {
			if _, ok := ref.dict[thisKey]; !ok {
				res = append(res, fmt.Sprintf("key in %s is not present in %s\t%s", thisLoc, refLoc, thisKey))
			}
		}

	}

	// 2. Load all keys and compare as list (to keep original order in the file)

	// list all i18n property files
	files, err := ioutil.ReadDir(db.Dir)
	if err != nil {
		return res, fmt.Errorf("couldn't list files in folder %s : %v", db.Dir, err)
	}

	// read all translation keys
	allKeys := make(map[string][]string)
	allLocs := []string{}
	for _, f := range files {
		fn := f.Name()
		fPath := filepath.Join(db.Dir, f.Name())
		ext := path.Ext(path.Base(fn))
		if ext != i18nExtension {
			continue
		}
		locName := strings.TrimSuffix(fn, ext)
		lines, err := util.ReadLines(fPath)
		if err != nil {
			return res, err
		}

		keys := []string{}
		for _, l := range lines {
			if strings.HasPrefix(strings.TrimSpace(l), "#") {
				continue
			}
			fs := strings.Split(l, "\t")
			if len(fs) == 2 {
				keys = append(keys, fs[0])
			}
		}
		allKeys[locName] = keys
		allLocs = append(allLocs, locName)
	}

	// compare all translation keys
	if len(allLocs) == 0 {
		return res, fmt.Errorf("no i18n prop files in folder %s", db.Dir)
	}

	refLoc = allLocs[0]
	refKeys := allKeys[refLoc]
	for _, thisLoc := range allLocs[1:] {
		thisKeys := allKeys[thisLoc]
		if rL, tL := len(refKeys), len(thisKeys); rL != tL {
			res = append(res, fmt.Sprintf("mismatching number of items; %s:%d vs. %s:%d", refLoc, rL, thisLoc, tL))
		}

		for i, refKey := range refKeys {
			if i >= len(thisKeys) {
				res = append(res, fmt.Sprintf("key no. %d in %s is not present in %s\t%s", (i+1), refLoc, thisLoc, refKey))
				continue
			}
			thisKey := thisKeys[i]
			if thisKey != refKey {
				res = append(res, fmt.Sprintf("mismatching key for line %d (%s vs. %s)\t%s\t%s", (i+1), refLoc, thisLoc, refKey, thisKey))
			}
		}
		if len(thisKeys) > len(refKeys) {
			for i, thisKey := range thisKeys {
				if i >= len(refKeys) {
					res = append(res, fmt.Sprintf("key no. %d in %s is not present in %s\t%s", (i+1), thisLoc, refLoc, thisKey))
				}
			}
		}
	}

	// Finally: clean out duplicates
	var contains = func(slice []string, s string) bool {
		for _, s0 := range slice {
			if s0 == s {
				return true
			}
		}
		return false
	}

	resUniq := []string{}
	for _, msg := range res {
		if !contains(resUniq, msg) {
			resUniq = append(resUniq, msg)
		}
	}

	log.Printf("Cross validation completed for locales: %v", allLocs)
	return resUniq, nil
}
