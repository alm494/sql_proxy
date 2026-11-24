package db

import (
	"database/sql"
	"sync"
	"time"
)

// The pool itself: open DB connections plus their prepared statements, guarded by an RWMutex.
// Hot paths take the read lock only — don't let that change
type DbList struct {
	items  map[string]*DbConn
	byHash map[[32]byte]string
	mu     sync.RWMutex
}

// One pooled connection together with its prepared statements.
// Timestamp is touched atomically, everything else goes through our write lock
type DbConn struct {
	Hash      [32]byte
	DB        *sql.DB
	Timestamp int64
	Stmt      []DbStmt
}

// A prepared statement plus its last-use time — the maintenance janitor watches this closely
type DbStmt struct {
	Id        string
	Stmt      *sql.Stmt
	Timestamp time.Time
}

// Connection parameters the API client sends us — we turn them into a DSN and never expose them again
type DbConnInfo struct {
	DbType   string `json:"db_type"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DbName   string `json:"db_name"`
	SSL      bool   `json:"ssl"`
}
