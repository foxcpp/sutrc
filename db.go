package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"golang.org/x/crypto/scrypt"
	"log"
	"sync"
)

type AccountType int

const (
	AcctAdmin AccountType = 1
	AcctAgent AccountType = 2
)

type DB struct {
	d *sql.DB

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

	// Used to reduce scrypt invocations count and thus
	// resist DoS'ing and improve agents performance.
	credsCache sync.Map
}

func OpenDB(path string) (*DB, error) {
	db := new(DB)
	var err error
	db.d, err = sql.Open("sqlite3", "file:"+path+"?cache=shared&_journal=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

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
	cachedPass, ok := db.credsCache.Load(user)
	if ok {
		return cachedPass == pass
	}

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
	success := res == 1
	if success {
		db.credsCache.Store(user, pass)
	}
	return success
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
	db.addAccount, err = db.d.Prepare(`
		INSERT INTO authInfo VALUES (?, ?, ?, ?)
		ON CONFLICT(user) DO UPDATE SET
			salt = excluded.salt,
			hashedPass = excluded.hashedPass
		WHERE type = excluded.type`)
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

	return nil
}
