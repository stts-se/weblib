package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/stts-se/weblib/userdb"
)

func promptPassword() (string, error) {
	bytePassword, err := terminal.ReadPassword(0)
	fmt.Println()
	if err != nil {
		return "", err
	}
	password := string(bytePassword)
	return password, nil
}

func getUserDB(m meta, args []string) userdb.UserDB {
	dbFile := m.getArgValue(args, "dbfile")

	userDB, err := userdb.ReadUserDB(dbFile)
	if err != nil {
		log.Fatalf("Could't read user db : %v", err)
	}
	userDB.Constraints = func(userName, password string) (bool, string) {
		if len(userName) == 0 {
			return false, "empty user name"
		}
		if len(userName) < 4 {
			return false, "username must have min 4 chars"
		}
		if len(password) == 0 {
			return false, "empty password"
		}
		if len(password) < 4 {
			return false, "password must have min 4 chars"
		}
		return true, ""
	}
	fmt.Fprintf(os.Stderr, "Loaded user db from file %s\n", dbFile)
	return userDB
}

func insertUser(meta meta, args []string) {
	userDB := getUserDB(meta, args)
	userName := meta.getArgValue(args, "username")
	fmt.Printf("Password: ")
	password, err := promptPassword()
	if err != nil {
		log.Fatalf("Could't read password from terminal : %v", err)
	}
	fmt.Printf("Repeat password: ")
	passwordCheck, err := promptPassword()
	if err != nil {
		log.Fatalf("Could't read password from terminal : %v", err)
	}
	if password != passwordCheck {
		log.Fatalf("Passwords do not match")
	}
	err = userDB.InsertUser(userName, password)
	if err != nil {
		log.Fatalf("Couldn't insert user : %v", err)
	}
	fmt.Fprintf(os.Stderr, "Created user %s\n", userName)
}

func deleteUser(meta meta, args []string) {
	userDB := getUserDB(meta, args)
	userName := meta.getArgValue(args, "username")
	err := userDB.DeleteUser(userName)
	if err != nil {
		log.Fatalf("Couldn't delete user : %v", err)
	}
	fmt.Fprintf(os.Stderr, "Deleted user %s\n", userName)
}

func createDB(meta meta, args []string) {
	dbFile := meta.getArgValue(args, "dbfile")
	fh, err := os.Create(dbFile)
	if err != nil {
		log.Fatalf("Couldn't create db : %v", err)
	}
	fmt.Fprintf(fh, "")
}

func clearDB(meta meta, args []string) {
	dbFile := meta.getArgValue(args, "dbfile")
	fh, err := os.Create(dbFile)
	if err != nil {
		log.Fatalf("Couldn't clear db : %v", err)
	}
	fmt.Fprintf(fh, "")
}

func listUsers(meta meta, args []string) {
	userDB := getUserDB(meta, args)
	for _, u := range userDB.GetUsers() {
		fmt.Println(u)
	}
}

var cmds = []cmd{
	cmd{
		meta: meta{
			name:     "insert",
			desc:     "Insert user",
			argNames: []string{"dbfile", "username"},
		},
		f: insertUser,
	},
	cmd{
		meta: meta{
			name:     "delete",
			desc:     "Delete user",
			argNames: []string{"dbfile", "username"},
		},
		f: deleteUser,
	},
	cmd{
		meta: meta{
			name:     "list",
			desc:     "List users",
			argNames: []string{"dbfile"},
		},
		f: listUsers,
	},
	cmd{
		meta: meta{
			name:     "create",
			desc:     "Create empty database",
			argNames: []string{"dbfile"},
		},
		f: createDB,
	},
	cmd{
		meta: meta{
			name:     "clear",
			desc:     "Clear database",
			argNames: []string{"dbfile"},
		},
		f: clearDB,
	},
}

type meta struct {
	name     string
	desc     string
	argNames []string
}

type cmd struct {
	meta meta
	f    func(meta meta, args []string)
}

func (m meta) validateArgs(args []string) {
	if len(args) != len(m.argNames) {
		log.Fatalf("%s : required args [%s], found [%s]", m.name, strings.Join(m.argNames, " "), strings.Join(args, " "))
	}
}

func (m meta) getArgValue(args []string, argName string) string {
	m.validateArgs(args)
	for i, s := range m.argNames {
		if s == argName {
			return args[i]
		}
	}
	log.Fatalf("Invalid arg name: %s", argName)
	return ""
}

func (c cmd) apply(args []string) {
	c.f(c.meta, args)
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "userdb <command> <args>\n")
		for _, c := range cmds {
			fmt.Fprintf(os.Stderr, " %s %s\n", c.meta.name, strings.Join(c.meta.argNames, " "))
		}
		os.Exit(0)
	}

	cmdName := args[0]
	for _, c := range cmds {
		if c.meta.name == cmdName {
			c.apply(args[1:])
			os.Exit(0)
		}
	}
	log.Fatalf("Invalid command: %s", cmdName)
}
