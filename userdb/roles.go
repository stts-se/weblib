package userdb

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
)

// RoleDB a database of roles (username - roles)
type RoleDB struct {
	mutex    *sync.RWMutex
	fileName string // optional
	roles    map[string]map[string]bool

	// Constraints is used to validate an input user + role(s)
	// returns true + empty string if the role is valid
	// returns false + message if the role is invalid
	Constraints func(user string, role string) (bool, string)
}

// NewRoles creates a new user database
func NewRoleDB() RoleDB {
	return RoleDB{
		mutex:       &sync.RWMutex{},
		roles:       make(map[string]map[string]bool),
		Constraints: func(user string, role string) (bool, string) { return true, "" },
	}
}

// EmptyRoleDB creates a new role database with the specified file name, which will be removed if it already exists
func EmptyRoleDB(fileName string) (RoleDB, error) {
	res := NewRoleDB()
	res.fileName = fileName
	err := res.clearFile()
	return res, err
}

// ReadRoleDB reads a role db from file
func ReadRoleDB(fileName string) (RoleDB, error) {
	return readRolesFile(fileName)
}

func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// GetRoles returns the roles defined in the database
func (rdb RoleDB) GetRoles() []string {
	var res []string

	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()

	for _, roles := range rdb.roles {
		for role := range roles {
			if !contains(res, role) {
				res = append(res, role)
			}
		}
	}
	sort.Strings(res)
	return res
}

// GetUsers returns the users defined in the database
func (rdb RoleDB) GetUsers() []string {
	var res []string

	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()

	for name := range rdb.roles {
		res = append(res, name)
	}

	sort.Strings(res)
	return res
}

// InsertRole is used to insert a user into the database
func (rdb RoleDB) InsertRole(userName, role string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	userName = normaliseUserName(userName)

	if ok, msg := rdb.Constraints(userName, role); !ok {
		return fmt.Errorf("constraints failed: %s", msg)
	}

	if _, exists := rdb.roles[userName]; exists {
		return fmt.Errorf("user already exists: %s", userName)
	}

	rdb.roles[userName][role] = true
	if rdb.fileName != "" {
		rdb.appendToFile(fmt.Sprintf("%s\t%s", userName, role))
	}
	return nil
}

// DeleteRole is used to delete a user role from the database
func (rdb RoleDB) DeleteRole(userName, role string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	userName = normaliseUserName(userName)

	roles, exists := rdb.roles[userName]
	if !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	if _, exists := roles[role]; !exists {
		return fmt.Errorf("no role %s for user: %s", role, userName)
	}
	delete(roles, role)
	rdb.roles[userName] = roles
	if rdb.fileName != "" {
		rdb.appendToFile(fmt.Sprintf("DELETE\t%s\t%s", userName, role))
	}
	return nil
}

// DeleteUser is used to delete a user from the database
func (rdb RoleDB) DeleteUser(userName string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	userName = normaliseUserName(userName)

	if _, exists := rdb.roles[userName]; !exists {
		return fmt.Errorf("no such user: %s", userName)
	}
	delete(rdb.roles, userName)
	if rdb.fileName != "" {
		rdb.appendToFile(fmt.Sprintf("DELETE\t%s", userName))
	}
	return nil
}

// Authorized is used to check if the password matches the specified user name
func (rdb RoleDB) Authorized(userName, role string) bool {

	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()
	userName = normaliseUserName(userName)

	_, ok := rdb.roles[userName][role]
	return ok
}

func (rdb RoleDB) SaveFile() error {
	if rdb.fileName == "" {
		return fmt.Errorf("file name not set")
	}

	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()

	fh, err := os.Create(rdb.fileName)
	if err != nil {
		return fmt.Errorf("failed to open file : %v", err)
	}
	defer fh.Close()

	for userName, roleMap := range rdb.roles {
		roles := []string{}
		for role := range roleMap {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		fmt.Fprintf(fh, "%s\t%s\n", userName, strings.Join(roles, ":"))
	}
	return nil
}

// NB that it is not thread-safe, and should be called after locking.
func (rdb RoleDB) appendToFile(line string) error {
	fh, err := os.OpenFile(rdb.fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
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

func readRolesFile(fName string) (RoleDB, error) {
	res := RoleDB{
		mutex:       &sync.RWMutex{},
		fileName:    fName,
		roles:       make(map[string]map[string]bool),
		Constraints: func(user string, role string) (bool, string) { return true, "" },
	}
	if !fileExists(fName) {
		return res, nil
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
			userName := normaliseUserName(fs[1])
			if _, exists := res.roles[userName]; !exists {
				return res, fmt.Errorf("no such user: %s", userName)
			}
			if len(fs) >= 3 {
				role := fs[2]
				roles := res.roles[userName]
				if _, exists := roles[role]; !exists {
					return res, fmt.Errorf("no role %s for user: %s", role, userName)
				}
				delete(roles, role)
				res.roles[userName] = roles
			} else {
				delete(res.roles, userName)
			}
		} else {
			userName := normaliseUserName(fs[0])
			if _, exists := res.roles[userName]; exists {
				return res, fmt.Errorf("user already exists: %s", userName)
			}
			roles := make(map[string]bool)
			for _, s := range strings.Split(fs[1], ":") {
				roles[s] = true
			}
			res.roles[userName] = roles
		}
	}
	return res, nil
}

func (rdb RoleDB) clearFile() error {
	if fileExists(rdb.fileName) {
		err := os.Remove(rdb.fileName)
		if err != nil {
			return err
		}
	}
	return nil
}
