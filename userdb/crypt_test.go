package userdb

import (
	"testing"
	//"fmt"
)

func Test_Crypt1(t *testing.T) {
	var err error

	var ps = &params{
		memory:      64 * 1024,
		iterations:  3,
		parallelism: 2,
		saltLength:  16,
		keyLength:   32,
	}

	password := "imsosecret"
	passwordHash, err := generateFromPassword(password, ps)
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}
	match, err := comparePasswordAndHash(password, passwordHash)
	if err != nil {
		t.Errorf("didn't expect error here : %v", err)
	}
	if w, g := true, match; w != g {
		t.Errorf(fs, w, g)
	}

}
