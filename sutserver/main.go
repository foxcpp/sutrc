package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
)

var db *DB
var agentsSelfregEnabled = false
var onlineAgents map[string]bool
var onlineAgentsLock sync.Mutex

func main() {
	if len(os.Args) == 1 {
		fmt.Println(os.Args[0], "server PORT DBFILE")
		fmt.Println("\tLaunch server on PORT using DBFILE.")
		fmt.Println(os.Args[0], "addaccount DBFILE TOKEN")
		fmt.Println("\tAdd with token TOKEN to DBFILE.")
		fmt.Println(os.Args[0], "remaccount DBFILE TOKEN")
		fmt.Println("\tRemove account with TOKEN from DBFILE.")
		fmt.Println(os.Args[0], "addagent DBFILE NAME HWID")
		fmt.Println("\tAdd agent NAME with HWID to DBFILE.")
		fmt.Println(os.Args[0], "remagent DBFILE NAME")
		fmt.Println("\tRemove agent NAME from DBFILE.")
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

func serverSubcommand() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "server PORT DBFILE")
		os.Exit(1)
	}
	port := os.Args[2]
	DBFile := os.Args[3]

	if os.Getenv("USING_SYSTEMD") == "1" {
		// Don't print timestamp in log because journald captures it anyway.
		log.SetFlags(0)
	}

	var err error
	db, err = OpenDB(DBFile)
	if err != nil {
		log.Fatalln("Failed to open DB:", err)
	}
	defer db.Close()

	onlineAgents = make(map[string]bool)
	taskResults = make(map[string]map[int]chan map[string]interface{})
	tasks = make(map[string]chan map[string]interface{})

	http.HandleFunc("/tasks", tasksHandler)
	http.HandleFunc("/task_result", tasksResultHandler)
	http.HandleFunc("/login", loginHandler)
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/agents", agentsHandler)
	http.HandleFunc("/agents_selfreg", agentsSelfregHandler)

	go func() {
		log.Println("Listening on :" + port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalln(err)
		}
	}()

	if os.Getenv("USING_SYSTEMD") == "1" {
		cmd := exec.Command("systemd-notify", "--ready", `--status=Listening on 0.0.0.0:`+port)
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
		if !agentsSelfregEnabled {
			writeError(w, http.StatusMethodNotAllowed, "Agents self-registration is disabled")
			return
		}

		name := r.URL.Query().Get("name")
		hwid := r.URL.Query().Get("hwid")
		if name == "" || hwid == "" {
			writeError(w, http.StatusBadRequest, "Pass 'name' and 'hwid' in query string")
			return
		}

		if err := db.AddAgent(name, hwid); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
	} else if r.Method == http.MethodGet {
		if !checkAdminAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		agents, err := db.ListAgents()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}

		onlineAgentsL := make(map[string]bool)
		for _, agent := range agents {
			onlineAgentsL[agent] = onlineAgents[agent]
		}

		writeJson(w, map[string]interface{}{"error": false, "agents": agents, "online": onlineAgentsL})
	} else if r.Method == http.MethodPatch {
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

		onlineAgentsLock.Lock()
		onlineAgents[newId] = onlineAgents[oldId]
		delete(onlineAgents, oldId)
		onlineAgentsLock.Unlock()
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/agents only supports POST, GET and PATCH")
	}

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	if !db.CheckAuth(r.URL.Query().Get("token")) {
		log.Println("Invalid login info submitted for", user, "from", r.RemoteAddr)
		writeError(w, http.StatusForbidden, "Invalid credentials")
		return
	}

	token, err := db.InitSession(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Println("Initialized session for", user+"; token="+token[:6]+"...")
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
