package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"sql-proxy/src/app"

	"time"

	"slices"

	"github.com/google/uuid"
)

// Init map
func (o *DbList) Init() {

	items := make(map[string]DbConn)
	o.items = items

}

// Gets SQL server connection by GUID
func (o *DbList) GetById(id string, updateTimestamp bool) (*sql.DB, bool) {

	o.mu.RLock()

	if dbConn, ok := o.items[id]; ok {
		if updateTimestamp {
			o.mu.RUnlock()
			o.mu.Lock()
			dbConn.Timestamp = time.Now()
			o.items[id] = dbConn
			o.mu.Unlock()
		}
		return dbConn.DB, true
	}

	o.mu.RUnlock()

	app.Logger.Errorf("SQL connection with guid='%s' not found", id)
	return nil, false

}

// Gets the new SQL server connection with parameters given.
// First lookups in pool, if fails opens new one and returns GUID value
func (o *DbList) GetByParams(connInfo *DbConnInfo) (string, bool) {
	hash, err := connInfo.GetHash()
	if err != nil {
		errMsg := "Hash calculation failed"
		app.Logger.Error(errMsg)
		return errMsg, false
	}

	guid := ""

	o.mu.RLock()

	for key, dbConn := range o.items {

		// Search existing connection by hash to reuse
		if bytes.Equal(dbConn.Hash[:], hash[:]) {
			guid = key
			app.Logger.Infof("DB connection with id %s found in the pool", guid)

			// Perform checks
			if err = dbConn.DB.Ping(); err == nil {
				o.mu.RUnlock()
				// Everything is ok, return guid
				return guid, true
			} else {
				// Bad connection, need to clean
				o.mu.RUnlock()
				o.mu.Lock()
				delete(o.items, guid)
				o.mu.Unlock()
				o.mu.RLock()
				app.Logger.Infof("DB connection with id %s is dead and removed from the pool", guid)
			}
		}
	}

	o.mu.RUnlock()

	// At this step nothing found, create the new
	return o.getNewConnection(connInfo, hash)
}

// Creates the new SQL connection regarding concurrency
func (o *DbList) getNewConnection(connInfo *DbConnInfo, hash [32]byte) (string, bool) {

	o.mu.Lock()
	defer o.mu.Unlock()

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
		errMsg := fmt.Sprintf("No suitable driver implemented for server type '%s'", connInfo.DbType)
		app.Logger.Error(errMsg)
		return errMsg, false
	}

	// Open new SQL server connection
	var err error
	var newDb *sql.DB

	newDb, err = sql.Open(connInfo.DbType, dsn)

	// Check for failure
	if err != nil {
		errMsg := "Error establishing SQL server connection"
		app.Logger.Error(errMsg)
		return errMsg, false
	}

	// Check if alive
	if err = newDb.Ping(); err != nil {
		errMsg := "Just created SQL connection is dead"
		app.Logger.Error(errMsg)
		return errMsg, false
	}

	// Insert into pool
	newId := uuid.New().String()
	newItem := DbConn{
		Hash:      hash,
		DB:        newDb,
		Timestamp: time.Now(),
	}

	o.items[newId] = newItem

	app.Logger.Infof("New SQL connection with id %s was added to the pool: "+
		"Host=%s, Port=%d, dbName=%s, user=%s, dbType=%s, Id=%s",
		newId,
		connInfo.Host,
		connInfo.Port,
		connInfo.DbName,
		connInfo.User,
		connInfo.DbType,
		newId,
	)

	return newId, true
}

// Deletes SQL server connection
func (o *DbList) Delete(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.items, id)
	app.Logger.Infof("DB connection with id %s was deleted by query", id)

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
	for i := range dbConn.Stmt {
		if dbConn.Stmt[i].Id == stmtId {
			dbConn.Stmt[i].Stmt.Close()
			dbConn.Stmt = slices.Delete(dbConn.Stmt, i, i+1)
			break
		}
	}

	return true

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

			var lostStmts []string
			countConn++

			if err := dbConn.DB.Ping(); err != nil {
				// dead connection
				deadItems = append(deadItems, key)
				countDeadConn++
			} else if time.Since(dbConn.Timestamp).Abs().Minutes() > 20 {
				// connection not used for last 20 minutes
				deadItems = append(deadItems, key)
				countDeadConn++
			}

			// check prepared statements
			for _, stmt := range dbConn.Stmt {
				// prepared statements not used last 20 minutes
				if time.Since(stmt.Timestamp).Abs().Minutes() > 20 {
					lostStmts = append(lostStmts, stmt.Id)
					countStmt++
				}
			}

			// delete lost prepared statements
			for _, lost := range lostStmts {
				for i := range dbConn.Stmt {
					if dbConn.Stmt[i].Id == lost {
						dbConn.Stmt[i].Stmt.Close()
						dbConn.Stmt = slices.Delete(dbConn.Stmt, i, i+1)
						break
					}
				}
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
