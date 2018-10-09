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
	"log"
	"os/exec"
	"syscall"

	"dutcontrol/agent"
)

const baseURL = "https://hexawolf.me/dutcontrol/api"

func longPoll(id, key string) {
	client := agent.NewClient(baseURL)
	client.SupportedTaskTypes = map[string]bool{
		"execute_cmd": true,
		"proclist":    true,
	}
	client.UseAccount(id, key)
	for {
		id, ttype, body, err := client.PollTasks()
		if err != nil {
			log.Println("Error during task polling:", err)
			if id != -1 {
				go client.SendTaskResult(id, map[string]interface{}{"error": true, "msg": err.Error()})
			}
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
		returnResult, err := out.Output()
		if err != nil {
			client.SendTaskResult(taskID, map[string]interface{}{"error": true, "msg": "Empty command is not allowed"})
		}

		client.SendTaskResult(taskID, map[string]interface{}{
			"status_code": out.ProcessState.Sys().(syscall.WaitStatus).ExitCode,
			"output":      string(returnResult),
		})

	case "proclist":
		log.Println("Received proclist task", body)
		windowsArray := ListWindows()
		responseMap := map[string]interface{}{
			"procs": windowsArray,
		}
		client.SendTaskResult(taskID, responseMap)
	}
}
