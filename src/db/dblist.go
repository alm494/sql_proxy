package db

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"sql-proxy/src/app"

	"github.com/google/uuid"
)

// Returned when the pool is full — callers should map it to HTTP 429, not a plain error
var ErrConnectionLimit = fmt.Errorf("connection pool limit reached")

const (
	createPingTimeout = 10 * time.Second // How long we wait for a ping when opening a new DB. Enough, really
	maintenanceIdle   = 20 * time.Minute // Unused longer than this and the connection gets tossed from the pool
	pingCheckTimeout  = 3 * time.Second  // Per-connection liveness check during the maintenance sweep
)

// Initialize our internal maps — call once before anything else, please
func (o *DbList) Init() {

	o.items = make(map[string]*DbConn)
	o.byHash = make(map[[32]byte]string)

}

// Bump the "last used" stamp without grabbing the lock — hot paths must stay fast
func (c *DbConn) touch() { atomic.StoreInt64(&c.Timestamp, time.Now().UnixNano()) }

// Read back what touch() wrote. Atomic read — no lock needed here either
func (c *DbConn) lastUsed() time.Time { return time.Unix(0, atomic.LoadInt64(&c.Timestamp)) }

// How many connections are currently pooled — mostly for the metrics folks
func (o *DbList) Size() int {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return len(o.items)
}

// Look up a connection by its id, and optionally refresh the "last used" stamp on the way out
func (o *DbList) GetById(id string, updateTimestamp bool) (*sql.DB, bool) {

	o.mu.RLock()
	dbConn, ok := o.items[id]
	o.mu.RUnlock()

	if !ok {
		app.Logger.Errorf("SQL connection with guid='%s' not found", id)
		return nil, false
	}

	if updateTimestamp {
		dbConn.touch()
	}

	return dbConn.DB, true

}

// Find a connection in the pool or open a new one. 
// Same params → same backend, never duplicates — that's the whole point
func (o *DbList) GetByParams(connInfo *DbConnInfo) (string, error) {
	hash, err := connInfo.GetHash()
	if err != nil {
		app.Logger.Error("Hash calculation failed")
		return "", fmt.Errorf("hash calculation failed")
	}

	o.mu.RLock()
	guid, ok := o.byHash[hash]
	if ok {

		o.items[guid].touch()
	}
	o.mu.RUnlock()

	if ok {
		app.Logger.Debugf("DB connection with id %s found in the pool", guid)
		return guid, nil
	}

	return o.getNewConnection(connInfo, hash)
}

