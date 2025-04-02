package db

import (
	"database/sql"
	"sync"
	"time"
)

// Class model to keep open SQL connections in the pool
// with concurrent read/write access
type DbList struct {
	items sync.Map
	mu    sync.Mutex
}

// Keeps SQL Db connection information
type DbConn struct {
	Hash      [32]byte  // Hash, as sql.DB does not store credentials
	DB        *sql.DB   // SQL server connection pool (provided by the driver)
	Timestamp time.Time // Last use
	Stmt      []DbStmt  // Prepared SQL statements
}

// Keeps SQL prepared statement information
type DbStmt struct {
	Id        string
	Stmt      *sql.Stmt
	Timestamp time.Time // Last use
}

// Keeps SQL connection string information
type DbConnInfo struct {
	DbType   string `json:"db_type"`
	Host     string `json:"host"`
	Port     uint16 `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	DbName   string `json:"db_name"`
	SSL      bool   `json:"ssl"`
}
