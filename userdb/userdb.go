package userdb

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/stts-se/weblib/util"
)

// UserDB a database of users
type UserDB struct {
	mutex    *sync.RWMutex
	fileName string // optional
	users    map[string]string

	// Constraints is used to validate an input user + password
	// returns true + empty string if the user is valid
	// returns false + message if the user is invalid
	Constraints func(user string, password string) (bool, string)
}

var prms = &params{
	memory:      64 * 1024,
	iterations:  3,
	parallelism: 2,
	saltLength:  16,
	keyLength:   32,
}

// NewUserDB creates a new user database
func NewUserDB() *UserDB {
	return &UserDB{
		mutex:       &sync.RWMutex{},
		users:       make(map[string]string),
		Constraints: func(user string, password string) (bool, string) { return true, "" },
	}
}

// EmptyUserDB creates a new user database with the specified file name, which will be removed if it already exists
func EmptyUserDB(fileName string) (*UserDB, error) {
	res := NewUserDB()
	res.fileName = fileName
	err := res.clearFile()
	return res, err
}

// ReadUserDB reads a user db from file
func ReadUserDB(fileName string) (*UserDB, error) {
	res := &UserDB{
		mutex:       &sync.RWMutex{},
		fileName:    fileName,
		users:       make(map[string]string),
		Constraints: func(user string, password string) (bool, string) { return true, "" },
	}
	if !util.FileExists(fileName) {
		return res, nil
	}

	lines, err := util.ReadLines(fileName)
	if err != nil {
		return res, err
	}

	res.mutex.Lock()
	defer res.mutex.Unlock()

	for _, l := range lines {
		fs := strings.Split(l, FieldSeparator)
		if fs[0] == "DELETE" {
			userName := normaliseField(fs[1])
			if _, exists := res.users[userName]; !exists {
				return res, fmt.Errorf("no such user: %s", userName)
			}
			delete(res.users, userName)
		} else {
			userName := normaliseField(fs[0])
			password := fs[1]
			if _, exists := res.users[userName]; exists {
				return res, fmt.Errorf("user already exists: %s", userName)
			}

			if ok, msg := res.CheckConstraints(userName, password); !ok {
				return res, fmt.Errorf("constraints failed: %s", msg)
			}
			res.users[userName] = password
		}
	}
	return res, nil
}

// CheckConstraints to check if the db entry is valid given certain constraints
func (udb *UserDB) CheckConstraints(userName, password string) (bool, string) {
	if ok, msg := defaultConstraints("user", userName); !ok {
		return ok, msg
	}
	return udb.Constraints(userName, password)
}

// GetUsers returns the users defined in the database
func (udb *UserDB) GetUsers() []string {
	var res []string

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()

	for name := range udb.users {
		res = append(res, name)
	}

	sort.Strings(res)
	return res
}

// GetPasswordHash returns the password_hash value for userName. If no
// such value is found, the empty string is returned (along with a
// non-nil error value)
func (udb *UserDB) GetPasswordHash(userName string) (string, error) {

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normaliseField(userName)

	hash, ok := udb.users[userName]
	if !ok {
		return "", fmt.Errorf("no such user: %s", userName)
	}
	return hash, nil
}

// InsertUser is used to insert a user into the database
func (udb *UserDB) InsertUser(userName, password string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normaliseField(userName)

	if ok, msg := udb.CheckConstraints(userName, password); !ok {
		return fmt.Errorf("constraints failed: %s", msg)
	}

	passwordHash, err := generateFromPassword(password, prms)
	if err != nil {
		return fmt.Errorf("failed to generate hash: %v", err)
	}

	if _, exists := udb.users[userName]; exists {
		return fmt.Errorf("user already exists: %s", userName)
	}

	udb.users[userName] = passwordHash
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("%s%s%s", userName, FieldSeparator, passwordHash))
	}
	return nil
}

// DeleteUser is used to delete a user from the database
func (udb *UserDB) DeleteUser(userName string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normaliseField(userName)

	if _, exists := udb.users[userName]; !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	delete(udb.users, userName)
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("%s%s%s", "DELETE", FieldSeparator, userName))
	}
	return nil
}

// UpdatePassword updates the password for the specified user
func (udb *UserDB) UpdatePassword(userName string, password string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normaliseField(userName)

	if ok, msg := udb.Constraints(userName, password); !ok {
		return fmt.Errorf("constraints failed: %s", msg)
	}

	if _, exists := udb.users[userName]; !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	passwordHash, err := generateFromPassword(password, prms)
	if err != nil {
		return fmt.Errorf("failed to get user '%s' from user db : %v", userName, err)
	}

	udb.users[userName] = passwordHash
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("%s%s%s", "DELETE", FieldSeparator, userName))
		udb.appendToFile(fmt.Sprintf("%s%s%s", userName, FieldSeparator, passwordHash))
	}
	return nil
}

// Authorized is used to check if the password matches the specified user name
func (udb *UserDB) Authorized(userName, password string) (bool, error) {

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normaliseField(userName)

	ok := false

	hash, err := udb.GetPasswordHash(userName)
	if err != nil {
		return ok, fmt.Errorf("failed to get user '%s' from user db : %v", userName, err)
	}

	ok, err = comparePasswordAndHash(password, hash)
	if err != nil {
		return ok, err
	}

	return ok, nil
}

// UserExists check if a user with the specified user name. Second return value is the normalised version of the input user name.
func (udb *UserDB) UserExists(userName string) (bool, string) {

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normaliseField(userName)

	_, ok := udb.users[userName]
	return ok, userName
}

// SaveFile save the db to file
func (udb *UserDB) SaveFile() error {
	if udb.fileName == "" {
		return fmt.Errorf("file name not set")
	}

	udb.mutex.Lock()
	defer udb.mutex.Unlock()

	fh, err := os.Create(udb.fileName)
	if err != nil {
		return fmt.Errorf("failed to open file : %v", err)
	}
	defer fh.Close()

	for userName, hash := range udb.users {
		fmt.Fprintf(fh, "%s%s%s\n", userName, FieldSeparator, hash)
	}
	return nil
}

// NB that it is not thread-safe, and should be called after locking.
func (udb *UserDB) appendToFile(line string) error {
	fh, err := os.OpenFile(udb.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer fh.Close()

	_, err = fh.WriteString(line + "\n")
	if err != nil {
		return err
	}

	return nil
}

func (udb *UserDB) clearFile() error {
	if util.FileExists(udb.fileName) {
		err := os.Remove(udb.fileName)
		if err != nil {
			return err
		}
	}
	return nil
}
