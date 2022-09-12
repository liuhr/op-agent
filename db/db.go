package db

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
	"github.com/openark/golib/log"
	"github.com/openark/golib/sqlutils"

	"op-agent/config"
)

// ExecDb will execute given query on the backend database.
func ExecDb(query string, args ...interface{}) (sql.Result, error) {
	var err error
	db, err := OpenDb()
	if err != nil {
		return nil, err
	}
	res, err := sqlutils.ExecNoPrepare(db, query, args...)
	return res, err
}

// OpenDb returns the DB instance for the  backed database
func OpenDb() (db *sql.DB, err error) {
	var fromCache bool
	if db, fromCache, err := openDbMySQLGeneric(); err != nil {
		return db, log.Errore(err)
	} else if !fromCache {
		// first time ever we talk to MySQL
		query := fmt.Sprintf("create database if not exists %s", config.Config.BackendDb)
		if _, err := db.Exec(query); err != nil {
			return db, log.Errore(err)
		}
	}

	db, fromCache, err = openDbMySQL()
	if err == nil && !fromCache {
		// do not show the password but do show what we connect to.
		safeMySQLURI := fmt.Sprintf("%s:?@tcp(%s:%d)/%s?timeout=%ds", config.Config.BackendDbUser,
			config.Config.BackendDbHosts, config.Config.BackendDbPort, config.Config.BackendDb, config.Config.MySQLConnectTimeoutSeconds)
		log.Debugf("Connected to backend db: %v", safeMySQLURI)
		if config.Config.MySQLMaxPoolConnections > 0 {
			log.Debugf("backend db pool SetMaxOpenConns: %d", config.Config.MySQLMaxPoolConnections)
			db.SetMaxOpenConns(config.Config.MySQLMaxPoolConnections)
		}
	}

	if err == nil && !fromCache {
		initDB(db)
		// A low value here will trigger reconnects which could
		// make the number of backend connections hit the tcp
		// limit. That's bad.  I could make this setting dynamic
		// but then people need to know which value to use. For now
		// allow up to 25% of MySQLrMaxPoolConnections
		// to be idle.  That should provide a good number which
		// does not keep the maximum number of connections open but
		// at the same time does not trigger disconnections and
		// reconnections too frequently.
		maxIdleConns := int(config.Config.MySQLMaxPoolConnections)
		if maxIdleConns < 10 {
			maxIdleConns = 10
		}
		log.Infof("Connecting to backend %s:%d: maxConnections: %d, maxIdleConns: %d",
			config.Config.BackendDbHosts,
			config.Config.BackendDbPort,
			config.Config.MySQLMaxPoolConnections,
			maxIdleConns)
		db.SetMaxIdleConns(maxIdleConns)
	}

	return db, nil
}

// QueryDBRowsMap
func QueryDBRowsMap(query string, on_row func(sqlutils.RowMap) error) error {
	db, err := OpenDb()
	if err != nil {
		return err
	}

	return sqlutils.QueryRowsMap(db, query, on_row)
}

// QueryOrchestrator
func QueryDB(query string, argsArray []interface{}, on_row func(sqlutils.RowMap) error) error {
	db, err := OpenDb()
	if err != nil {
		return err
	}
	return log.Criticale(sqlutils.QueryRowsMap(db, query, on_row, argsArray...))
}

