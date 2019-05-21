package userdb

import (
	"fmt"
	"os"
	"strings"
)

// FieldSeparator separates fields in a file (for file reading/writing)
const FieldSeparator = "\t"

// ItemSeparator separates items in a list (for file reading/writing)
const ItemSeparator = " "

var defaultConstraints = func(fieldName, fieldValue string) (bool, string) {
	if len(fieldValue) == 0 {
		return false, fmt.Sprintf("empty %s", fieldName)
	}
	if strings.Contains(fieldValue, FieldSeparator) {
		return false, fmt.Sprintf("%s cannot contain %s", fieldName, FieldSeparator)
	}
	if strings.Contains(fieldValue, ItemSeparator) {
		return false, fmt.Sprintf("%s cannot contain %s", fieldName, FieldSeparator)
	}
	if normaliseField(fieldValue) != fieldValue {
		return false, fmt.Sprintf("%s is not normalised", fieldName)
	}
	return true, ""
}

func fileExists(fileName string) bool {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false
	}
	return true
}

func normaliseField(field string) string {
	return strings.TrimSpace(strings.ToLower(field))
}

func contains(slice []string, value string) bool {
	for _, s := range slice {
		if s == value {
			return true
		}
	}
	return false
}

// Validate user db with role db (all user names in the role db must be defined in the user db)
func Validate(userDB *UserDB, roleDB *RoleDB) error {
	for role, users := range roleDB.ListRolesAndUsers() {
		for _, user := range users {
			if exists, _ := userDB.UserExists(user); !exists {
				return fmt.Errorf("role %s contains invalid user: %s", role, user)
			}
		}
	}
	return nil
}
