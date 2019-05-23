package i18n

import (
	"fmt"
	"testing"
)

var fs = "Expected '%v', got '%v'"

func Test_I18N(t *testing.T) {
	i18n := NewI18N("sv")
	i18n.dict = map[string]string{
		"Logged in":                     "Inloggad",
		"Logged in as user %s":          "Inloggad som användare %s",
		"Logged in as user %s, role %s": "Inloggad som användare %s, roll %s",
	}

	if exp, got := "Inloggad", i18n.S("Logged in"); exp != got {
		t.Errorf(fs, exp, got)
	}

	if exp, got := "Inloggad som användare hanna", i18n.S("Logged in as user %s", "hanna"); exp != got {
		t.Errorf(fs, exp, got)
	}

	if exp, got := "Inloggad som användare hanna, roll admin", i18n.S("Logged in as user %s, role %s", "hanna", "admin"); exp != got {
		t.Errorf(fs, exp, got)
	}

	inputSlice1 := []interface{}{"hanna", "admin"}
	if exp, got := "Inloggad som användare hanna, roll admin", i18n.S("Logged in as user %s, role %s", inputSlice1...); exp != got {
		t.Errorf(fs, exp, got)
	}

	inputSlice2 := []string{"hanna", "admin"}
	if exp, got := "Inloggad som användare hanna, roll admin", i18n.S("Logged in as user %s, role %s", inputSlice2); exp != got {
		t.Errorf(fs, exp, got)
	}

}

func Test_ValidateI18NPropFiles_Valid(t *testing.T) {
	var msgs []string
	var err error
	var data map[string]*I18N

	dir := "test_files/valid"

	data, err = readI18NPropFiles(dir)
	if err != nil {
		t.Errorf("Unexpected error : %v", err)
	}
	msgs, err = crossValidateI18NPropFiles(data, dir)
	if err != nil {
		t.Errorf("Unexpected error : %v", err)
	}
	if len(msgs) > 0 {
		for _, msg := range msgs {
			t.Errorf("Unexpected validation error : %v", msg)
		}
	}
}

func Test_ValidateI18NPropFiles_Invalid(t *testing.T) {
	var msgs []string
	var err error
	var data map[string]*I18N

	dir := "test_files/invalid"

	data, err = readI18NPropFiles(dir)
	if err != nil {
		t.Errorf("Unexpected error : %v", err)
	}
	msgs, err = crossValidateI18NPropFiles(data, dir)
	if len(msgs) == 0 {
		t.Errorf("Expected validation errors, got none.")
	}
	fmt.Printf("YES!! Wanted validation errors, and got validation errors!\n")
	for _, msg := range msgs {
		fmt.Printf(" - %v\n", msg)
	}
}
