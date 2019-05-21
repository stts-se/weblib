package i18n

import (
	"testing"
)

var fs = "Expected '%v', got '%v'"

func Test_I18N(t *testing.T) {
	i18n := I18N{
		"Logged in":                     "Inloggad",
		"Logged in as user %s":          "Inloggad som användare %s",
		"Logged in as user %s, role %s": "Inloggad som användare %s, roll %s",
	}

	if exp, got := "Inloggad", i18n.S("Logged in"); exp != got {
		t.Errorf(fs, exp, got)
	}

	// if exp, got := "Inloggad som användare hanna", i18n.S("Logged in as user %s", "hanna"); exp != got {
	// 	t.Errorf(fs, exp, got)
	// }

	// if exp, got := "Inloggad som användare hanna, roll admin", i18n.S("Logged in as user %s, role %s", "hanna", "admin"); exp != got {
	// 	t.Errorf(fs, exp, got)
	// }

}
