/* MIT License
 *
 * Copyright (c) 2018  Max Mazurov (fox.cpp) and Vladyslav Yamkovyi (Hexawolf)
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
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var taskResults = make(map[string]map[int]chan map[string]interface{})

// tasks map stores queue of tasks per agent,
// note that if there is empty map object - it should be ignored,
// because it is used for "cancelled" tasks
var tasks = make(map[string]chan map[string]interface{})
var nextTaskID = 1

// Should be locked if any variables above (except channel I/O) are accessed.
var taskMetaLock sync.Mutex

func tasksResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !checkAgentAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		agentID, err := db.GetAgentName(r.Header.Get("Authorization"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		lastRequestStampLock.Lock()
		lastRequestStamp[agentID] = time.Now()
		lastRequestStampLock.Unlock()

		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil {
			return
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		bodyJson := make(map[string]interface{})
		if err := json.Unmarshal(body, &bodyJson); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON passed in body")
			return
		}

		if _, prs := bodyJson["error"]; !prs {
			bodyJson["error"] = false
		}

		debugLog("Received task", id, "result from", agentID)

		// taskResults[agentID] is created on task submit if it doesn't exists.
		taskMetaLock.Lock()
		c := taskResults[agentID][id]
		taskMetaLock.Unlock()
		if c == nil {
			// If channel doesn't exists - nobody is waiting for task result. Just drop it.
			debugLog("Unexpected task", id, "result from", agentID)
			return
		}
		c <- bodyJson
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/tasks_result supports only POST")
	}
}

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !checkAdminAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		acceptTask(w, r)
	} else if r.Method == http.MethodGet {
		if !checkAgentAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		// 26 seconds seems to be reasonable choice even with presence of VPNs
		// and proxies.
		tasksLongpool(w, r, time.Second*26)
	} else {
		writeError(w, http.StatusBadRequest, "/tasks endpoint supports only GET and POST")
	}
}

func acceptTask(w http.ResponseWriter, r *http.Request) {
	targetsStr := r.URL.Query().Get("target")
	if targetsStr == "" {
		writeError(w, http.StatusBadRequest, "Missing target parameter")
		return
	}
	timeoutStr := r.URL.Query().Get("timeout")
	timeout := 26 * time.Second
	if timeoutStr != "" {
		secs, err := strconv.Atoi(timeoutStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Invalid timeout value")
			return
		}
		timeout = time.Duration(secs) * time.Second
	}

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	task := map[string]interface{}{}
	if err := json.Unmarshal(buf, &task); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}

	if _, prs := task["type"]; !prs {
		writeError(w, http.StatusBadRequest, "Task type missing")
		return
	}

	targets := strings.Split(targetsStr, ",")
	responses := make([]map[string]interface{}, len(targets))
	taskCopies := make([]map[string]interface{}, len(targets))
	for i, target := range targets {
		taskCpy := make(map[string]interface{})
		for k, v := range task {
			taskCpy[k] = v
		}
		taskCopies[i] = taskCpy

		if !db.AgentExists(target) {
			responses[i] = map[string]interface{}{"error": true, "msg": "Agent doesn't exists"}
		}

		taskMetaLock.Lock()

		// This can be first time we see this Agent ID, allocate everything we need.
		if _, prs := tasks[target]; !prs {
			// Leave enough space to buffer few tasks in case of
			// "lagging" agent of network.
			tasks[target] = make(chan map[string]interface{}, 16)
		}
		if _, prs := taskResults[target]; !prs {
			taskResults[target] = make(map[int]chan map[string]interface{})
		}

		// "Allocate" task ID.
		id := nextTaskID
		nextTaskID++
		taskCopies[i]["id"] = id

		// Prepare storage for result.
		taskResults[target][id] = make(chan map[string]interface{})

		tasksChan := tasks[target]
		taskMetaLock.Unlock()

		taskCpy["id"] = id
		buf, err = json.Marshal(task)
		if err != nil {
			responses[i] = map[string]interface{}{"error": true, "msg": "Internal error: " + err.Error()}
			return
		}

		select {
		case tasksChan <- taskCpy:
		default:
			responses[i] = map[string]interface{}{"error": true, "msg": "Queue is overflowed. Check agent."}
			return
		}

		debugLog("Added task", id, "for", target, "from", r.Header.Get("Authorization")[:6])
	}

	for i, target := range targets {
		if responses[i] == nil {
			responses[i] = waitTaskResult(target, taskCopies[i], r, timeout)
			responses[i]["target"] = target
		}
	}

	writeJson(w, map[string]interface{}{"error": false, "results": responses})
}

func waitTaskResult(agentID string, task map[string]interface{}, r *http.Request, timeout time.Duration) map[string]interface{} {
	taskID := task["id"].(int)

	taskMetaLock.Lock()
	taskResChan := taskResults[agentID][taskID]
	taskMetaLock.Unlock()
	select {
	case res, ok := <-taskResChan:
		if !ok {
			return map[string]interface{}{"error": true, "msg": "Agent deregistered"}
		}
		debugLog("Forwarding task", taskID, "result from", agentID, "to", r.Header.Get("Authorization")[:6])
		return res
	case <-time.After(timeout):
		debugLog("Timed out while waiting for task", taskID, "result from", agentID)
		taskMetaLock.Lock()
		delete(taskResults[agentID], taskID)

		// Mark task as cancelled by clearing it.
		for k, _ := range task {
			delete(task, k)
		}
		taskMetaLock.Unlock()
		return map[string]interface{}{"error": true, "msg": "Time out while waiting for task result"}
	}
	return nil
}

func tasksLongpool(w http.ResponseWriter, r *http.Request, timeout time.Duration) {
	agentID, err := db.GetAgentName(r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	taskMetaLock.Lock()
	// This can be first time we see this Agent ID, allocate everything we need..
	if _, prs := tasks[agentID]; !prs {
		// Leave enough space to buffer few tasks in case of
		// "lagging" agent of network.
		tasks[agentID] = make(chan map[string]interface{}, 16)
	}
	if _, prs := taskResults[agentID]; !prs {
		taskResults[agentID] = make(map[int]chan map[string]interface{})
	}
	tasksChan := tasks[agentID]
	taskMetaLock.Unlock()

	lastRequestStampLock.Lock()
	lastRequestStamp[agentID] = time.Now()
	lastRequestStampLock.Unlock()

	// We 'register' agent as online only if it listens for tasks.
	// Agent is expected to handle them asynchronously so it will be
	// listening most of the time.
	onlineAgentsLock.Lock()
	onlineAgents[agentID] = true
	onlineAgentsLock.Unlock()
	defer func() {
		onlineAgentsLock.Lock()
		onlineAgents[agentID] = false
		onlineAgentsLock.Unlock()
	}()

	debugLog(agentID, "is watching for tasks")

	for {
		select {
		case <-time.After(timeout):
			writeJson(w, map[string]interface{}{})
		case task, ok := <-tasksChan:
			if !ok {
				writeError(w, http.StatusForbidden, "Agent deregistered")
				return
			}
			// Ignore "cancelled" tasks.
			if len(task) == 0 {
				continue
			}
			debugLog("Sending task", task["id"].(int), "to", agentID)
			writeJson(w, task)
		}
		return
	}
}
