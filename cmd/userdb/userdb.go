package main

import (
	"bufio"
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

func getUserDB(dbFile string) *userdb.UserDB {
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

func insertUser(meta meta, dbFile string, args []string) {
	userDB := getUserDB(dbFile)
	//userName := meta.getArgValue(args, "username")

	fmt.Printf("Username: ")
	reader := bufio.NewReader(os.Stdin)
	userName, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Could't read username from terminal : %v", err)
	}
	userName = strings.ToLower(strings.TrimSpace(userName))

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
	err = userDB.SaveFile()
	if err != nil {
		log.Fatalf("Couldn't save db : %v", err)
	}
}

func deleteUsers(meta meta, dbFile string, args []string) {
	userDB := getUserDB(dbFile)
	userNames := meta.getArgValues(args, "usernames*")
	for _, userName := range userNames {
		err := userDB.DeleteUser(userName)
		if err != nil {
			log.Fatalf("Couldn't delete user : %v", err)
		}
		fmt.Fprintf(os.Stderr, "Deleted user %s\n", userName)
	}
	err := userDB.SaveFile()
	if err != nil {
		log.Fatalf("Couldn't save db : %v", err)
	}
}

func createDB(meta meta, dbFile string, args []string) {
	fh, err := os.Create(dbFile)
	if err != nil {
		log.Fatalf("Couldn't create db : %v", err)
	}
	fmt.Fprintf(fh, "")
}

func clearDB(meta meta, dbFile string, args []string) {
	fh, err := os.Create(dbFile)
	if err != nil {
		log.Fatalf("Couldn't clear db : %v", err)
	}
	fmt.Fprintf(fh, "")
}

func listUsers(meta meta, dbFile string, args []string) {
	userDB := getUserDB(dbFile)
	users := userDB.GetUsers()
	for _, u := range users {
		fmt.Println(u)
	}
	pluralS := "s"
	if len(users) == 1 {
		pluralS = ""
	}
	fmt.Printf("%d user%s\n", len(users), pluralS)
}

var cmds = []cmd{
	{
		meta: meta{
			name:     "insert",
			desc:     "Insert user",
			argNames: []string{},
		},
		f: insertUser,
	},
	{
		meta: meta{
			name:     "delete",
			desc:     "Delete users",
			argNames: []string{"usernames*"},
		},
		f: deleteUsers,
	},
	{
		meta: meta{
			name:     "list",
			desc:     "List users",
			argNames: []string{},
		},
		f: listUsers,
	},
	{
		meta: meta{
			name:     "create",
			desc:     "Create empty database",
			argNames: []string{},
		},
		f: createDB,
	},
	{
		meta: meta{
			name:     "clear",
			desc:     "Clear database",
			argNames: []string{},
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
	f    func(meta meta, dbFile string, args []string)
}

func (m meta) validateArgs(args []string) {
	for i, arg := range m.argNames {
		if i != len(m.argNames)-1 && strings.HasSuffix(arg, "*") {
			log.Fatalf("%s : variable length argument can only be used in final position, found [%s]", m.name, strings.Join(m.argNames, " "))
		}
	}
	lastArgName := m.argNames[len(m.argNames)-1]
	if strings.HasSuffix(lastArgName, "*") {
		args = args[0:len(m.argNames)]
	}
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

func (m meta) getArgValues(args []string, argName string) []string {
	m.validateArgs(args)
	startIndex := len(args)
	res := []string{}
	for i, s := range m.argNames {
		if s == argName {
			startIndex = i
		}
	}
	for i, s := range args {
		if i >= startIndex {
			res = append(res, s)
		}
	}
	if len(res) == 0 {
		log.Fatalf("Invalid arg name: %s", argName)
	}
	return res
}

func (c cmd) apply(dbFile string, args []string) {
	c.f(c.meta, dbFile, args)
}

func printHelp() {
	fmt.Fprintf(os.Stderr, "userdb <dbfile> <command> <args>\n")
	fmt.Fprintf(os.Stderr, " %s\n", "help")
	for _, c := range cmds {
		args := []string{}
		for _, a := range c.meta.argNames {
			args = append(args, fmt.Sprintf("<%s>", a))
		}
		argsString := strings.Join(args, " ")
		fmt.Fprintf(os.Stderr, " %s %s\n", c.meta.name, argsString)
	}
}

func main() {
	args := os.Args[1:]
	if len(args) < 2 || args[0] == "help" {
		printHelp()
		os.Exit(0)
	}

	var dbFile = args[0]
	var cmdName = args[1]

	if cmdName == "help" {
		printHelp()
		os.Exit(0)
	}
	for _, c := range cmds {
		if c.meta.name == cmdName {
			c.apply(dbFile, args[2:])
			os.Exit(0)
		}
	}
	log.Fatalf("Invalid command: %s", cmdName)
}
