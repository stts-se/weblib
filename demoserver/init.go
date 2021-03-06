package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/gorilla/sessions"

	"github.com/stts-se/weblib/i18n"
	"github.com/stts-se/weblib/userdb"
	"github.com/stts-se/weblib/util"
)

func initI18NPropFiles(dir string) error {
	var msgs []string
	var err error
	var db *i18n.I18NDB

	db, err = i18n.ReadI18NPropDir(dir, "en")
	if err != nil {
		return err
	}
	msgs, err = db.CrossValidate()
	if err != nil {
		return fmt.Errorf("I18N cross validation failed : %v", err)
	}
	if len(msgs) > 0 {
		log.Printf("I18N cross validation errors below.\n")
		for _, msg := range msgs {
			fmt.Fprintf(os.Stderr, "I18N validation error : %s", msg)
		}
		return fmt.Errorf("I18N cross validation failed")
	}
	i18nCache = db
	return nil
}

func initUserDB(dbFile string) (*userdb.UserDB, error) {
	var constraints = func(userName, password string) (bool, string) {
		if len(userName) == 0 {
			return false, "empty user name"
		}
		if len(userName) < 4 {
			return false, "username must have min 4 chars"
		}
		if len(password) == 0 {
			return false, "empty password"
		}
		if len(password) < 4 {
			return false, "password must have min 4 chars"
		}
		return true, ""
	}

	loadedOrCreated := "Loaded"
	if !util.FileExists(dbFile) {
		loadedOrCreated = "Created"
	}

	userDB, err := userdb.ReadUserDB(dbFile)
	if err != nil {
		return userDB, fmt.Errorf("couldn't read user db : %v", err)
	}
	userDB.Constraints = constraints
	err = userDB.SaveFile()
	if err != nil {
		return userDB, fmt.Errorf("couldn't save user db : %v", err)
	}
	log.Printf("%s user database %s", loadedOrCreated, dbFile)
	return userDB, nil
}

func initRoleDB(dbFile string) (*userdb.RoleDB, error) {
	var constraints = func(role string, userNames []string) (bool, string) {
		if len(role) < 4 {
			return false, "role must have min 4 chars"
		}
		for _, userName := range userNames {
			if len(userName) < 4 {
				return false, "username must have min 4 chars"
			}
		}
		return true, ""
	}
	loadedOrCreated := "Loaded"
	if !util.FileExists(dbFile) {
		loadedOrCreated = "Created"
	}

	roleDB, err := userdb.ReadRoleDB(dbFile)
	if err != nil {
		return roleDB, fmt.Errorf("couldn't read role db : %v", err)
	}
	roleDB.Constraints = constraints
	err = roleDB.SaveFile()
	if err != nil {
		return roleDB, fmt.Errorf("couldn't save role db : %v", err)
	}
	log.Printf("%s role database %s", loadedOrCreated, dbFile)
	return roleDB, nil
}

func mkParentDir(fileName string) error {
	dir := filepath.Dir(fileName)
	return os.MkdirAll(dir, os.ModePerm)
}

func initCookieStore(keyFile string) (*sessions.CookieStore, error) {
	var cs *sessions.CookieStore
	var key []byte
	var err error
	if !util.FileExists(keyFile) {
		// Note: Don't store your key in your source code. Pass it via an
		// environmental variable, or flag (or both), and don't accidentally commit it
		// alongside your code. Ensure your key is sufficiently random - i.e. use Go's
		// crypto/rand or securecookie.GenerateRandomKey(32) and persist the result.
		//var cookieStore = sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))

		// key must be 16, 24 or 32 bytes long (AES-128, AES-192 or AES-256)

		fmt.Printf("No server key defined. Create new server key? (Ctrl-c to exit) [Y/n] ")
		reader := bufio.NewReader(os.Stdin)
		var r string
		r, err = reader.ReadString('\n')
		if err != nil {
			return cs, err
		}
		r = strings.ToLower(strings.TrimSpace(r))
		if len(r) > 0 && !strings.HasPrefix(r, "y") {
			fmt.Fprintf(os.Stderr, "BYE!\n")
			os.Exit(0)
		}

		err = mkParentDir(keyFile)
		if err != nil {
			return cs, err
		}

		key = make([]byte, 32)
		_, err = rand.Read(key)
		if err != nil {
			return cs, err
		}
		err = ioutil.WriteFile(keyFile, key, 0644)
		if err != nil {
			return cs, err
		}
		keyCheck, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return cs, fmt.Errorf("couldn't re-read key file")
		}
		if !reflect.DeepEqual(key, keyCheck) {
			return cs, fmt.Errorf("session key mismatch")
		}
		log.Printf("New key saved to file %s", keyFile)

	} else {
		key, err = ioutil.ReadFile(keyFile)
		if err != nil {
			return cs, err
		}
		if len(key) != 32 {
			return cs, fmt.Errorf("Invalid key length: %d", len(key))
		}
	}
	cs = sessions.NewCookieStore([]byte(key))
	return cs, nil
}
