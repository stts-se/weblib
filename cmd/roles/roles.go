package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/stts-se/weblib/userdb"
)

func getRoleDB(dbFile string) *userdb.RoleDB {
	roleDB, err := userdb.ReadRoleDB(dbFile)
	if err != nil {
		log.Fatalf("Could't read role db : %v", err)
	}
	roleDB.Constraints = func(role string, userNames []string) (bool, string) {
		if len(role) == 0 {
			return false, "empty role"
		}
		if len(role) < 4 {
			return false, "role must have min 4 chars"
		}
		for _, userName := range userNames {
			if len(userName) == 0 {
				return false, "empty user name"
			}
			if len(userName) < 4 {
				return false, "username must have min 4 chars"
			}
		}
		return true, ""
	}
	fmt.Fprintf(os.Stderr, "Loaded role db from file %s\n", dbFile)
	return roleDB
}

func insertRole(meta meta, dbFile string, args []string) {
	roleDB := getRoleDB(dbFile)
	role := meta.getArgValue(args, "role")
	userNames := meta.getArgValues(args, "usernames*")
	err := roleDB.InsertRole(role, userNames)
	if err != nil {
		log.Fatalf("Couldn't insert role : %v", err)
	}
	fmt.Fprintf(os.Stderr, "Created role %s\n", role)
	err = roleDB.SaveFile()
	if err != nil {
		log.Fatalf("Couldn't save db : %v", err)
	}
}

func deleteRoles(meta meta, dbFile string, args []string) {
	roleDB := getRoleDB(dbFile)
	roles := meta.getArgValues(args, "roles*")
	for _, role := range roles {
		err := roleDB.DeleteRole(role)
		if err != nil {
			log.Fatalf("Couldn't delete role : %v", err)
		}
		fmt.Fprintf(os.Stderr, "Deleted role %s\n", role)
		err = roleDB.SaveFile()
		if err != nil {
			log.Fatalf("Couldn't save db : %v", err)
		}
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

func listRoles(meta meta, dbFile string, args []string) {
	roleDB := getRoleDB(dbFile)
	roles := roleDB.GetRoles()
	for _, u := range roles {
		fmt.Println(u)
	}
	pluralS := "s"
	if len(roles) == 1 {
		pluralS = ""
	}
	fmt.Printf("%d role%s\n", len(roles), pluralS)
}

var cmds = []cmd{
	cmd{
		meta: meta{
			name:     "insert",
			desc:     "Insert role",
			argNames: []string{"role", "usernames*"},
		},
		f: insertRole,
	},
	cmd{
		meta: meta{
			name:     "delete",
			desc:     "Delete roles",
			argNames: []string{"roles*"},
		},
		f: deleteRoles,
	},
	cmd{
		meta: meta{
			name:     "list",
			desc:     "List roles",
			argNames: []string{},
		},
		f: listRoles,
	},
	cmd{
		meta: meta{
			name:     "create",
			desc:     "Create empty database",
			argNames: []string{},
		},
		f: createDB,
	},
	cmd{
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
	fmt.Fprintf(os.Stderr, "roles <dbfile> <command> <args>\n")
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
	if len(args) < 1 || args[0] == "help" {
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
