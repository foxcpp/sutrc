package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"golang.org/x/crypto/scrypt"
	"log"
)

const MaxEventLogSize = 512

type AccountType int

const (
	AcctAdmin AccountType = 1
	AcctAgent AccountType = 2
)

type DB struct {
	d             *sql.DB
	popTaskWakeUp chan bool

	listAgents *sql.Stmt

	// Account management
	getUsrSalt        *sql.Stmt
	checkUsrExistance *sql.Stmt
	addAccount        *sql.Stmt
	remAccount        *sql.Stmt

	// Session management
	initSession  *sql.Stmt
	killSession  *sql.Stmt
	checkSession *sql.Stmt

	// Task queue
	pushTask     *sql.Stmt
	getFirstTask *sql.Stmt
	delTask      *sql.Stmt

	// Events log
	pushEvent          *sql.Stmt
	countAndFirstEvent *sql.Stmt
	delEvent           *sql.Stmt
	listEvents         *sql.Stmt
}

func OpenDB(path string) (*DB, error) {
	db := new(DB)
	var err error
	db.d, err = sql.Open("sqlite3", "file:"+path+"?cache=shared&_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}
	db.popTaskWakeUp = make(chan bool)

	if err := db.initSchema(); err != nil {
		panic(err)
	}
	if err := db.initStmts(); err != nil {
		panic(err)
	}
	return db, nil
}

func (db *DB) Close() error {
	return db.d.Close()
}

func (db *DB) ListAgents() ([]string, error) {
	rows, err := db.listAgents.Query()
	if err != nil {
		if err == sql.ErrNoRows {
			return []string{}, nil
		}
		return nil, err
	}

	var res []string
	for rows.Next() {
		user := ""
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		res = append(res, user)
	}
	return res, nil
}

func (db *DB) CheckAuth(user, pass string, t AccountType) bool {
	row := db.getUsrSalt.QueryRow(user)
	salt := []byte{}
	if err := row.Scan(&salt); err != nil {
		log.Println("salt read", err)
		return false
	}

	hashed, err := scrypt.Key([]byte(pass), salt, 32768, 8, 1, 32)
	if err != nil {
		return false
	}

	row = db.checkUsrExistance.QueryRow(user, hashed, t)
	res := 0
	if err := row.Scan(&res); err != nil {
		return false
	}
	return res == 1
}

func (db *DB) AddAccount(user string, pass string, acctType AccountType) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	hashed, err := scrypt.Key([]byte(pass), salt, 32768, 8, 1, 32)
	if err != nil {
		return err
	}

	_, err = db.addAccount.Exec(user, acctType, salt, hashed)
	return err
}

func (db *DB) RemAccount(user string) error {
	_, err := db.remAccount.Exec(user)
	return err
}

func (db *DB) InitSession(user string) (string, error) {
	rawSID := make([]byte, 32)
	if _, err := rand.Read(rawSID); err != nil {
		return "", err
	}
	sid := hex.EncodeToString(rawSID)

	_, err := db.initSession.Exec(sid, user)
	return sid, err
}

func (db *DB) KillSession(sid string) error {
	_, err := db.killSession.Exec(sid)
	return err
}

func (db *DB) CheckSession(sid string) bool {
	row := db.checkSession.QueryRow(sid)
	res := 0
	if err := row.Scan(&res); err != nil {
		return false
	}
	return res == 1
}

func (db *DB) LogEvent(agentID string, jsonObj json.RawMessage) error {
	tx, err := db.d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.Stmt(db.countAndFirstEvent).QueryRow(agentID)
	firstId, eventsCount := sql.NullInt64{}, 0
	if err := row.Scan(&firstId, &eventsCount); err != nil {
		return err
	}

	if eventsCount+1 > MaxEventLogSize && firstId.Valid {
		if _, err := tx.Stmt(db.delEvent).Exec(firstId.Int64); err != nil {
			return err
		}
	}

	if _, err := tx.Stmt(db.pushEvent).Exec(agentID, []byte(jsonObj)); err != nil {
		return err
	}
	return tx.Commit()
}

func (db *DB) ListLoggedEvents(agentID string) ([]json.RawMessage, error) {
	var res []json.RawMessage

	rows, err := db.listEvents.Query(agentID)
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		jsonBlob := []byte{}
		if err := rows.Scan(&jsonBlob); err != nil {
			return nil, err
		}
		res = append(res, jsonBlob)
	}

	return res, nil
}

func (db *DB) PushTask(agentID string, jsonObj json.RawMessage) (int, error) {
	res, err := db.pushTask.Exec(agentID, jsonObj)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	select {
	case db.popTaskWakeUp <- true:
	default:
	}
	return int(id), err
}

