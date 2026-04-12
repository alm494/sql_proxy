package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"sql-proxy/src/app"

	"time"

	"github.com/google/uuid"
)

var ErrConnectionLimit = fmt.Errorf("connection pool limit reached")

// Init map
func (o *DbList) Init() {

	items := make(map[string]DbConn)
	o.items = items

}

// Returns current pool size
func (o *DbList) Size() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.items)
}

// Gets SQL server connection by GUID
func (o *DbList) GetById(id string, updateTimestamp bool) (*sql.DB, bool) {

	if updateTimestamp {
		o.mu.Lock()
		defer o.mu.Unlock()

		if dbConn, ok := o.items[id]; ok {
			dbConn.Timestamp = time.Now()
			o.items[id] = dbConn
			return dbConn.DB, true
		}
	} else {
		o.mu.RLock()
		defer o.mu.RUnlock()

		if dbConn, ok := o.items[id]; ok {
			return dbConn.DB, true
		}
	}

	app.Logger.Errorf("SQL connection with guid='%s' not found", id)
	return nil, false

}

// Gets the new SQL server connection with parameters given.
// First lookups in pool, if fails opens new one and returns GUID value
func (o *DbList) GetByParams(connInfo *DbConnInfo) (string, error) {
	hash, err := connInfo.GetHash()
	if err != nil {
		app.Logger.Error("Hash calculation failed")
		return "", fmt.Errorf("hash calculation failed")
	}

	guid := ""

	o.mu.RLock()

	for key, dbConn := range o.items {

		// Search existing connection by hash to reuse
		if bytes.Equal(dbConn.Hash[:], hash[:]) {
			guid = key
			app.Logger.Debugf("DB connection with id %s found in the pool", guid)

			// Perform checks
			if err = dbConn.DB.Ping(); err == nil {
				o.mu.RUnlock()
				// Everything is ok, return guid
				return guid, nil
			} else {
				// Bad connection, need to clean
				o.mu.RUnlock()
				o.mu.Lock()
				delete(o.items, guid)
				o.mu.Unlock()
				o.mu.RLock()
				app.Logger.Debugf("DB connection with id %s is dead and removed from the pool", guid)
			}
		}
	}

	o.mu.RUnlock()

	// At this step nothing found, create the new
	return o.getNewConnection(connInfo, hash)
}

// Creates the new SQL connection regarding concurrency
func (o *DbList) getNewConnection(connInfo *DbConnInfo, hash [32]byte) (string, error) {

	o.mu.Lock()
	defer o.mu.Unlock()

	// Check connection pool limit
	if len(o.items) >= MaxConnections {
		app.Logger.Errorf("Connection pool limit reached: %d", MaxConnections)
		return "", ErrConnectionLimit
	}

	// Prepare DSN string
	var dsn string

	encodedPassword := url.QueryEscape(connInfo.Password)

	switch connInfo.DbType {
	case "postgres":
		sslMode := "disable"
		if connInfo.SSL {
			sslMode = "enable"
		}
		dsn = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			connInfo.Host, connInfo.Port, connInfo.User, encodedPassword, connInfo.DbName, sslMode)
	case "sqlserver":
		dsn = fmt.Sprintf("server=%s;user id=%s;password=%s;database=%s;port=%d",
			connInfo.Host, connInfo.User, encodedPassword, connInfo.DbName, connInfo.Port)
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
			connInfo.User, encodedPassword, connInfo.Host, connInfo.Port, connInfo.DbName)
	default:
		errMsg := fmt.Sprintf("unsupported database type '%s'", connInfo.DbType)
		app.Logger.Error(errMsg)
		return "", fmt.Errorf("%s", errMsg)
	}

	// Open new SQL server connection
	var err error
	var newDb *sql.DB

	newDb, err = sql.Open(connInfo.DbType, dsn)

	// Check for failure
	if err != nil {
		app.Logger.Errorf("Error establishing SQL server connection: %v", err)
		return "", fmt.Errorf("error establishing SQL server connection")
	}

	// Check if alive
	if err = newDb.Ping(); err != nil {
		app.Logger.Errorf("Just created SQL connection is dead: %v", err)
		return "", fmt.Errorf("error establishing SQL server connection")
	}

	// Insert into pool
	newId := uuid.New().String()
	newItem := DbConn{
		Hash:      hash,
		DB:        newDb,
		Timestamp: time.Now(),
	}

	o.items[newId] = newItem

	app.Logger.Debugf("New SQL connection with id %s was added to the pool: "+
		"Host=%s, Port=%d, dbName=%s, user=%s, dbType=%s, Id=%s",
		newId,
		connInfo.Host,
		connInfo.Port,
		connInfo.DbName,
		connInfo.User,
		connInfo.DbType,
		newId,
	)

	return newId, nil
}

