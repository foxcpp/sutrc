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
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/foxcpp/filedrop"
	"gopkg.in/yaml.v2"
)

const PathPrefix = "/sutrc/api"

var db *DB
var agentsSelfregEnabled = false
var onlineAgents = make(map[string]bool)
var onlineAgentsLock sync.Mutex
var lastRequestStamp = make(map[string]time.Time)
var lastRequestStampLock sync.Mutex

func main() {
	if len(os.Args) == 1 {
		fmt.Println(os.Args[0], "server CONFIGFILE")
		fmt.Println("\tLaunch server with configuration from CONFIGFILE.")
		fmt.Println(os.Args[0], "addaccount CONFIGFILE TOKEN")
		fmt.Println("\tAdd account token TOKEN to server DB from CONFIGFILE.")
		fmt.Println(os.Args[0], "remaccount CONFIGFILE TOKEN")
		fmt.Println("\tRemove account token TOKEN from server DB from CONFIGFILE.")
		fmt.Println(os.Args[0], "addagent CONFIGFILE NAME HWID")
		fmt.Println("\tAdd agent NAME with HWID to server DB from CONFIGFILE.")
		fmt.Println(os.Args[0], "remagent CONFIGFILE NAME")
		fmt.Println("\tRemove agent NAME from server DB from CONFIGFILE.")
		return
	}

	subCmd := os.Args[1]
	switch subCmd {
	case "server":
		serverSubcommand()
	case "addaccount":
		addAccountSubcmd()
	case "remaccount":
		remAccountSubcmd()
	case "addagent":
		addAgentSubcmd()
	case "remagent":
		remAgentSubcmd()
	default:
		fmt.Fprintln(os.Stderr, "Unknown subcommand.")
		os.Exit(1)
	}
}

// Can be set using `go build -ldflags "-X main.debugLog="true""`
var debug string

func debugLog(v ...interface{}) {
	if debug == "true" { // -X flag can't handle non-string types
		log.Println(v...)
	}
}

func serverSubcommand() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "server CONFIGFILE")
		os.Exit(1)
	}

	if os.Getenv("USING_SYSTEMD") == "1" {
		// Don't print timestamp in log because journald captures it anyway.
		log.SetFlags(0)
	}

	confBlob, err := ioutil.ReadFile(os.Args[2])
	if err != nil {
		log.Fatalln("Failed to read config file:", err)
	}
	conf := Config{}
	if err := yaml.Unmarshal(confBlob, &conf); err != nil {
		log.Fatalln("Failed to parse config file:", err)
	}

	db, err = OpenDB(conf.DB.Driver, conf.DB.DSN)
	if err != nil {
		log.Fatalln("Failed to open DB:", err)
	}
	defer db.Close()

	conf.Filedrop.DB.Driver = conf.DB.Driver
	conf.Filedrop.DB.DSN = conf.DB.DSN
	filedropSrv := startFiledrop(conf.Filedrop)
	defer filedropSrv.Close()

	http.HandleFunc(PathPrefix+"/tasks", tasksHandler)
	http.HandleFunc(PathPrefix+"/task_result", tasksResultHandler)
	http.HandleFunc(PathPrefix+"/login", loginHandler)
	http.HandleFunc(PathPrefix+"/logout", logoutHandler)
	http.HandleFunc(PathPrefix+"/agents", agentsHandler)
	http.HandleFunc(PathPrefix+"/agents_selfreg", agentsSelfregHandler)
	http.Handle(PathPrefix+"/filedrop/", filedropSrv)

	go func() {
		log.Println("Listening on", conf.ListenOn)
		if err := http.ListenAndServe(conf.ListenOn, nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if os.Getenv("USING_SYSTEMD") == "1" {
		cmd := exec.Command("systemd-notify", "--ready", `--status=Listening on `+conf.ListenOn)
		if out, err := cmd.Output(); err != nil {
			log.Println("Failed to notify systemd about successful startup:", err)
			log.Println(string(out))
		}
	}

	// Handle Ctrl-C and stuff gracefully.
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
	<-sig
}

func startFiledrop(conf filedrop.Config) *filedrop.Server {
	conf.UploadAuth.Callback = func(r *http.Request) bool {
		if checkAdminAuth(r.Header) || checkAgentAuth(r.Header) {
			return true
		}
		cookie, err := r.Cookie("sutcp_session")
		if err != nil {

			return false
		}
		return db.CheckSession(cookie.Value)
	}
	conf.DownloadAuth.Callback = conf.UploadAuth.Callback
	if err := os.MkdirAll(conf.StorageDir, 0777); err != nil {
		log.Fatalln("Failed to create filedrop storage dir:", err)
	}
	filedropSrv, err := filedrop.New(conf)
	if err != nil {
		log.Fatalln("Failed to start filedrop:", err)
	}
	return filedropSrv
}

func agentsSelfregHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r.Header) {
		writeError(w, http.StatusForbidden, "Authorization failure")
		return
	}

	if r.Method == http.MethodPost {
		enabled := r.URL.Query().Get("enabled")
		switch enabled {
		case "1":
			agentsSelfregEnabled = true
		case "0":
			agentsSelfregEnabled = false
		default:
			writeError(w, http.StatusBadRequest, "Pass 'enabled=1' or 'enabled=0' in query string")
		}
	} else if r.Method == http.MethodGet {
		w.WriteHeader(http.StatusOK)
		if agentsSelfregEnabled {
			w.Write([]byte("1"))
		} else {
			w.Write([]byte("0"))
		}
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/agents_selfreg only supports POST and GET")
	}
}

func agentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		agentSelfreg(w, r)
	} else if r.Method == http.MethodGet {
		agentListHandler(w, r)
	} else if r.Method == http.MethodPatch {
		renameAgentHandler(w, r)
	} else if r.Method == http.MethodDelete {
		deregAgent(w, r)
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/agents only supports POST, GET, PATCH and DELETE")
	}

}

func deregAgent(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r.Header) {
		writeError(w, http.StatusForbidden, "Authorization failure")
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "Pass 'id' in query string")
		return
	}

	removeAgentQueues(id)
	if err := db.RemAgent(id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
	}
}

func removeAgentQueues(id string) {
	taskMetaLock.Lock()
	for _, v := range taskResults[id] {
		close(v)
	}
	delete(taskResults, id)

	if _, prs := tasks[id]; prs {
		close(tasks[id])
	}
	delete(tasks, id)
	taskMetaLock.Unlock()

	lastRequestStampLock.Lock()
	delete(lastRequestStamp, id)
	lastRequestStampLock.Unlock()

	onlineAgentsLock.Lock()
	delete(onlineAgents, id)
	onlineAgentsLock.Unlock()
}

func agentSelfreg(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	hwid := r.URL.Query().Get("hwid")
	if name == "" || hwid == "" {
		writeError(w, http.StatusBadRequest, "Pass 'name' and 'hwid' in query string")
		return
	}

	if db.CheckAgentAuth(hwid) {
		return
	}

	if !agentsSelfregEnabled {
		writeError(w, http.StatusMethodNotAllowed, "Agents self-registration is disabled")
		return
	}

	if err := db.AddAgent(name, hwid); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
}

// StringSlice is a sort.Interface implementation that considers longer strings
// to be "bigger", which makes comparsion work just like numeric comparsion
// when string consists of numbers.
type StringSlice []string

func (s StringSlice) Len() int {
	return len(s)
}

func (s StringSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s StringSlice) Less(i, j int) bool {
	if len(s[i]) != len(s[j]) {
		return len(s[i]) < len(s[j])
	}
	return s[i] < s[j]
}

func agentListHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r.Header) {
		writeError(w, http.StatusForbidden, "Authorization failure")
		return
	}

	agents, err := db.ListAgents()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	sort.Sort(StringSlice(agents))

	lastRequestStampLock.Lock()
	onlineAgentsLock.Lock()
	defer lastRequestStampLock.Unlock()
	defer onlineAgentsLock.Unlock()

	onlineAgentsL := make(map[string]bool)
	for _, agent := range agents {
		if !lastRequestStamp[agent].IsZero() {
			// 28 seconds = longpolling interval + 2 seconds for possible delay
			// due to agent lags.
			onlineAgentsL[agent] = onlineAgents[agent] || (time.Now().Sub(lastRequestStamp[agent]) < 28*time.Second)
		} else {
			onlineAgentsL[agent] = onlineAgents[agent]
		}
	}

	writeJson(w, map[string]interface{}{"error": false, "agents": agents, "online": onlineAgentsL})
}

func renameAgentHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r.Header) {
		writeError(w, http.StatusForbidden, "Authorization failure")
		return
	}

	oldId := r.URL.Query().Get("id")
	newId := r.URL.Query().Get("newId")
	if oldId == "" || newId == "" {
		writeError(w, http.StatusBadRequest, "Pass 'id' and 'oldId' in query string.")
		return
	}

	if !db.AgentExists(oldId) {
		writeError(w, http.StatusNotFound, "Agent doesn't exists")
		return
	}

	if err := db.RenameAgent(oldId, newId); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	taskMetaLock.Lock()
	taskResults[newId] = taskResults[oldId]
	delete(taskResults, oldId)
	tasks[newId] = tasks[oldId]
	delete(tasks, oldId)
	taskMetaLock.Unlock()

	lastRequestStampLock.Lock()
	lastRequestStamp[newId] = lastRequestStamp[oldId]
	delete(lastRequestStamp, oldId)
	lastRequestStampLock.Unlock()

	onlineAgentsLock.Lock()
	onlineAgents[newId] = onlineAgents[oldId]
	delete(onlineAgents, oldId)
	onlineAgentsLock.Unlock()
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if !db.CheckAuth(r.URL.Query().Get("token")) {
		log.Println("Invalid login info submitted from", r.RemoteAddr, "("+r.Header.Get("X-Real-IP")+")")
		writeError(w, http.StatusForbidden, "Invalid credentials")
		return
	}

	token, err := db.InitSession()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Println("Initialized session with token=" + token[:6] + "...")
	writeJson(w, map[string]interface{}{"error": false, "token": token})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "" {
		writeError(w, http.StatusBadRequest, "Missing Authorization header")
		return
	}

	log.Println("Killed session with token=" + r.Header.Get("Authorization")[:6] + "...")
	db.KillSession(r.Header.Get("Authorization"))
	writeJson(w, map[string]interface{}{"error": false, "msg": "Logged out"})
}

func writeJson(w http.ResponseWriter, in interface{}) {
	w.Header().Set("Content-Type", "application/json")
	buf, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	w.Write(buf)
}

func writeError(w http.ResponseWriter, httpCode int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpCode)
	buf, err := json.Marshal(map[string]interface{}{"error": true, "msg": msg})
	if err != nil {
		panic(err)
	}
	w.Write(buf)
}

func checkAgentAuth(h http.Header) bool {
	return db.CheckAgentAuth(h.Get("Authorization"))
}

func checkAdminAuth(h http.Header) bool {
	return db.CheckSession(h.Get("Authorization"))
}
