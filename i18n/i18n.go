package i18n

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stts-se/weblib"
)

// see also https://blog.golang.org/matchlang

// I18N a key-value dictionary container for a certain locale
type I18N map[string]string

// S is used to look up the localized version of the param. It will also fill in the values using fmt.Sprintf.
func (i *I18N) S(param string, values ...string) string {
	res := param
	if r, ok := (*i)[param]; ok {
		res = r
	}
	if len(values) == 0 {
		return res
	}
	return fmt.Sprintf(res, values)
}

// I18Ns the cached localisation dictionaries
var I18Ns = make(map[string]*I18N)

// DefaultLocale a default locale (string) for when it's not set by the user
const DefaultLocale = "en"
const i18nDir = "i18n"
const i18nExtension = ".properties"

// Default I18N instance for DefaultLocale
func Default() *I18N {
	return GetOrCreate(DefaultLocale)
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
	return sortedKeys(I18Ns)
}

// GetOrCreate get the I18N instance for the locale, create if necessary
func GetOrCreate(locale string) *I18N {
	if _, ok := I18Ns[locale]; !ok {
		loc := I18N(make(map[string]string))
		I18Ns[locale] = &loc
	}
	loc, _ := I18Ns[locale]
	return loc
}

// ReadI18NPropFiles read all i18n property files in the folder i18nDir (see source code)
func ReadI18NPropFiles() error {
	files, err := ioutil.ReadDir(i18nDir)
	if err != nil {
		return fmt.Errorf("Couldn't list files in folder %s : %v", i18nDir, err)
	}
	for _, f := range files {
		loc := I18N(make(map[string]string))
		fn := f.Name()
		ext := path.Ext(path.Base(fn))
		if ext != i18nExtension {
			continue
		}
		locName := strings.TrimSuffix(fn, ext)
		lines, err := weblib.ReadLines(filepath.Join(i18nDir, f.Name()))
		if err != nil {
			return err
		}

		for _, l := range lines {
			fs := strings.Split(l, "\t")
			if len(fs) == 2 {
				loc[fs[0]] = fs[1]
			}
		}
		I18Ns[locName] = &loc
		log.Printf("Read locale %s", locName)
	}
	if _, ok := I18Ns[DefaultLocale]; !ok {
		loc := I18N(make(map[string]string))
		I18Ns[DefaultLocale] = &loc
	}
	return nil
}
