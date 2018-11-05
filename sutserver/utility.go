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
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

func openDBFromConf(configFile string) (*DB, error) {
	confBlob, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		return nil, err
	}
	conf := Config{}
	if err := yaml.Unmarshal(confBlob, &conf); err != nil {
		return nil, err
	}
	db, err = OpenDB(conf.DB.Driver, conf.DB.DSN)
	return db, err
}

func addAgentSubcmd() {
	if len(os.Args) != 5 {
		fmt.Println("Usage:", os.Args[0], "addagent CONFIGFILE NAME HWID")
		os.Exit(2)
	}
	db, err := openDBFromConf(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	name := os.Args[3]
	hwid := os.Args[4]

	if err := db.AddAgent(name, hwid); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	} else {
		fmt.Println("OK!")
	}
}

func remAgentSubcmd() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:", os.Args[0], "remagent CONFIGFILE NAME")
		os.Exit(2)
	}
	db, err := openDBFromConf(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
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
		fmt.Println("Usage:", os.Args[0], "addaccount DRIVER=DSN TOKEN")
		os.Exit(2)
	}
	db, err := openDBFromConf(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	token := os.Args[3]

	if err := db.AddAccount(token); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	} else {
		fmt.Println("OK!")
	}
}

func remAccountSubcmd() {
	if len(os.Args) != 4 {
		fmt.Println("Usage:", os.Args[0], "remaccount CONFIGFILE TOKEN")
		os.Exit(2)
	}
	db, err := openDBFromConf(os.Args[2])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer db.Close()
	token := os.Args[3]

	if err := db.RemAccount(token); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	} else {
		fmt.Println("OK!")
	}
}
