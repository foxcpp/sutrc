package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

var taskResults map[string]map[int]chan map[string]interface{}
var tasks map[string]chan map[string]interface{}
var nextTaskID = 1

// Should be locked if any variables above (except channel I/O) are accessed.
var taskMetaLock sync.Mutex

func tasksResultHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if !checkAgentAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}
		agentID := strings.Split(r.Header.Get("Authorization"), ":")[0]
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

		// taskResults[agentID] is created on task submit if it doesn't exists.
		taskMetaLock.Lock()
		c := taskResults[agentID][id]
		taskMetaLock.Unlock()
		if c == nil {
			// If channel doesn't exists - nobody is waiting for task result. Just drop it.
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
	target := r.URL.Query().Get("target")
	if target == "" {
		writeError(w, http.StatusBadRequest, "Missing target parameter")
		return
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

	// Prepare storage for result.
	taskResults[target][id] = make(chan map[string]interface{})

	tasksChan := tasks[target]
	taskMetaLock.Unlock()

	task["id"] = id
	buf, err = json.Marshal(task)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	select {
	case tasksChan <- task:
	default:
		writeError(w, http.StatusServiceUnavailable, "Task queue is overflowed. Check agent.")
		return
	}

	log.Println("Added task", id, "for", target, "from", r.Header.Get("Authorization")[:6])
	waitTaskResult(w, target, id, r, 5*time.Second)
}

func waitTaskResult(w http.ResponseWriter, agentID string, taskID int, r *http.Request, timeout time.Duration) {
	taskMetaLock.Lock()
	taskResChan := taskResults[agentID][taskID]
	taskMetaLock.Unlock()
	select {
	case <-time.After(timeout):
		log.Println("Timed out while waiting for task", taskID, "result from", agentID)
		writeError(w, http.StatusGatewayTimeout, "Time out while waiting for task result")
		taskMetaLock.Lock()
		delete(taskResults[agentID], taskID)
		taskMetaLock.Unlock()
	case res := <-taskResChan:
		log.Println("Forwarding task", taskID, "result from", agentID, "to", r.Header.Get("Authorization")[:6])
		writeJson(w, res)
	}
}

func tasksLongpool(w http.ResponseWriter, r *http.Request, timeout time.Duration) {
	agentID := strings.Split(r.Header.Get("Authorization"), ":")[0]

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
	taskMetaLock.Unlock()

	// We 'register' agent as online only if it listens for tasks.
	// Agent is expected to handle them asynchronously so it will be
	// listening most of the time.
	onlineAgents[agentID] = true

	log.Println(agentID, "is watching for tasks")

	taskMetaLock.Lock()
	tasksChan := tasks[agentID]
	taskMetaLock.Unlock()

	select {
	case <-time.After(timeout):
		writeJson(w, map[string]interface{}{})
		onlineAgents[agentID] = false
	case task := <-tasksChan:
		log.Println("Sending task", task["id"].(int), "to", agentID)
		writeJson(w, task)
		onlineAgents[agentID] = false
	}
}
