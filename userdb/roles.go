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

	// Constraints is used to validate an input role + users
	// returns true + empty string if the role/users are valid
	// returns false + message if the role/users are invalid
	Constraints func(role string, users []string) (bool, string)
}

// NewRoleDB creates a new user database
func NewRoleDB() *RoleDB {
	return &RoleDB{
		mutex:       &sync.RWMutex{},
		roles:       make(map[string]map[string]bool),
		Constraints: func(role string, users []string) (bool, string) { return true, "" },
	}
}

// EmptyRoleDB creates a new role database with the specified file name, which will be removed if it already exists
func EmptyRoleDB(fileName string) (*RoleDB, error) {
	res := NewRoleDB()
	res.fileName = fileName
	err := res.clearFile()
	return res, err
}

// ReadRoleDB reads a role db from file
func ReadRoleDB(fileName string) (*RoleDB, error) {
	res := &RoleDB{
		mutex:       &sync.RWMutex{},
		fileName:    fileName,
		roles:       make(map[string]map[string]bool),
		Constraints: func(role string, users []string) (bool, string) { return true, "" },
	}
	if !fileExists(fileName) {
		return res, nil
	}

	lines, err := readLines(fileName)
	if err != nil {
		return res, err
	}

	res.mutex.Lock()
	defer res.mutex.Unlock()

	for _, l := range lines {
		fs := strings.Split(l, fieldSeparator)
		if fs[0] == "DELETE" {
			role := normaliseField(fs[1])
			if _, exists := res.roles[role]; !exists {
				return res, fmt.Errorf("no such role: %s", role)
			}
			delete(res.roles, role)
		} else {
			role := fs[0]
			userNames := strings.Split(fs[1], itemSeparator)
			userMap := make(map[string]bool)
			for _, userName := range userNames {
				userMap[userName] = true
			}
			if ok, msg := res.CheckConstraints(role, userNames); !ok {
				return res, fmt.Errorf("constraints failed: %s", msg)
			}
			res.roles[role] = userMap
		}
	}
	return res, nil
}

// CheckConstraints to check if the db entry is valid given certain constraints
func (rdb *RoleDB) CheckConstraints(role string, users []string) (bool, string) {
	if ok, msg := defaultConstraints("role", role); !ok {
		return ok, msg
	}
	for _, userName := range users {
		if ok, msg := defaultConstraints("user", userName); !ok {
			return ok, msg
		}
	}
	return rdb.Constraints(role, users)
}

// GetRoles returns the roles defined in the database
func (rdb *RoleDB) GetRoles() []string {
	var res []string

	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()

	for role := range rdb.roles {
		if !contains(res, role) {
			res = append(res, role)
		}
	}
	sort.Strings(res)
	return res
}

// InsertRole is used to insert a user into the database
func (rdb *RoleDB) InsertRole(role string, userNames []string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	role = normaliseField(role)
	if ok, msg := rdb.CheckConstraints(role, userNames); !ok {
		return fmt.Errorf("constraints failed: %s", msg)
	}
	if _, exists := rdb.roles[role]; exists {
		return fmt.Errorf("role already exists: %s", role)
	}

	userMap := make(map[string]bool)
	for _, userName := range userNames {
		userName = normaliseField(userName)
		userMap[userName] = true
	}
	rdb.roles[role] = userMap

	if rdb.fileName != "" {
		userNames = []string{}
		for userName := range userMap {
			userNames = append(userNames, userName)
		}
		sort.Strings(userNames)
		rdb.appendToFile(fmt.Sprintf("%s%s%s", role, fieldSeparator, strings.Join(userNames, itemSeparator)))
	}
	return nil
}

// DeleteRole is used to delete a user role from the database
func (rdb *RoleDB) DeleteRole(role string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	role = normaliseField(role)

	if _, exists := rdb.roles[role]; !exists {
		return fmt.Errorf("no such role: %s", role)
	}
	delete(rdb.roles, role)
	if rdb.fileName != "" {
		rdb.appendToFile(fmt.Sprintf("%s%s%s", "DELETE", fieldSeparator, role))
	}
	return nil
}

// DeleteUserRole is used to delete a user role from the database
func (rdb *RoleDB) DeleteUserRole(role, userName string) error {
	rdb.mutex.Lock()
	defer rdb.mutex.Unlock()
	role = normaliseField(role)

	userMap, exists := rdb.roles[role]
	if !exists {
		return fmt.Errorf("no such role: %s", role)
	}

	if _, exists = userMap[userName]; !exists {
		return fmt.Errorf("no such role for user: %s", userName)
	}
	delete(userMap, userName)
	rdb.roles[role] = userMap

	if rdb.fileName != "" {
		rdb.appendToFile(fmt.Sprintf("%s%s%s", "DELETE", fieldSeparator, role))
		userNames := []string{}
		for userName := range userMap {
			userNames = append(userNames, userName)
		}
		sort.Strings(userNames)
		rdb.appendToFile(fmt.Sprintf("%s%s%s", role, fieldSeparator, strings.Join(userNames, itemSeparator)))
	}
	return nil
}

// Authorized is used to check if a user has access to a specified role
func (rdb *RoleDB) Authorized(role, userName string) bool {
	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()
	role = normaliseField(role)
	userName = normaliseField(userName)

	_, ok := rdb.roles[role][userName]
	return ok
}

// RoleExists looks up the role with the specified name
func (rdb *RoleDB) RoleExists(role string) bool {
	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()
	role = normaliseField(role)

	_, exists := rdb.roles[role]
	return exists
}

// ListUsers looks up the users for the specified role
func (rdb *RoleDB) ListUsers(role string) ([]string, bool) {
	rdb.mutex.RLock()
	defer rdb.mutex.RUnlock()
	role = normaliseField(role)

	userMap, exists := rdb.roles[role]
	userNames := []string{}
	for userName := range userMap {
		userNames = append(userNames, userName)
	}
	return userNames, exists
}

// SaveFile save the db to file
func (rdb *RoleDB) SaveFile() error {
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

	for role, userMap := range rdb.roles {
		userNames := []string{}
		for userName := range userMap {
			userNames = append(userNames, userName)
		}
		sort.Strings(userNames)
		fmt.Fprintf(fh, "%s%s%s\n", role, fieldSeparator, strings.Join(userNames, itemSeparator))
	}
	return nil
}

// NB that it is not thread-safe, and should be called after locking.
func (rdb *RoleDB) appendToFile(line string) error {
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

func (rdb *RoleDB) clearFile() error {
	if fileExists(rdb.fileName) {
		err := os.Remove(rdb.fileName)
		if err != nil {
			return err
		}
	}
	return nil
}
