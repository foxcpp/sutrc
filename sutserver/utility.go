package main

import (
	"fmt"
	"os"
)

func addAgentSubcmd() {
	if len(os.Args) != 5 {
		fmt.Println("Usage:", os.Args[0], "addagent DBFILE NAME HWID")
		return
	}
	db, err := OpenDB(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open DB:", err)
		return
	}
	defer db.Close()
	name := os.Args[3]
	hwid := os.Args[4]

	if err := db.AddAgent(name, hwid); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("OK!")
	}
}

func remAgentSubcmd() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:", os.Args[0], "remagent DBFILE NAME")
		return
	}
	db, err := OpenDB(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open DB:", err)
		return
	}
	defer db.Close()
	name := os.Args[3]

	if err := db.RemAgent(name); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("OK!")
	}
}

func addAccountSubcmd() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:", os.Args[0], "addaccount DBFILE TOKEN")
	}
	db, err := OpenDB(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open DB:", err)
		return
	}
	defer db.Close()
	token := os.Args[3]

	if err := db.AddAccount(token); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("OK!")
	}
}

func remAccountSubcmd() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:", os.Args[0], "remaccount DB TOKEN")
		return
	}
	db, err := OpenDB(os.Args[2])
	if err != nil {
		fmt.Println("Failed to open DB:", err)
		return
	}
	defer db.Close()
	token := os.Args[3]

	if err := db.RemAccount(token); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Println("OK!")
	}
}
