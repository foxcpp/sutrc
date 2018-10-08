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
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

const baseURL = "https://hexawolf.me/dutcontrol/api"

func processResponse(resp *http.Response, client *http.Client) {
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Unable to get entire packet:", err)
		return
	}
	var body map[string]interface{}
	err = json.Unmarshal(bodyBytes, &body)
	if err != nil {
		elog.Warning(1, "Invalid packet received: " + err.Error())
		return
	}
	resp.Body.Close()
	taskType := body["type"].(string)
	var response string
	switch taskType {
	case "execute_cmd":
		command := body["cmd"].(string)
		if command == "" {
			response = "Empty command is not allowed!"
		}
		out := exec.Command("cmd", "/C", command)
		returnResult, err := out.Output()
		if err != nil {
			response = err.Error()
		}
		response = string(returnResult)
		break
	case "proclist":
		out := exec.Command("cmd", "/C", "tasklist /V")
		returnResult, err := out.Output()
		if err != nil {
			response = err.Error()
		}
		taskListStruct := make(map[string][]string)
		scanner := bufio.NewScanner(strings.NewReader(string(returnResult)))
		// Skip 3 lines
		scanner.Scan()
		scanner.Scan()
		scanner.Scan()
		for scanner.Scan() {

		}
		//taskListStruct["procs"] =
		response = string(returnResult)
		break
	default:
		response = "Invalid command"
	}
	http.NewRequest("POST", baseURL + "/task_result?id=" + body["id"].(string), strings.NewReader(response))
}

func longPoll(id, key string) {
	client := &http.Client{
		Timeout: 26,
	}

	for {
		req, err := http.NewRequest("GET", baseURL + "/tasks", nil)
		req.Header.Add("Authorization", id + ":" + key)
		resp, err := client.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(30 * time.Second)
			continue
		}
		go processResponse(resp, client)
	}
}
