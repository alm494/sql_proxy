package db

import (
	"database/sql"
	"time"
)

type DbConn struct {
	// Hash, as sql.DB does not store credentials
	Hash [32]byte
	// SQL server connection pool (provided by the driver)
	DB *sql.DB
	// Last use
	Timestamp time.Time
}
