package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/stts-se/weblib/i18n"
)

func printHelp() {
	fmt.Fprintf(os.Stderr, "Cmd line validation for i18n property files\n")
	fmt.Fprintf(os.Stderr, "Usage: i18n <i18n files>\n")
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(0)
	}

	dir := filepath.Dir(os.Args[1])
	db, err := i18n.ReadI18NPropFiles(dir, os.Args[1:], "")
	if err != nil {
		log.Fatal(err)
	}
	msgs, err := db.CrossValidate()
	if err != nil {
		log.Fatal(err)
	}
	if len(msgs) > 0 {
		fmt.Fprintf(os.Stderr, "Cross validation errors\n")
		for _, msg := range msgs {
			fmt.Fprintf(os.Stderr, "%s\n", msg)
		}
		log.Fatal("Cross validation failed")
	}
}
