/*
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
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/foxcpp/sutrc/agent"

	"golang.org/x/text/encoding/charmap"
)

var baseURL string

var cmdEncoding = charmap.CodePage866

func longPoll(hwid string) {
	client := agent.NewClient(baseURL)
	client.SupportedTaskTypes = map[string]bool{
		"execute_cmd": true,
		"proclist":    true,
		"downloadfile": true,
		"uploadfile": true,
		"dircontents": true,
	}
	client.UseAccount(hwid)
	for {
		id, ttype, body, err := client.PollTasks()
		if err != nil {
			log.Println("Error during task polling:", err)
			if id != -1 {
				go client.SendTaskResult(id, map[string]interface{}{"error": true, "msg": err.Error()})
			}
			time.Sleep(30*time.Second)
			continue
		}
		if id == -1 {
			continue
		}
		go executeTask(&client, id, ttype, body)
	}
}

func executeTask(client *agent.Client, taskID int, type_ string, body map[string]interface{}) {
	switch type_ {
	case "execute_cmd":
		executeCmdTask(client, taskID, body)
	case "proclist":
		proclistTask(client, taskID, body)
	case "downloadfile":
		downloadFileTask(client, taskID, body)
	case "uploadfile":
		uploadFileTask(client, taskID, body)
	case "dircontents":
		dirContentsTask(client, taskID, body)
	}
}

func uploadFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	path, ok := body["path"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "path should be string"})
		return
	}

	file, err := os.Open(path)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	url, err := client.UploadFile(file)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Upload fail: " + err.Error()})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{"error": false, "url": url})
}

func dirContentsTask(client *agent.Client, taskID int, body map[string]interface{}) {
	path, ok := body["path"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "path should be string"})
		return
	}

	dirInfo, err := ioutil.ReadDir(path)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	res := []map[string]interface{}{}
	for _, entry := range dirInfo {
		res = append(res, map[string]interface{}{
			"name": entry.Name(),
			"dir": entry.IsDir(),
			"fullpath": filepath.Join(path, entry.Name()),
		})
	}
	client.SendTaskResult(taskID, map[string]interface{}{"error": false, "contents": res})
}

func downloadFileTask(client *agent.Client, taskID int, body map[string]interface{}) {
	url, ok := body["url"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "url should be string"})
		return
	}
	outPath, ok := body["out"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "out should be string"})
		return
	}

	remoteFile, err := client.Download(url)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Download fail: " + err.Error()})
		return
	}

	file, err := os.Open(outPath)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}

	_, err = io.Copy(file, remoteFile)
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
		return
	}
	client.SendTaskResult(taskID, map[string]interface{}{"error": false})
}

func proclistTask(client *agent.Client, taskID int, _ map[string]interface{}) {
	procs, err := Processes()
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{
			"error": true,
			"msg":   err.Error(),
		})
		return
	}
	var windowsArray []Window
	for _, v := range procs {
		windowsArray = append(windowsArray, Window{
			PID:  v.Pid(),
			Name: v.Executable(),
		})
	}
	responseMap := map[string]interface{}{
		"procs": windowsArray,
	}
	client.SendTaskResult(taskID, responseMap)
}

func executeCmdTask(client *agent.Client, taskID int, body map[string]interface{}) {
	dec := cmdEncoding.NewDecoder()

	log.Println("Received execute_cmd task", body)
	command, ok := body["cmd"].(string)
	if !ok {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "cmd should be string"})
		return
	}
	if command == "" {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Empty command is not allowed"})
		return
	}

	out := exec.Command("cmd", "/C", command)
	returnResult, err := out.CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": err.Error()})
			return
		}
	}

	decodedOut, err := dec.String(string(returnResult))
	if err != nil {
		client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Can't convert output to Unicode"})
		return
	}

	client.SendTaskResult(taskID, map[string]interface{}{
		"status_code": out.ProcessState.Sys().(syscall.WaitStatus).ExitCode,
		"output":      decodedOut,
	})
}