func DBQueryAll(query string) (*sql.Rows, error) {
	var (
		db *sql.DB
		stmt *sql.Stmt
		err error
		rows *sql.Rows
	)

	db, err = OpenDb()
	if err != nil {
		return nil, err
	}
	stmt, err = db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()
	rows, err = stmt.Query()
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// GetDB returns a MySQL DB instance based on uri.
// bool result indicates whether the DB was returned from cache; err
func GetDB(mysql_uri string) (*sql.DB, bool, error) {
	return GetGenericDB("mysql", mysql_uri)
}

// knownDBs is a DB cache by uri
var knownDBs map[string]*sql.DB = make(map[string]*sql.DB)
var knownDBsMutex = &sync.Mutex{}

// GetDB returns a DB instance based on uri.
// bool result indicates whether the DB was returned from cache; err
func GetGenericDB(driverName, dataSourceName string) (*sql.DB, bool, error) {
	knownDBsMutex.Lock()
	defer func() {
		knownDBsMutex.Unlock()
	}()

	var exists bool
	if _, exists = knownDBs[dataSourceName]; !exists {
		if db, err := sql.Open(driverName, dataSourceName); err == nil {
			knownDBs[dataSourceName] = db
		} else {
			return db, exists, err
		}
	}
	err := knownDBs[dataSourceName].Ping()
	if err != nil {
		if knownDBs[dataSourceName] != nil {
			knownDBs[dataSourceName].Close()
		}
		delete(knownDBs, dataSourceName)
		return nil, false, err
	}
	return knownDBs[dataSourceName], exists, nil
}

func openDbMySQLGeneric() (db *sql.DB, fromCache bool, err error) {
	var uri string
	var dbConn *sql.DB
	var exists bool
	for _, host := range strings.Split(config.Config.BackendDbHosts, ",") {
		uri = fmt.Sprintf("%s:%s@tcp(%s:%d)/?timeout=%ds&readTimeout=%ds&interpolateParams=true",
			config.Config.BackendDbUser, config.Config.BackendDbPass, host, config.Config.BackendDbPort,
			config.Config.MySQLConnectTimeoutSeconds, config.Config.MySQLReadTimeoutSeconds)
		dbConn, exists, err = GetDB(uri)
		if err == nil {
			return dbConn, exists, err
		}
	}
	return dbConn, exists, err
}

func openDbMySQL() (db *sql.DB, fromCache bool, err error) {
	for _, host := range strings.Split(config.Config.BackendDbHosts, ",") {
		uri := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=%ds&readTimeout=%ds&rejectReadOnly=%t&interpolateParams=true",
			config.Config.BackendDbUser,
			config.Config.BackendDbPass,
			host,
			config.Config.BackendDbPort,
			config.Config.BackendDb,
			config.Config.MySQLConnectTimeoutSeconds,
			config.Config.MySQLReadTimeoutSeconds,
			config.Config.MySQLRejectReadOnly,
		)
		db, fromCache, err = GetDB(uri)
		if err == nil {
			return db, fromCache, err
		}
	}
	return db, fromCache, err
}

// initDB attempts to create the  backend database. It is created once in the
// application's lifetime.
func initDB(db *sql.DB) error {
	log.Debug("Initializing backend db")
	deployStatements(db, generateSQLBase)
	return nil
}

// deployStatements will issue given sql queries that are not already known to be deployed.
// This iterates both lists (to-run and already-deployed) and also verifies no contraditions.
func deployStatements(db *sql.DB, queries []string) error {
	tx, err := db.Begin()
	if err != nil {
		log.Fatale(err)
	}
	// Ugly workaround ahead.
	// Origin of this workaround is the existence of some "timestamp NOT NULL," column definitions,
	// where in NO_ZERO_IN_DATE,NO_ZERO_DATE sql_mode are invalid (since default is implicitly "0")
	// This means installation of orchestrator fails on such configured servers, and in particular on 5.7
	// where this setting is the dfault.
	// For purpose of backwards compatability, what we do is force sql_mode to be more relaxed, create the schemas
	// along with the "invalid" definition, and then go ahead and fix those definitions via following ALTER statements.
	// My bad.
	originalSqlMode := ""
	err = tx.QueryRow(`select @@session.sql_mode`).Scan(&originalSqlMode)
	if _, err := tx.Exec(`set @@session.sql_mode=REPLACE(@@session.sql_mode, 'NO_ZERO_DATE', '')`); err != nil {
		log.Fatale(err)
	}
	if _, err := tx.Exec(`set @@session.sql_mode=REPLACE(@@session.sql_mode, 'NO_ZERO_IN_DATE', '')`); err != nil {
		log.Fatale(err)
	}

	for i, query := range queries {
		if i == 0 {
			//log.Debugf("sql_mode is: %+v", originalSqlMode)
		}

		if _, err := tx.Exec(query); err != nil {
			if strings.Contains(err.Error(), "syntax error") {
				return log.Fatalf("Cannot initiate backend db: %+v; query=%+v", err, query)
			}
			if !sqlutils.IsAlterTable(query) && !sqlutils.IsCreateIndex(query) && !sqlutils.IsDropIndex(query) {
				return log.Fatalf("Cannot initiate backend db: %+v; query=%+v", err, query)
			}
			if !strings.Contains(err.Error(), "duplicate column name") &&
				!strings.Contains(err.Error(), "Duplicate column name") &&
				!strings.Contains(err.Error(), "check that column/key exists") &&
				!strings.Contains(err.Error(), "already exists") &&
				!strings.Contains(err.Error(), "Duplicate key name") {
				log.Errorf("Error initiating backend db: %+v; query=%+v", err, query)
			}
		}
	}

	if _, err := tx.Exec(`set session sql_mode=?`, originalSqlMode); err != nil {
		log.Fatale(err)
	}

	if err := tx.Commit(); err != nil {
		log.Fatale(err)
	}
	return nil
}