// Deletes SQL server connection
func (o *DbList) Delete(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.items, id)
	app.Logger.Debugf("DB connection with id %s was deleted by query", id)

}

// *** SQL prepared statements ***

// Saves SQL prepared statement
func (o *DbList) PutPreparedStatement(id string, stmt *sql.Stmt) (string, bool) {

	o.mu.Lock()
	defer o.mu.Unlock()

	dbConn, ok := o.items[id]
	if !ok {
		return "", false
	}

	newId := uuid.New().String()
	dbStmt := DbStmt{
		Id:        newId,
		Stmt:      stmt,
		Timestamp: time.Now(),
	}

	dbConn.Timestamp = time.Now()
	dbConn.Stmt = append(dbConn.Stmt, dbStmt)
	o.items[id] = dbConn

	return newId, true
}

// Gets SQL prepared statement
func (o *DbList) GetPreparedStatement(connId, stmtId string) (*sql.Stmt, bool) {

	o.mu.RLock()
	defer o.mu.RUnlock()

	dbConn, ok := o.items[connId]
	if !ok {
		return nil, false
	}

	for i := range dbConn.Stmt {
		if dbConn.Stmt[i].Id == stmtId {
			return dbConn.Stmt[i].Stmt, true
		}
	}
	return nil, false
}

// Closes and deletes SQL prepared statement
func (o *DbList) ClosePreparedStatement(connId, stmtId string) bool {

	o.mu.Lock()
	defer o.mu.Unlock()

	dbConn, ok := o.items[connId]
	if !ok {
		return false
	}

	for i, stmt := range dbConn.Stmt {
		if stmt.Id == stmtId {
			stmt.Stmt.Close()
			dbConn.Stmt = append(dbConn.Stmt[:i], dbConn.Stmt[i+1:]...)
			o.items[connId] = dbConn
			return true
		}
	}

	return false

}

// *** Maintenance ***

func (o *DbList) RunMaintenance() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C

		// detect dead connections
		var deadItems []string
		var countConn, countDeadConn, countStmt int

		o.mu.Lock()

		for key, dbConn := range o.items {
			countConn++

			isDead := false
			if err := dbConn.DB.Ping(); err != nil {
				// dead connection
				deadItems = append(deadItems, key)
				countDeadConn++
				isDead = true
			} else if time.Since(dbConn.Timestamp).Abs().Minutes() > 20 {
				// connection not used for last 20 minutes
				deadItems = append(deadItems, key)
				countDeadConn++
				isDead = true
			}

			// Close and remove expired prepared statements
			activeStmts := make([]DbStmt, 0, len(dbConn.Stmt))
			for _, stmt := range dbConn.Stmt {
				if time.Since(stmt.Timestamp).Abs().Minutes() > 20 {
					stmt.Stmt.Close()
					countStmt++
				} else {
					activeStmts = append(activeStmts, stmt)
				}
			}
			dbConn.Stmt = activeStmts

			// Update connection in pool (without timestamp change)
			if !isDead {
				o.items[key] = dbConn
			}
		}

		// remove dead connections
		for _, item := range deadItems {
			dbConn := o.items[item]
			dbConn.DB.Close()
			delete(o.items, item)
		}

		o.mu.Unlock()

		app.Logger.Infof("Regular task: SQL connection pool size = %d", countConn)
		app.Logger.Infof("Regular task: %d dead connections removed", countDeadConn)
		app.Logger.Infof("Regular task: %d lost prepared statements removed", countStmt)
	}
}
