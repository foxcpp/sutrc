package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
)

type DB struct {
	d *sql.DB

	// Account management
	checkUsrExistance *sql.Stmt
	addAccount        *sql.Stmt
	remAccount        *sql.Stmt

	// Agents management
	listAgents       *sql.Stmt
	addAgent         *sql.Stmt
	remAgent         *sql.Stmt
	renameAgent      *sql.Stmt
	checkAgentByHWID *sql.Stmt
	checkAgentByName *sql.Stmt
	getAgentName     *sql.Stmt

	// Session management
	initSession  *sql.Stmt
	killSession  *sql.Stmt
	checkSession *sql.Stmt
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

	res := []string{}
	for rows.Next() {
		user := ""
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		res = append(res, user)
	}
	return res, nil
}

func (db *DB) CheckAuth(token string) bool {
	row := db.checkUsrExistance.QueryRow(token)
	res := 0
	if err := row.Scan(&res); err != nil {
		return false
	}
	return res == 1
}

func (db *DB) AddAccount(token string) error {
	_, err := db.addAccount.Exec(token)
	return err
}

func (db *DB) RemAccount(token string) error {
	_, err := db.remAccount.Exec(token)
	return err
}

func (db *DB) RemAgent(name string) error {
	_, err := db.remAgent.Exec(name)
	return err
}

func (db *DB) AddAgent(name, hwid string) error {
	_, err := db.addAgent.Exec(name, hwid)
	return err
}

func (db *DB) RenameAgent(fromName, toName string) error {
	_, err := db.renameAgent.Exec(toName, fromName)
	return err
}

func (db *DB) AgentExists(name string) bool {
	row := db.checkAgentByName.QueryRow(name)
	res := 0
	if err := row.Scan(&res); err != nil {
		return false
	}
	return res == 1
}

func (db *DB) CheckAgentAuth(hwid string) bool {
	row := db.checkAgentByHWID.QueryRow(hwid)
	res := 0
	if err := row.Scan(&res); err != nil {
		return false
	}
	return res == 1
}

func (db *DB) GetAgentName(hwid string) (string, error) {
	row := db.getAgentName.QueryRow(hwid)
	name := ""
	return name, row.Scan(&name)
}

func (db *DB) InitSession(user string) (string, error) {
	rawSID := make([]byte, 32)
	if _, err := rand.Read(rawSID); err != nil {
		return "", err
	}
	sid := hex.EncodeToString(rawSID)

	_, err := db.initSession.Exec(sid)
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

	_, err := db.d.Exec(`CREATE TABLE IF NOT EXISTS admins (
		token TEXT PRIMARY KEY NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.d.Exec(`CREATE TABLE IF NOT EXISTS agents (
		name TEXT PRIMARY KEY NOT NULL,
		hwid TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		return err
	}

	_, err = db.d.Exec(`CREATE TABLE IF NOT EXISTS sessions (
		sessionId TEXT PRIMARY KEY NOT NULL
	)`)
	return err
}

func (db *DB) initStmts() error {
	var err error

	db.listAgents, err = db.d.Prepare(`SELECT name FROM agents`)
	if err != nil {
		return err
	}

	db.checkUsrExistance, err = db.d.Prepare(`SELECT COUNT() FROM admins WHERE token = ?`)
	if err != nil {
		return err
	}
	db.addAccount, err = db.d.Prepare(`INSERT OR IGNORE INTO admins VALUES (?)`)
	if err != nil {
		return err
	}
	db.remAccount, err = db.d.Prepare(`DELETE FROM admins WHERE token = ?`)
	if err != nil {
		return err
	}

	db.addAgent, err = db.d.Prepare(`INSERT INTO agents VALUES (?, ?)`)
	if err != nil {
		return err
	}
	db.remAgent, err = db.d.Prepare(`DELETE FROM agents WHERE name = ?`)
	if err != nil {
		return err
	}
	db.renameAgent, err = db.d.Prepare(`UPDATE agents SET name = ? WHERE name = ?`)
	if err != nil {
		return err
	}
	db.checkAgentByName, err = db.d.Prepare(`SELECT COUNT() FROM agents WHERE name = ?`)
	if err != nil {
		return err
	}
	db.checkAgentByHWID, err = db.d.Prepare(`SELECT COUNT() FROM agents WHERE hwid = ?`)
	if err != nil {
		return err
	}
	db.getAgentName, err = db.d.Prepare(`SELECT name FROM agents WHERE hwid = ?`)
	if err != nil {
		return err
	}

	db.initSession, err = db.d.Prepare(`INSERT INTO sessions VALUES (?)`)
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
