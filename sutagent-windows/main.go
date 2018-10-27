/* MIT License
 *
 * Copyright (c) 2018  Vladyslav Yamkovyi (Hexawolf)
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
	"log"
	"os"
	"strings"

	"github.com/denisbrodbeck/machineid"
	"github.com/foxcpp/sutrc/agent"
)

func usage(errmsg string) {
	fmt.Fprintf(os.Stderr,
		"%s\n\n"+
			"usage: %s <command>\n"+
			"       where <command> is one of\n"+
			"       install, remove, debug, start, stop.\n",
		errmsg, os.Args[0])
	os.Exit(2)
}

const svcname = "sutagent"
const dispName = "State University of Telecommunications Remote Control Service Agent"
const description = "Implements remote control functionality and performs background longpolling, " +
	"."

func main() {

	if len(os.Args) >= 2 {
		cmd := strings.ToLower(os.Args[1])
		switch cmd {
		case "debug":
			break
		case "install":
			fmt.Println("Installing service")
			hostname, err := os.Hostname()
			if err != nil {
				log.Fatalf("failed to get hostname: %v", err)
			}
			fmt.Println("Hostname:", hostname)
			// Generating a fingerprint for this machine
			// ID parameter is passed with install command
			mid, err := machineid.ProtectedID(hostname)
			if err != nil {
				log.Fatalf("failed generating machine ID: %s", err)
			}
			fmt.Println("HWID:", mid)
			err = ioutil.WriteFile("C:\\Windows\\sutpc.key", []byte(mid), 0640)
			if err != nil {
				log.Fatalf("failed to save a key for this PC: %s", err)
			}
			fmt.Println("Trying to register client")
			client := agent.NewClient(apiURL)
			if err := client.RegisterAgent(hostname, mid); err != nil {
				log.Fatalf("failed to register on central server: %s", err)
			}
			return
		default:
			usage(fmt.Sprintf("invalid command %s", cmd))
		}
	}

	hwid, err := ioutil.ReadFile("C:\\Windows\\sutpc.key")
	if err != nil {
		log.Fatalln("Failed to read authorization key:", err)
	}
	longPoll(string(hwid))
}
