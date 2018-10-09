/* MIT License
 *
 * Copyright (c) 2018 Vladyslav Yamkovyi (Hexawolf)
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
	"encoding/json"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var elog debug.Log

type dutService struct{}

func runService(name string, isDebug bool) {
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
	err = run(name, &dutService{})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}

func startService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	err = s.Start("is", "auto-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}

	return nil
}

func controlService(name string, c svc.Cmd, to svc.State) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()
	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(500 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}

	return nil
}

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

func installService(name, desc, id string) error {
	// Generating a fingerprint for this machine
	// ID parameter is passed with install command
	mid, err := machineid.ProtectedID(id)
	if err != nil {
		return fmt.Errorf("failed generating machine ID: %s", err)
	}
	d1 := []byte(id + " " + mid)
	err = ioutil.WriteFile("C:\\Windows\\dutpc.key", d1, 0640)
	if err != nil {
		return err
	}

	// Register this computer on the central server
	midSplit := strings.Split(id,"-")
	numID, err := strconv.Atoi(midSplit[1])
	if err != nil {
		log.Fatalln("Invalid ID:", err)
	}
	room, err := strconv.Atoi(midSplit[0])
	if err != nil {
		log.Fatalln("Invalid ID:", err)
	}
	d1 = append(d1, byte(' '))
	d1 = append(d1, byte(numID))
	d1 = append(d1, byte(' '))
	d1 = append(d1, byte(room))

	client := &http.Client{
		Timeout: 26 * time.Second,
	}
	res, err := client.Get(baseURL + "/agent_selfreg")
	if err != nil {
		log.Fatalln("Cannot contact server to perform self-registration:", err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatalln("Failed to read server response to GET request:", err)
	}
	if string(body) != "1" {
		log.Fatalln("Server does not accepts new agents right now.")
	}
	req, err := http.NewRequest("POST", baseURL + "/agents?user=" + id + "&pass=" + mid, nil)
	if err != nil {
		log.Fatalln("Failed to construct POST request:", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln("Cannot contact server to perform self-registration:", err)
	}
	if resp.StatusCode != 200 {
		// We have two possible causes for non-200 code:
		// - Error at dutserver level
		// - Error at intermediate server (nginx)
		//   Likely this means that dutserver itself is down.
		// In second case response body will not contain JSON so we
		// can only use StatusCode.

		if resp.Header.Get("Content-Type") == "application/json" {
			respBody, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Fatalln("Failed to read server response body:", err)
			}
			jsonErr := make(map[string]interface{})
			if err := json.Unmarshal(respBody, &jsonErr); err != nil {
				log.Fatalln("Failed to parse JSON server response:", err)
			}
			log.Fatalln("Self-registration request rejected by server:", jsonErr["msg"])
		} else {
			log.Fatalln("Failed to contact server to perform self-registration:", resp.Status)
		}
	}

	exepath, err := exePath()
	if err != nil {
		return err
	}
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}
	s, err = m.CreateService(name, exepath, mgr.Config{DisplayName: desc}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()
	err = eventlog.InstallAsEventCreate(name, eventlog.Error|eventlog.Warning|eventlog.Info)
	if err != nil {
		s.Delete()
		return fmt.Errorf("SetupEventLogSource() failed: %s", err)
	}

	return nil
}

func removeService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()
	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()
	err = s.Delete()
	if err != nil {
		return err
	}
	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}

func (m *dutService) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown
	changes <- svc.Status{State: svc.StartPending}
	data, err := ioutil.ReadFile("C:\\Windows\\dutpc.key")
	if err != nil {
		const authKeyErr = "Failed to read authorization key:"
		elog.Error(1, authKeyErr + " " + err.Error())
		log.Fatalln(authKeyErr, err)
	}
	mid := strings.Split(string(data), " ")
	go longPoll(mid[0], mid[1])
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
	elog.Info(1, "")

	for {
		c := <-r
		switch c.Cmd {
		case svc.Interrogate:
			changes <- c.CurrentStatus
		case svc.Stop, svc.Shutdown:
			break
		default:
			elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}