func (db *DB) PopTask(cancelChan chan bool, agentID string) (int, json.RawMessage, error) {
	tx, err := db.d.Begin()
	if err != nil {
		return 0, nil, err
	}

	row := tx.Stmt(db.getFirstTask).QueryRow(agentID)
	id, blob := 0, []byte{}
	err = row.Scan(&id, &blob)
	if err == nil {
		if _, err := tx.Stmt(db.delTask).Exec(id); err != nil {
			return 0, nil, err
		}

		return id, json.RawMessage(blob), tx.Commit()
	}
	if err != sql.ErrNoRows {
		return 0, nil, err
	}
	if err := tx.Rollback(); err != nil {
		return 0, nil, err
	}

	select {
	case <-db.popTaskWakeUp:
		return db.PopTask(cancelChan, agentID)
	case <-cancelChan:
		return 0, nil, err
	}
}

func (db *DB) initSchema() error {
	db.d.Exec(`PRAGMA foreign_keys = ON`)
	db.d.Exec(`PRAGMA auto_vacuum = INCREMENTAL`)
	db.d.Exec(`PRAGMA journal_mode = WAL`)
	db.d.Exec(`PRAGMA defer_foreign_keys = ON`)
	db.d.Exec(`PRAGMA synchronous = NORMAL`)
	db.d.Exec(`PRAGMA temp_store = MEMORY`)
	db.d.Exec(`PRAGMA cache_size = 5000`)

	_, err := db.d.Exec(`CREATE TABLE IF NOT EXISTS authInfo (
		user TEXT PRIMARY KEY NOT NULL,
		type INTEGER NOT NULL, 
		salt BLOB NOT NULL,
		hashedPass BLOB NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = db.d.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		sessionId TEXT PRIMARY KEY NOT NULL,
		user TEXT NOT NULL REFERENCES authInfo(user) ON DELETE CASCADE
	)`)

	_, err = db.d.Exec(`CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		target TEXT NOT NULL REFERENCES authInfo(user) ON DELETE CASCADE,
		jsonBlob BLOB NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.d.Exec(`CREATE TABLE IF NOT EXISTS events (
		id INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
		source TEXT NOT NULL REFERENCES authInfo(user) ON DELETE CASCADE,
		jsonBlob BLOB NOT NULL
	)`)
	return err
}

func (db *DB) initStmts() error {
	var err error

	db.listAgents, err = db.d.Prepare(`SELECT user FROM authInfo WHERE type = 2`)
	if err != nil {
		return err
	}

	db.getUsrSalt, err = db.d.Prepare(`SELECT salt FROM authInfo WHERE user = ?`)
	if err != nil {
		return err
	}
	db.checkUsrExistance, err = db.d.Prepare(`SELECT COUNT() FROM authInfo WHERE user = ? AND hashedPass = ? AND type = ?`)
	if err != nil {
		return err
	}
	db.addAccount, err = db.d.Prepare(`INSERT INTO authInfo VALUES (?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	db.remAccount, err = db.d.Prepare(`DELETE FROM authInfo WHERE user = ?`)
	if err != nil {
		return err
	}

	db.initSession, err = db.d.Prepare(`INSERT INTO sessions VALUES (?, ?)`)
	if err != nil {
		return err
	}
	db.killSession, err = db.d.Prepare(`DELETE FROM sessions WHERE sessionId = ?`)
	if err != nil {
		return err
	}
	db.checkSession, err = db.d.Prepare(`SELECT COUNT() FROM sessions WHERE sessionId = ?`)
	if err != nil {
		return err
	}

	db.pushTask, err = db.d.Prepare(`INSERT INTO tasks(target, jsonBlob) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	db.getFirstTask, err = db.d.Prepare(`SELECT id, jsonBlob FROM tasks WHERE id = (SELECT MIN(id) FROM tasks WHERE target = ?)`)
	if err != nil {
		return err
	}
	db.delTask, err = db.d.Prepare(`DELETE FROM tasks WHERE id = ?`)
	if err != nil {
		return err
	}

	db.pushEvent, err = db.d.Prepare(`INSERT INTO events(source, jsonBlob) VALUES (?, ?)`)
	if err != nil {
		return err
	}
	db.countAndFirstEvent, err = db.d.Prepare(`SELECT MIN(id), COUNT() FROM events WHERE source = ?`)
	if err != nil {
		return err
	}
	db.delEvent, err = db.d.Prepare(`DELETE FROM events WHERE id = ?`)
	if err != nil {
		return err
	}
	db.listEvents, err = db.d.Prepare(`SELECT id, jsonBlob FROM events WHERE source = ?`)
	if err != nil {
		return err
	}

	return nil
}
