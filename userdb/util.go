package userdb

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
)

const fieldSeparator = "\t" // separates fields in a file
const itemSeparator = " "   // separates items in a list

var defaultConstraints = func(fieldName, fieldValue string) (bool, string) {
	if len(fieldValue) == 0 {
		return false, fmt.Sprintf("empty %s", fieldName)
	}
	if strings.Contains(fieldValue, fieldSeparator) {
		return false, fmt.Sprintf("%s cannot contain %s", fieldName, fieldSeparator)
	}
	if strings.Contains(fieldValue, itemSeparator) {
		return false, fmt.Sprintf("%s cannot contain %s", fieldName, fieldSeparator)
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

func readLines(fn string) ([]string, error) {
	var res []string
	var scanner *bufio.Scanner
	fh, err := os.Open(fn)
	if err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
	}

	if strings.HasSuffix(fn, ".gz") {
		gz, err := gzip.NewReader(fh)
		if err != nil {
			return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
		}
		scanner = bufio.NewScanner(gz)
	} else {
		scanner = bufio.NewScanner(fh)
	}
	for scanner.Scan() {
		res = append(res, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return res, fmt.Errorf("failed to read '%s' : %v", fn, err)
	}
	return res, nil
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
