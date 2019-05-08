package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strings"

	"github.com/gorilla/sessions"
	"golang.org/x/crypto/ssh/terminal"

	"github.com/stts-se/weblib/userdb"
)

func promptPassword() (string, error) {
	bytePassword, err := terminal.ReadPassword(0)
	fmt.Println()
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	return password, nil
}

func initCookieStore(keyFile string) (*sessions.CookieStore, error) {
	var cs *sessions.CookieStore
	var key []byte
	var err error
	if !fileExists(keyFile) {
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

func initDB(dbFile string) (userdb.UserDB, error) {
	userDB, err := userdb.ReadUserDB(dbFile)
	if err != nil {
		return userDB, err
	}
	userDB.Constraints = func(userName, password string) (bool, string) {
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

	if len(userDB.GetUsers()) == 0 {
		reader := bufio.NewReader(os.Stdin)
		fmt.Println("Empty user db. Create new user? (Ctrl-c to exit)")
		for {
			fmt.Printf("Username: ")
			userName, err := reader.ReadString('\n')
			if err != nil {
				return userDB, err
			}

			fmt.Printf("Password: ")
			password, err := promptPassword()
			if err != nil {
				return userDB, err
			}
			fmt.Printf("Repeat password: ")
			passwordCheck, err := promptPassword()
			if err != nil {
				return userDB, err
			}
			if password != passwordCheck {
				return userDB, fmt.Errorf("Passwords do not match")
			}
			err = userDB.InsertUser(userName, password)
			if err != nil {
				return userDB, err
			}
			log.Printf("Created user %s", userName)
			fmt.Printf("Create another user? [Y/n] ")
			r, err := reader.ReadString('\n')
			if err != nil {
				return userDB, err
			}
			r = strings.ToLower(strings.TrimSpace(r))
			if len(r) > 0 && !strings.HasPrefix(r, "y") {
				break
			}
		}
	}
	return userDB, nil
}
