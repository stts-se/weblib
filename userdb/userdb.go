package userdb

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"sync"
)

type UserDB struct {
	mutex       *sync.RWMutex
	fileName    string
	users       map[string]string
	Constraints func(user string, password string) bool
}

var prms = &params{
	memory:      64 * 1024,
	iterations:  3,
	parallelism: 2,
	saltLength:  16,
	keyLength:   32,
}

func NewUserDB() UserDB {
	return UserDB{
		mutex:       &sync.RWMutex{},
		users:       make(map[string]string),
		Constraints: func(user string, password string) bool { return true },
	}
}
func EmptyUserDB(fileName string) (UserDB, error) {
	res := NewUserDB()
	res.fileName = fileName
	err := res.clearFile()
	return res, err
}

func ReadUserDB(fileName string) (UserDB, error) {
	res, err := readFile(fileName)
	if err != nil {
		return res, err
	}
	return res, nil
}

func NewUserDBWithConstraints(fileName string, constraints func(user string, password string) bool) UserDB {
	return UserDB{
		mutex:       &sync.RWMutex{},
		fileName:    fileName,
		users:       make(map[string]string),
		Constraints: constraints,
	}
}

func normalise(userName string) string {
	return strings.ToLower(userName)
}

// GetUsers returns the users defined in the database
func (udb UserDB) GetUsers() ([]string, error) {
	var res []string

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()

	for name := range udb.users {
		res = append(res, name)
	}

	return res, nil
}

// GetUserByName looks up the user with the specified name
func (udb UserDB) UserExists(userName string) (string, bool) {
	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normalise(userName)

	_, exists := udb.users[userName]

	return userName, exists
}

// GetPasswordHash returns the password_hash value for userName. If no
// such value is found, the empty string is returned (along with a
// non-nil error value)
func (udb UserDB) GetPasswordHash(userName string) (string, error) {

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normalise(userName)

	hash, ok := udb.users[userName]
	if !ok {
		return "", fmt.Errorf("no such user: %s", userName)
	}
	return hash, nil
}

// InsertUser is used to insert a user into the database
func (udb UserDB) InsertUser(userName, password string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normalise(userName)

	passwordHash, err := generateFromPassword(password, prms)
	if err != nil {
		return fmt.Errorf("failed to generate hash: %v", err)
	}

	if _, exists := udb.users[userName]; exists {
		return fmt.Errorf("user already exists: %s", userName)
	}

	udb.users[userName] = passwordHash
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("%s\t%s", userName, passwordHash))
	}
	return nil
}

// DeleteUser is used to delete a user from the database
func (udb UserDB) DeleteUser(userName string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normalise(userName)

	if _, exists := udb.users[userName]; !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	delete(udb.users, userName)
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("DELETE\t%s", userName))
	}
	return nil
}

// UpdatePassword updates the fields of User except for User.ID and User.Name.
// Zero valued fields (empty string) will be treated as acceptable
// values, and updated to the empty string in the DB.
func (udb UserDB) UpdatePassword(userName string, password string) error {
	udb.mutex.Lock()
	defer udb.mutex.Unlock()
	userName = normalise(userName)

	if _, exists := udb.users[userName]; !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	passwordHash, err := generateFromPassword(password, prms)
	if err != nil {
		return fmt.Errorf("failed to get user '%s' from user db : %v", userName, err)
	}

	udb.users[userName] = passwordHash
	if udb.fileName != "" {
		udb.appendToFile(fmt.Sprintf("UPDATE\t%s\t%s", userName, passwordHash))
	}
	return nil
}

// Authorized is used to check if the password matches the specified user name
func (udb UserDB) Authorized(userName, password string) (bool, error) {

	udb.mutex.RLock()
	defer udb.mutex.RUnlock()
	userName = normalise(userName)

	ok := false

	res, err := udb.GetPasswordHash(userName)
	if err != nil {
		return ok, fmt.Errorf("failed to get user '%s' from user db : %v", userName, err)
	}

	ok, err = comparePasswordAndHash(password, res)
	if err != nil {
		return ok, fmt.Errorf("password doesn't match")
	}

	return ok, nil
}

func (udb UserDB) saveFile() error {

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
		fmt.Fprintf(fh, "%s\t%s\n", userName, hash)
	}
	return nil
}

// NB that it is not thread-safe, and should be called after locking.
func (udb UserDB) appendToFile(line string) error {
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

func readLines(fn string) ([]string, error) {
	var res []string
	var scanner *bufio.Scanner
	fh, err := os.Open(fn)
	if err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
	}

	if strings.HasSuffix(fn, ".gz") {
		gz, err := gzip.NewReader(fh)
		if err != nil {
			return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
		}
		scanner = bufio.NewScanner(gz)
	} else {
		scanner = bufio.NewScanner(fh)
	}
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
	}
	return res, nil
}

func readFile(fName string) (UserDB, error) {
	res := UserDB{
		mutex:       &sync.RWMutex{},
		fileName:    fName,
		users:       make(map[string]string),
		Constraints: func(user string, password string) bool { return true },
	}
	lines, err := readLines(fName)
	if err != nil {
		return res, err
	}

	res.mutex.Lock()
	defer res.mutex.Unlock()

	for _, l := range lines {
		fs := strings.Split(l, "\t")
		f1 := fs[0]
		if f1 == "DELETE" {
			userName := normalise(fs[1])
			if _, exists := res.users[userName]; !exists {
				return res, fmt.Errorf("no such user: %s", userName)
			}
			delete(res.users, userName)
		} else {
			userName := normalise(fs[0])
			passwordHash, err := generateFromPassword(fs[1], prms)
			if err != nil {
				return res, fmt.Errorf("failed to generate hash: %v", err)
			}

			if _, exists := res.users[userName]; exists {
				return res, fmt.Errorf("user already exists: %s", userName)
			}

			res.users[userName] = passwordHash
		}
	}
	return res, nil
}

func (udb UserDB) clearFile() error {
	if _, err := os.Stat(udb.fileName); !os.IsNotExist(err) {
		err := os.Remove(udb.fileName)
		if err != nil {
			return err
		}
	}
	return nil
}
