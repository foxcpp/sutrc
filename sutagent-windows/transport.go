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
	"log"
	"os"
	"time"

	"github.com/foxcpp/sutrc/agent"
)

var baseURL string
var apiURL = baseURL + "/api"

func longPoll(hwid string) {
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
	client.UseAccount(hwid)
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
		go executeTask(&client, id, ttype, body)
	}
}

func executeTask(client *agent.Client, taskID int, type_ string, body map[string]interface{}) {
	log.Println("Received task", body)
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
	case "deletefile":
		deleteFileTask(client, taskID, body)
	case "movefile":
		moveFileTask(client, taskID, body)
	case "screenshot":
		screenshotTask(client, taskID, body)
	case "update":
		selfUpdateTask(client, taskID, body)
	}
}
