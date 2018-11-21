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
	"log"
	"os"
	"time"

	"github.com/denisbrodbeck/machineid"
	"github.com/foxcpp/sutrc/agent"
	"golang.org/x/sys/windows"
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

var baseURL string
var apiURL = baseURL + "/api"

func main() {
	client := agent.NewClient(apiURL)

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalln("Gailed to get hostname:", err)
	}
	fmt.Println("Hostname:", hostname)

	// Generating a fingerprint for this machine
	hwid, err := machineid.ProtectedID(hostname)

	if err := client.RegisterAgent(hostname, hwid); err != nil {
		log.Fatalf("failed to register on central server: %s", err)
	}

	log.Println("Starting longpolling")

	client.SupportedTaskTypes = []string{
		"execute_cmd",
		"proclist",
		"downloadfile",
		"uploadfile",
		"dircontents",
		"deletefile",
		"movefile",
		"screenshot",
		"update",
	}
	client.UseAccount(string(hwid))
	for {
		id, ttype, body, err := client.PollTasks()
		if err != nil {
			log.Println("Error during task polling:", err)
			if err.Error() == "access denied" {
				log.Println("Exiting!")
				os.Exit(1)
				return
			}
			if id != -1 {
				go client.SendTaskResult(id, map[string]interface{}{"error": true, "msg": err.Error()})
			}
			time.Sleep(30 * time.Second)
			continue
		}
		if id == -1 {
			continue
		}
		log.Println("Received task", body)
		switch ttype {
		case "execute_cmd":
			executeCmdTask(&client, id, body)
		case "proclist":
			proclistTask(&client, id, body)
		case "downloadfile":
			downloadFileTask(&client, id, body)
		case "uploadfile":
			uploadFileTask(&client, id, body)
		case "dircontents":
			dirContentsTask(&client, id, body)
		case "deletefile":
			deleteFileTask(&client, id, body)
		case "movefile":
			moveFileTask(&client, id, body)
		case "screenshot":
			screenshotTask(&client, id, body)
		case "update":
			selfUpdateTask(&client, id, body)

			// Golang have a very weird logic somewhere that prevents us from
			// leaving a running children process and terminate.
			// So basicallly we have to "hide" children from golang code by
			// calling CreateProcess directly.

			cmd, err := windows.UTF16PtrFromString(`C:\Windows\sutagent.exe`)
			if err != nil {
				panic(err)
			}
			si := windows.StartupInfo{}        // It's important to pass these structures
			pi := windows.ProcessInformation{} // otherwise it will fail.
			err = windows.CreateProcess(cmd, cmd, nil, nil, false, 0, nil, nil, &si, &pi)
			if err != nil {
				log.Println(err)
			} else {
				log.Println("Exiting")
				os.Exit(0)
			}
		}
	}

}
