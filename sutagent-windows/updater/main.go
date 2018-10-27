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
	"github.com/foxcpp/sutrc/agent"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const svcname = "sutupdate"
const dispName = "State University of Telecommunications Agent Updater"
const description = "Performs agent executable file rotation for sutrc by server request, " +
	"allowing remote deployment of updates."

var elog debug.Log

type sutService struct{}

var baseURL string

func exePath() (string, error) {
	prog := os.Args[0]
	p, err := filepath.Abs(prog)
	if err != nil {
		return "", err
	}
	fi, err := os.Stat(p)
	if err == nil {
		if !fi.Mode().IsDir() {
			return p, nil
		}
		err = fmt.Errorf("%s is directory", p)
	}
	if filepath.Ext(p) == "" {
		p += ".exe"
		fi, err := os.Stat(p)
		if err == nil {
			if !fi.Mode().IsDir() {
				return p, nil
			}
			err = fmt.Errorf("%s is directory", p)
		}
	}
	return "", err
}

func (m *sutService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	changes <- svc.Status{State: svc.StartPending, Accepts: svc.AcceptStop | svc.AcceptShutdown}
	client := agent.NewClient(baseURL)
	const executable = "C:\\Windows\\sutagent.exe"
	out, err := os.Create(executable)
	if err != nil  {
		elog.Error(1, "Failed to download sutagent update: " + err.Error())
		return
	}
	defer out.Close()

	inp, err := client.Download(baseURL + "/sutagent.exe")
	if err != nil  {
		elog.Error(1, "Failed to download sutagent update: " + err.Error())
		return
	}

	_, err = io.Copy(out, inp)
	if err != nil  {
		elog.Error(1, "Failed to download sutagent update: " + err.Error())
		return
	}

	changes <- svc.Status{State: svc.Running}

	agent.StartService("sutagent")

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			changes <- svc.Status{State: svc.StopPending}
			return
		default:
			elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
		}
	}
}

func RunService(name string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &sutService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
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

	exe, err := exePath()

	if err != nil {
		log.Fatalln("Failed to find exe path:", err)
	}

	arg := ""
	if len(os.Args) >= 2 {
		arg = os.Args[1]
	}
	cmd := strings.ToLower(arg)
	switch cmd {
	case "install":
		err = agent.InstallService(exe, svcname, dispName, description, 3)
	case "remove":
		err = agent.RemoveService(svcname)
	case "start":
		err = agent.StartService(svcname)
	case "stop":
		err = agent.ControlService(svcname, svc.Stop, svc.Stopped)
	default:
		RunService(svcname, true)
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, svcname, err)
	}
}
