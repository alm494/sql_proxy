package db

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/url"
	"sql-proxy/src/utils"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Class model to keep open SQL connections in the pool
// with concurrent read/write access
type DbList struct {
	items sync.Map
}

// Gets SQL server connection by GUID
func (o *DbList) GetById(guid string, updateTimestamp bool) (*sql.DB, bool) {
	val, ok := o.items.Load(guid)
	if ok {
		res := val.(*DbConn)
		if updateTimestamp {
			res.Timestamp = time.Now()
			o.items.Store(guid, res)
		}
		return res.DB, true
	}
	utils.Log.Error(fmt.Sprintf("SQL connection with guid='%s' not found", guid))
	return nil, false
}

// Gets the new SQL server connection with parameters given.
// First lookups in pool, if fails opens new one and returns GUID value
func (o *DbList) GetByParams(connInfo *DbConnInfo) (string, bool) {
	hash, err := connInfo.GetHash()
	if err != nil {
		errMsg := "Hash calculation failed"
		utils.Log.WithError(err).Error(errMsg)
		return errMsg, false
	}

	// Step 1. Search existing connection by hash to reuse
	guid := ""
	o.items.Range(
		func(key, value interface{}) bool {
			if bytes.Equal(value.(*DbConn).Hash[:], hash[:]) {
				guid = key.(string)
				utils.Log.Debug(fmt.Sprintf("DB connection with id %s found in the pool", guid))
				return false
			}
			return true
		})

	// Step 2. Perform checks and return guid if passed
	if len(guid) > 0 {
		conn, ok := o.items.Load(guid)
		if ok {
			err = conn.(*DbConn).DB.Ping()
			if err == nil {
				// Everything is ok, return guid
				return guid, true
			} else {
				// Remove dead connection from the pool
				o.items.Delete(guid)
				utils.Log.Debug(fmt.Sprintf("DB connection with id %s is dead and removed from the pool", guid))
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
		utils.Log.Error(errMsg)
		return errMsg, false
	}

	// 2. Open new SQL server connection
	var err error
	var newDb *sql.DB

	newDb, err = sql.Open(connInfo.DbType, dsn)

	// 3. Check for failure
	if err != nil {
		errMsg := "Error establishing SQL server connection"
		utils.Log.WithError(err).Error(errMsg)
		return errMsg, false
	}

	// 4. Check if alive
	err = newDb.Ping()
	if err != nil {
		errMsg := "Just created SQL connection is dead"
		utils.Log.WithError(err).Error(errMsg)
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

	utils.Log.WithFields(logrus.Fields{
		"Host":   connInfo.Host,
		"Port":   connInfo.Port,
		"dbName": connInfo.DbName,
		"user":   connInfo.User,
		"dbType": connInfo.DbType,
		"Id":     newId,
	}).Info(fmt.Sprintf("New SQL connection with id %s was added to the pool", newId))

	return newId, true
}

func (o *DbList) RunMaintenance() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		utils.Log.Debug("Regular task: checking if pooled SQL connections are alive...")

		// to do: detect and remove dead connections
		//for o.items.Range()
	}
}

func (o *DbList) Delete(id string) {

	o.items.Delete(id)

}
