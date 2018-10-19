/* MIT License
 *
 * Copyright (c) 2018  Max Mazurov (fox.cpp) and Vladyslav Yamkovyi (Hexawolf)
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */
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
