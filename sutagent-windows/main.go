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
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

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

var baseURL string
var apiURL = baseURL + "/api"

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.LUTC)
	logFile, err := os.OpenFile("sutagent.log",
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0660)
	if err != nil {
		log.Fatalln("Unable to configure logging:", err)
	}
	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)

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
	log.Println("Starting longpolling")

	client := agent.NewClient(apiURL)
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
			logFile.Close()
			out, err := exec.Command(os.Args[0]).Output()
			if err != nil {
				fmt.Println(string(out))
				panic(err)
			}
			return
		}
	}

}
