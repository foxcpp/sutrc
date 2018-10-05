package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var db *DB
var agentsSelfregEnabled = false

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

		http.HandleFunc("/events", eventsHandler)
		http.HandleFunc("/tasks", tasksHandler)
		http.HandleFunc("/login", loginHandler)
		http.HandleFunc("/logout", logoutHandler)
		http.HandleFunc("/agents", agentsHandler)
		http.HandleFunc("/agents_selfreg", agentsSelfregHandler)

		go func() {
			log.Println("Listening on :" + port)
			log.Fatalln(http.ListenAndServe(":"+port, nil))
		}()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM)
		<-sig
	case "addaccount":
		if len(os.Args) != 5 {
			fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "addaccount DBFILE NAME TYPE")
			os.Exit(1)
		}

		DBFile := os.Args[2]
		name := os.Args[3]
		type_ := AccountType(0)
		switch os.Args[4] {
		case "agent":
			type_ = AcctAgent
		case "admin":
			type_ = AcctAdmin
		default:
			fmt.Fprintln(os.Stderr, "Invalid type. Use either 'admin' or 'agent'.")
			os.Exit(1)
		}

		// TODO: Hide entered password from console.
		fmt.Print("Enter password for account: ")
		passIn := bufio.NewScanner(os.Stdin)
		if !passIn.Scan() {
			log.Fatalln(passIn.Err())
		}
		pass := passIn.Text()

		db, err := OpenDB(DBFile)
		if err != nil {
			log.Fatalln("Failed to open DB:", err)
		}
		defer db.Close()

		err = db.AddAccount(name, pass, type_)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to add account:", err)
			os.Exit(1)
		} else {
			fmt.Fprintln(os.Stderr, "OK.")
		}
	case "remove":
		if len(os.Args) != 4 {
			fmt.Fprintln(os.Stderr, "Usage:", os.Args[0], "remove DBFILE NAME")
			os.Exit(1)
		}

		DBFile := os.Args[2]
		name := os.Args[3]

		db, err := OpenDB(DBFile)
		if err != nil {
			log.Fatalln("Failed to open DB:", err)
		}
		defer db.Close()

		err = db.RemAccount(name)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to remove account:", err)
			os.Exit(1)
		} else {
			fmt.Fprintln(os.Stderr, "OK.")
		}
	default:
		fmt.Fprintln(os.Stderr, "Unknown subcommand.")
		os.Exit(1)
	}
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

		writeJson(w, map[string]interface{}{"error": false, "agents": agents})
	} else {
		writeError(w, http.StatusMethodNotAllowed, "/agents only supports POST and GET")
	}

}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")

	if !db.CheckAuth(user, r.URL.Query().Get("pass"), AcctAdmin) {
		log.Println("Invalid login info submitted for", user, "from", r.RemoteAddr)
		writeError(w, http.StatusForbidden, "Invalid credentials")
	}

	token, err := db.InitSession(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Println("Initialized session for", user + "; token=" + token[:6] + "...")
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

		tasksLongpool(w, r, time.Second*26)
	} else {
		writeError(w, http.StatusBadRequest, "/tasks endpoint supports only GET and POST")
	}
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		if !checkAdminAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		listEvents(w, r)
	} else if r.Method == http.MethodPost {
		if !checkAgentAuth(r.Header) {
			writeError(w, http.StatusForbidden, "Authorization failure")
			return
		}

		acceptEvent(w, r)
	} else {
		writeError(w, http.StatusBadRequest, "/events endpoint supports only GET and POST")
	}
}

func acceptEvent(w http.ResponseWriter, r *http.Request) {
	agentID := strings.Split(r.Header.Get("Authorization"), ":")[0]

	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// TODO: More advanced input validation.
	if !json.Valid(buf) {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	log.Println("Accepted event from", agentID)
	if err := db.LogEvent(agentID, buf); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
	}

}

func listEvents(w http.ResponseWriter, r *http.Request) {
	agentID := r.URL.Query().Get("agent")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, "Missing agent parameter")
		return
	}

	events, err := db.ListLoggedEvents(agentID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJson(w, map[string]interface{}{"error": false, "events": events, "max_size": MaxEventLogSize})
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

	// TODO: More advanced input validation.
	if !json.Valid(buf) {
		writeError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if err := db.PushTask(target, buf); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	log.Println("Added task for", target, "from", strings.Split(r.Header.Get("Authorization"), ":")[0])
	writeJson(w, map[string]interface{}{"error": false, "msg": "Task enqueued"})
}

func tasksLongpool(w http.ResponseWriter, r *http.Request, timeout time.Duration) {
	agentID := strings.Split(r.Header.Get("Authorization"), ":")[0]

	taskWaitCancel := make(chan bool, 1)
	doneChan := make(chan bool, 1)

	log.Println(agentID, "is watching for events")

	var blob []byte
	var id int
	var err error
	go func() {
		id, blob, err = db.PopTask(taskWaitCancel, agentID)
		doneChan <- true
	}()

	select {
	case <-time.After(timeout):
		taskWaitCancel <- true
		writeJson(w, map[string]interface{}{"error": false, "events": []interface{}{}})
	case <-doneChan:
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		log.Println("Sending event to", agentID)
		writeJson(w, map[string]interface{}{"error": false, "events": map[int]json.RawMessage{id: blob}})
	}
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
