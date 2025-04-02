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
	"github.com/sirupsen/logrus"
)

// *** SQL connections ***

// Gets SQL server connection by GUID
func (o *DbList) GetById(id string, updateTimestamp bool) (*sql.DB, bool) {
	if val, ok := o.items.Load(id); ok {
		res := val.(*DbConn)
		if updateTimestamp {
			res.Timestamp = time.Now()
			o.items.Store(id, res)
		}
		return res.DB, true
	}
	app.Log.Errorf("SQL connection with guid='%s' not found", id)
	return nil, false
}

// Gets the new SQL server connection with parameters given.
// First lookups in pool, if fails opens new one and returns GUID value
func (o *DbList) GetByParams(connInfo *DbConnInfo) (string, bool) {
	hash, err := connInfo.GetHash()
	if err != nil {
		errMsg := "Hash calculation failed"
		app.Log.WithError(err).Error(errMsg)
		return errMsg, false
	}

	// Step 1. Search existing connection by hash to reuse
	guid := ""
	o.items.Range(
		func(key, value any) bool {
			if bytes.Equal(value.(*DbConn).Hash[:], hash[:]) {
				guid = key.(string)
				app.Log.Debugf("DB connection with id %s found in the pool", guid)
				return false // stop iteraton
			}
			return true // continue iteration
		})

	// Step 2. Perform checks and return guid if passed
	if len(guid) > 0 {
		if conn, ok := o.items.Load(guid); ok {
			if err = conn.(*DbConn).DB.Ping(); err == nil {
				// Everything is ok, return guid
				return guid, true
			} else {
				// Remove dead connection from the pool
				o.items.Delete(guid)
				app.Log.Debugf("DB connection with id %s is dead and removed from the pool", guid)
			}

		}
	}

	// Step 3. Nothing found, create the new
	return o.getNewConnection(connInfo, hash)
}

// Creates the new SQL connection regarding concurrency
func (o *DbList) getNewConnection(connInfo *DbConnInfo, hash [32]byte) (string, bool) {
	// 1. Prepare DSN string
	var dsn string

	encodedPassword := url.QueryEscape(connInfo.Password)

	switch connInfo.DbType {
	case "postgres":
		sslMode := "enable"
		if !connInfo.SSL {
			sslMode = "disable"
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
		app.Log.Error(errMsg)
		return errMsg, false
	}

	// 2. Open new SQL server connection
	var err error
	var newDb *sql.DB

	newDb, err = sql.Open(connInfo.DbType, dsn)

	// 3. Check for failure
	if err != nil {
		errMsg := "Error establishing SQL server connection"
		app.Log.WithError(err).Error(errMsg)
		return errMsg, false
	}

	// 4. Check if alive
	if err = newDb.Ping(); err != nil {
		errMsg := "Just created SQL connection is dead"
		app.Log.WithError(err).Error(errMsg)
		return errMsg, false
	}

	// 5. Insert into pool
	newId := uuid.New().String()
	newItem := DbConn{
		Hash:      hash,
		DB:        newDb,
		Timestamp: time.Now(),
	}

	o.items.Store(newId, &newItem)

	app.Log.WithFields(logrus.Fields{
		"Host":   connInfo.Host,
		"Port":   connInfo.Port,
		"dbName": connInfo.DbName,
		"user":   connInfo.User,
		"dbType": connInfo.DbType,
		"Id":     newId,
	}).Infof("New SQL connection with id %s was added to the pool", newId)

	return newId, true
}

// Deletes SQL server connection
func (o *DbList) Delete(id string) {
	o.items.Delete(id)
	app.Log.Debugf("DB connection with id %s was deleted by query", id)
}

// *** SQL prepared statements ***

// Saves SQL prepared statement
func (o *DbList) PutPreparedStatement(id string, stmt *sql.Stmt) (string, bool) {
	o.mu.Lock()
	defer o.mu.Unlock()

	val, ok := o.items.Load(id)
	if !ok {
		return "", false
	}

	newId := uuid.New().String()
	dbStmt := DbStmt{
		Id:        newId,
		Stmt:      stmt,
		Timestamp: time.Now(),
	}
	res := val.(*DbConn)
	res.Timestamp = time.Now()
	res.Stmt = append(res.Stmt, dbStmt)
	//o.items.Store(id, res)
	return newId, true
}

// Gets SQL prepared statement
func (o *DbList) GetPreparedStatement(connId, stmtId string) (*sql.Stmt, bool) {
	val, ok := o.items.Load(connId)
	if !ok {
		return nil, false
	}
	res := val.(*DbConn)
	for i := range res.Stmt {
		if res.Stmt[i].Id == stmtId {
			return res.Stmt[i].Stmt, true
		}
	}
	return nil, false
}

// Closes and deletes SQL prepared statement
func (o *DbList) ClosePreparedStatement(connId, stmtId string) bool {
	val, ok := o.items.Load(connId)
	if !ok {
		return false
	}
	res := val.(*DbConn)
	for i := range res.Stmt {
		if res.Stmt[i].Id == stmtId {
			res.Stmt[i].Stmt.Close()
			res.Stmt = slices.Delete(res.Stmt, i, i+1)
			break
		}
	}
	o.items.Store(connId, res)
	return true
}

// *** Maintenance ***

func (o *DbList) RunMaintenance() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		// detect dead connections
		var deadItems []string
		var countConn, countDeadConn, countStmt int

		o.mu.Lock()

		o.items.Range(
			func(key, value any) bool {
				var lostStmts []string
				countConn++
				dbConn := value.(*DbConn)

				err := dbConn.DB.Ping()
				if err != nil {
					// dead connection
					deadItems = append(deadItems, key.(string))
					countDeadConn++
				} else if time.Since(dbConn.Timestamp).Abs().Minutes() > 20 {
					// connection not used for last 20 minutes
					deadItems = append(deadItems, key.(string))
					countDeadConn++
				}

				for _, stmt := range dbConn.Stmt {
					// prepared statements not used last 20 minutes
					if time.Since(stmt.Timestamp).Abs().Minutes() > 20 {
						lostStmts = append(lostStmts, stmt.Id)
						countStmt++
					}
				}

				// delete lost prepared statements
				for _, lost := range lostStmts {
					for i := 0; i < len(dbConn.Stmt); i++ {
						if dbConn.Stmt[i].Id == lost {
							dbConn.Stmt[i].Stmt.Close()
							dbConn.Stmt = slices.Delete(dbConn.Stmt, i, i+1)
							break
						}
					}
				}

				return true // continue iteration
			})

		// remove dead connections
		for _, item := range deadItems {
			conn, _ := o.GetById(item, false)
			conn.Close()
			o.Delete(item)
		}

		o.mu.Unlock()

		app.Log.Debugf("Regular task: SQL connection pool size = %d", countConn)
		app.Log.Debugf("Regular task: %d dead connections removed", countDeadConn)
		app.Log.Debugf("Regular task: %d lost prepared statements removed", countStmt)
	}
}
