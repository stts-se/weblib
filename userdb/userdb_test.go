package userdb

import (
	"strings"
	//"os"
	"testing"
	//"fmt"
)

var fs = "Expected '%v', got '%v'"

func Test_UserDB(t *testing.T) {
	var err error

	udb := NewUserDB()

	u := "KalleA"

	s1, err0 := udb.GetPasswordHash("KalleA")
	if w, g := "", s1; w != g {
		t.Errorf(fs, w, g)
	}
	if err0 == nil {
		t.Error("expected error, got nil")
	}

	err = udb.InsertUser(u, "secret")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	userName, exists := udb.UserExists("KalleA")
	if !exists {
		t.Errorf("oh no : %v", err)
	}
	if w, g := "kallea", userName; w != g {
		t.Errorf(fs, w, g)
	}
	s2, err2 := udb.GetPasswordHash("KalleA")
	if s2 == "" {
		t.Errorf("expected password hash, got empty string")
	}
	if err2 != nil {
		t.Errorf("expected nil, got %v", err)
	}

	ok, err := udb.Authorized(userName, "secret")
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	ok, err = udb.Authorized(userName, "wrongily")
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}
	if w, g := false, ok; w != g {
		t.Errorf(fs, w, g)
	}

	// change password
	err = udb.UpdatePassword(userName, "another_secret")
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}

	// test with new password
	ok, err = udb.Authorized(userName, "another_secret")
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}
}

func Test_UserDB_File(t *testing.T) {
	var err error
	udb1, err := EmptyUserDB("test_files/userdb_test_file")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	err = udb1.InsertUser("angela", "angelas-secret")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	err = udb1.InsertUser("robert", "roberts-secret")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	err = udb1.DeleteUser("robert")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	udb2, err := ReadUserDB(udb1.fileName)
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	_, exists := udb2.UserExists("robert")
	if exists {
		t.Errorf("oh no : %v", err)
	}

	// login udb1 angela
	ok, err := udb1.Authorized("angela", "angelas-secret")
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	// login udb2 angela
	ok, err = udb2.Authorized("angela", "angelas-secret")
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}

	if w, g := true, ok; w != g {
		t.Errorf(fs, w, g)
	}

	lines, err := readLines(udb1.fileName)
	if err != nil {
		t.Errorf("Fail: %v", err)
	}
	if w, g := 3, len(lines); w != g {
		t.Errorf(fs, w, g)
		t.Errorf("%s", strings.Join(lines, "\n"))
	}

	_, err = readLines("test_files/userdb_test_file_does_not_exist")
	if err == nil {
		t.Errorf("Fail: expected error here")
	}

	//lines, err := readLines(udb1.fileName)
}

func Test_UserDB_Constraints(t *testing.T) {
	var err error
	udb := NewUserDB()

	udb.Constraints = func(userName, password string) (bool, string) {
		if len(userName) == 0 {
			return false, "empty user name"
		}
		if !strings.Contains(userName, "@") {
			return false, "user name must be an email address"
		}
		if len(password) == 0 {
			return false, "empty password"
		}
		return true, ""
	}

	err = udb.InsertUser("leif", "secret1asds%sdd+")
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = udb.InsertUser("nizze@somewhere.else", "aspbsdf sdx2")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

	err = udb.InsertUser("", "sdx2")
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = udb.UpdatePassword("nizze@somewhere.else", "")
	if err == nil {
		t.Errorf("Fail: expected error")
	}

	err = udb.UpdatePassword("nizze@somewhere.else", "asdfasdfasdf ")
	if err != nil {
		t.Errorf("Fail: %v", err)
	}

}