// The slow path: build a DSN, connect for real, ping until we're sure it's alive.
// Pray the database is up — this is where everything goes when nothing else works
func (o *DbList) getNewConnection(connInfo *DbConnInfo, hash [32]byte) (string, error) {

	o.mu.Lock()
	defer o.mu.Unlock()

	if guid, ok := o.byHash[hash]; ok {
		app.Logger.Debugf("DB connection with id %s was created concurrently", guid)
		return guid, nil
	}

	if len(o.items) >= MaxConnections {
		app.Logger.Errorf("Connection pool limit reached: %d", MaxConnections)
		return "", ErrConnectionLimit
	}

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

	var err error
	var newDb *sql.DB

	newDb, err = sql.Open(connInfo.DbType, dsn)

	if err != nil {
		app.Logger.Errorf("Error establishing SQL server connection: %v", err)
		return "", fmt.Errorf("error establishing SQL server connection")
	}

	newDb.SetMaxOpenConns(MaxPoolConns)
	newDb.SetConnMaxLifetime(time.Hour)
	newDb.SetConnMaxIdleTime(15 * time.Minute)

	pingCtx, cancel := context.WithTimeout(context.Background(), createPingTimeout)
	err = newDb.PingContext(pingCtx)
	cancel()

	if err != nil {
		newDb.Close()
		app.Logger.Errorf("Just created SQL connection is dead: %v", err)
		return "", fmt.Errorf("error establishing SQL server connection")
	}

	newId := uuid.New().String()
	o.items[newId] = &DbConn{
		Hash:      hash,
		DB:        newDb,
		Timestamp: time.Now().UnixNano(),
	}
	o.byHash[hash] = newId

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

// Kill the connection and everything inside it (prepared statements included). No leftovers allowed
func (o *DbList) Delete(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()

	conn, ok := o.items[id]
	if !ok {
		return
	}

	for _, stmt := range conn.Stmt {
		stmt.Stmt.Close()
	}
	conn.DB.Close()

	delete(o.items, id)
	if o.byHash[conn.Hash] == id {
		delete(o.byHash, conn.Hash)
	}

	app.Logger.Debugf("DB connection with id %s was deleted by query", id)

}

// Save a prepared statement under the connection and hand back its new id — hold onto it, you'll need it
func (o *DbList) PutPreparedStatement(id string, stmt *sql.Stmt) (string, bool) {

	o.mu.Lock()
	defer o.mu.Unlock()

	conn, ok := o.items[id]
	if !ok {
		return "", false
	}

	newId := uuid.New().String()
	dbStmt := DbStmt{
		Id:        newId,
		Stmt:      stmt,
		Timestamp: time.Now(),
	}

	conn.Stmt = append(conn.Stmt, dbStmt)
	conn.touch()

	return newId, true
}

// Fetch a saved prepared statement by its id — or not. That's how it is sometimes
func (o *DbList) GetPreparedStatement(connId, stmtId string) (*sql.Stmt, bool) {

	o.mu.RLock()
	defer o.mu.RUnlock()

	conn, ok := o.items[connId]
	if !ok {
		return nil, false
	}

	for i := range conn.Stmt {
		if conn.Stmt[i].Id == stmtId {
			return conn.Stmt[i].Stmt, true
		}
	}
	return nil, false
}

// Close the saved prepared statement and drop it from our list forever
func (o *DbList) ClosePreparedStatement(connId, stmtId string) bool {

	o.mu.Lock()
	defer o.mu.Unlock()

	conn, ok := o.items[connId]
	if !ok {
		return false
	}

	for i, stmt := range conn.Stmt {
		if stmt.Id == stmtId {
			stmt.Stmt.Close()
			conn.Stmt = append(conn.Stmt[:i], conn.Stmt[i+1:]...)
			return true
		}
	}

	return false

}

// The background janitor. Runs every 2 minutes, tosses idle or dead connections and their
// abandoned statements — the only reason we don't leak everything. Don't kill this goroutine
func (o *DbList) RunMaintenance() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		<-ticker.C

		o.mu.RLock()
		snapshot := make(map[string]*DbConn, len(o.items))
		for id, conn := range o.items {
			snapshot[id] = conn
		}
		countConn := len(snapshot)
		o.mu.RUnlock()

		deadCh := make(chan string, len(snapshot))
		var wg sync.WaitGroup

		for id, conn := range snapshot {
			if time.Since(conn.lastUsed()) > maintenanceIdle {

				deadCh <- id
				continue
			}

			wg.Add(1)
			go func(id string, db *sql.DB) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), pingCheckTimeout)
				defer cancel()
				if err := db.PingContext(ctx); err != nil {
					deadCh <- id
				}
			}(id, conn.DB)
		}

		wg.Wait()
		close(deadCh)

		deadSet := make(map[string]bool, len(deadCh))
		for id := range deadCh {
			deadSet[id] = true
		}

		var countDeadConn, countStmt int

		o.mu.Lock()

		for id, conn := range snapshot {
			if _, ok := o.items[id]; !ok {
				continue
			}
			if deadSet[id] {
				continue
			}

			activeStmts := make([]DbStmt, 0, len(conn.Stmt))
			for _, stmt := range conn.Stmt {
				if time.Since(stmt.Timestamp) > maintenanceIdle {
					stmt.Stmt.Close()
					countStmt++
				} else {
					activeStmts = append(activeStmts, stmt)
				}
			}
			conn.Stmt = activeStmts
		}

		for id := range deadSet {
			conn, ok := o.items[id]
			if !ok {
				continue
			}

			for _, stmt := range conn.Stmt {
				stmt.Stmt.Close()
			}
			conn.DB.Close()

			delete(o.items, id)
			if o.byHash[conn.Hash] == id {
				delete(o.byHash, conn.Hash)
			}
			countDeadConn++
		}

		o.mu.Unlock()

		app.Logger.Infof("Regular task: SQL connection pool size = %d", countConn)
		app.Logger.Infof("Regular task: %d dead connections removed", countDeadConn)
		app.Logger.Infof("Regular task: %d lost prepared statements removed", countStmt)
	}
}
