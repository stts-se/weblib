package userdb

import (
	"strings"
	//"os"
	"testing"
)

//var fs = "Expected '%v', got '%v'"

func Test_RoleDB(t *testing.T) {
	var err error

	rdb := NewRoleDB()

	r := "user"

	err = rdb.InsertRole("user", []string{"john"})
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	exists := rdb.RoleExists(r)
	if !exists {
		t.Errorf("oh no : %v", err)
	}
	users, ok := rdb.ListUsers(r)
	if len(users) != 1 || users[0] != "john" {
		t.Errorf("expected john, got %v", users)
	}
	if !ok {
		t.Errorf("expected ok=true, got %v", ok)
	}

	ok = rdb.Authorized(r, "john")
	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	ok = rdb.Authorized(r, "joh")
	if w, g := false, ok; w != g {
		t.Errorf(fs, w, g)
	}

	// change role
	err = rdb.DeleteRole(r)
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}
	err = rdb.InsertRole(r, []string{"angela", "carole"})
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}

	// test with new role
	ok = rdb.Authorized(r, "angela")
	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}
}

func Test_RoleDB_File(t *testing.T) {
	var err error
	rdb1, err := EmptyRoleDB("test_files/roledb_test_file")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	err = rdb1.InsertRole("user", []string{"angela", "james"})
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	err = rdb1.InsertRole("admin", []string{"james"})
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	err = rdb1.DeleteRole("admin")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	rdb2, err := ReadRoleDB(rdb1.fileName)
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	exists := rdb2.RoleExists("admin")
	if exists {
		t.Errorf("oh no : %v", err)
	}

	ok := rdb1.Authorized("user", "james")
	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	ok = rdb2.Authorized("user", "james")
	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	ok = rdb1.Authorized("admin", "james")
	if w, g := ok, ok; w != g {
		t.Errorf(fs, w, g)
	}

	ok = rdb2.Authorized("admin", "james")
	if w, g := false, ok; w != g {
		t.Errorf(fs, w, g)
	}

	lines, err := readLines(rdb1.fileName)
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	if w, g := 3, len(lines); w != g {
		t.Errorf(fs, w, g)
		t.Errorf("%s", strings.Join(lines, "\n"))
	}

	_, err = readLines("test_files/roledb_test_file_does_not_exist")
	if err == nil {
		t.Errorf("Fail: expected error here")
	}
}

func Test_RoleDB_Constraints(t *testing.T) {
	var err error
	rdb := NewRoleDB()

	rdb.Constraints = func(role string, userNames []string) (bool, string) {
		if len(role) == 0 {
			return false, "empty role name"
		}
		if !strings.Contains(role, "@") {
			return false, "role name must contain @"
		}
		if len(userNames) == 0 {
			return false, "empty user list"
		}
		return true, ""
	}

	err = rdb.InsertRole("user", []string{"leif"})
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = rdb.InsertRole("admin@somedomain.se", []string{})
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = rdb.InsertRole("", []string{"anton"})
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = rdb.InsertRole("admin@domain.com", []string{"anton"})
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
}
