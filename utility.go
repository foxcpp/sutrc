package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
)

func removeAccountSubcommand() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "delaccount DBFILE NAME")
		os.Exit(1)
	}
	DBFile := os.Args[2]
	name := os.Args[3]
	db, err := OpenDB(DBFile)
	if err != nil {
		log.Fatalln("Failed to open DB:", err)
	}
	defer db.Close()
	err = db.RemAccount(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to remove account:", err)
		os.Exit(1)
	} else {
		fmt.Fprintln(os.Stderr, "OK.")
	}
}

func addAccountSubcommand() {
	if len(os.Args) != 5 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "addaccount DBFILE NAME TYPE")
		os.Exit(1)
	}
	DBFile := os.Args[2]
	name := os.Args[3]
	type_ := AccountType(0)
	switch os.Args[4] {
	case "agent":
		type_ = AcctAgent
	case "admin":
		type_ = AcctAdmin
	default:
		fmt.Fprintln(os.Stderr, "Invalid type. Use either 'admin' or 'agent'.")
		os.Exit(1)
	}
	// TODO: Hide entered password from console.
	fmt.Print("Enter password for account: ")
	passIn := bufio.NewScanner(os.Stdin)
	if !passIn.Scan() {
		log.Fatalln(passIn.Err())
	}
	pass := passIn.Text()
	db, err := OpenDB(DBFile)
	if err != nil {
		log.Fatalln("Failed to open DB:", err)
	}
	defer db.Close()
	err = db.AddAccount(name, pass, type_)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to add account:", err)
		os.Exit(1)
	} else {
		fmt.Fprintln(os.Stderr, "OK.")
	}
}
