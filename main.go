package main

import (
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
)

var db *DB
var agentsSelfregEnabled = false
var onlineAgents map[string]bool

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, os.Args[0], "server PORT DBFILE")
		fmt.Fprintln(os.Stderr, "\tLaunch server on PORT using DBFILE.")
		fmt.Fprintln(os.Stderr, os.Args[0], "addaccount DBFILE NAME TYPE")
		fmt.Fprintln(os.Stderr, "\tAdd client NAME as TYPE to DBFILE. Password will be readen from stdin. Type can be either 'agent' or 'admin'.")
		fmt.Fprintln(os.Stderr, os.Args[0], "remove DBFILE NAME")
		fmt.Fprintln(os.Stderr, "\tRemove client NAME from DBFILE.")
		return
	}

	subCmd := os.Args[1]
	switch subCmd {
	case "server":
		serverSubcommand()
	case "addaccount":
		addAccountSubcommand()
	case "remove":
		removeAccountSubcommand()
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

	var err error
	db, err = OpenDB(DBFile)
	if err != nil {
		log.Fatalln("Failed to open DB:", err)
	}
	defer db.Close()

	onlineAgents = make(map[string]bool)
	taskResults = make(map[string]map[int]chan json.RawMessage)
	tasks = make(map[string]chan map[string]interface{})

	http.HandleFunc("/tasks", tasksHandler)
	http.HandleFunc("/tasks_result", tasksResultHandler)
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

		user := r.URL.Query().Get("user")
		pass := r.URL.Query().Get("pass")
		if user == "" || pass == "" {
			writeError(w, http.StatusBadRequest, "Pass 'user' and 'pass' in query string")
			return
		}

		if err := db.AddAccount(user, pass, AcctAgent); err != nil {
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
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/agents only supports POST and GET")
	}

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	if !db.CheckAuth(user, r.URL.Query().Get("pass"), AcctAdmin) {
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
	writeJson(w, map[string]interface{}{"error": false, "msg": "Logged in", "token": token})
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
	authParts := strings.Split(h.Get("Authorization"), ":")
	if len(authParts) != 2 {
		return false
	}

	return db.CheckAuth(authParts[0], authParts[1], AcctAgent)
}

func checkAdminAuth(h http.Header) bool {
	return db.CheckSession(h.Get("Authorization"))
}
