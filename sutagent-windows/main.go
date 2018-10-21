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
	"github.com/denisbrodbeck/machineid"
	"github.com/foxcpp/sutrc/agent"
	"golang.org/x/sys/windows/svc"
	"io/ioutil"
	"log"
	"os"
	"strings"
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
	"allowing remote procedure execution as needed by sutserver according to internal protocol."

func installAgent(id string) error {
	// Generating a fingerprint for this machine
	// ID parameter is passed with install command
	mid, err := machineid.ProtectedID(id)
	if err != nil {
		return fmt.Errorf("failed generating machine ID: %s", err)
	}
	err = ioutil.WriteFile("C:\\Windows\\sutpc.key", []byte(mid), 0640)
	if err != nil {
		return err
	}

	client := agent.NewClient(apiURL)
	if err := client.RegisterAgent(id, mid); err != nil {
		log.Fatalln("Failed to register on central server:", err)
	}

	path, err := exePath()
	if err != nil {
		log.Fatalln("Failed to get program path:", err)
	}

	return agent.InstallService(path, svcname, dispName, description, 2)
}

func main() {
	// If we are running as an interactive session, we need to launch the service by itself.
	isInteractive, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if !isInteractive {
		// Assume we are running as a service and start polling server
		RunService(svcname, false)
		return
	}

	if len(os.Args) < 2 {
		usage("no command specified")
	}

	cmd := strings.ToLower(os.Args[1])
	switch cmd {
	case "debug":
		RunService(svcname, true)
		return
	case "install":
		if len(os.Args) < 3 {
			usage("invalid command usage")
		}
		hostname, err := os.Hostname()
		if err != nil {
			log.Fatalf("failed to generate HWID: %v", err)
		}
		err = installAgent(hostname)
	case "remove":
		err = agent.RemoveService(svcname)
	case "start":
		err = agent.StartService(svcname)
	case "stop":
		err = agent.ControlService(svcname, svc.Stop, svc.Stopped)
	default:
		usage(fmt.Sprintf("invalid command %s", cmd))
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcname, err)
	}
}
