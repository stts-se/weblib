package main

import (
	"crypto/rand"
	"fmt"
	"os"
)

func fileExists(fileName string) bool {
	if _, err := os.Stat(fileName); os.IsNotExist(err) {
		return false
	}
	return true
}

func genUUID(len int) (s string, err error) {
	b := make([]byte, len)
	_, err = rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b), nil
}
