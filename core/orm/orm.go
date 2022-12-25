package orm

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/encryption/hash"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/safemap"
)

var (
	Debug              = false
	UseCache           = true
	databases          = []DatabaseEntity{}
	mDbNameConnection  = map[string]*sql.DB{}
	mDbNameDialect     = map[string]string{}
	mModelTablename    = map[any]string{}
	cacheGetAllTables  = safemap.New[string, []string]()
	cacheGetAllColumns = safemap.New[string, map[string]string]()
	cachesOneM         = safemap.New[dbCache, map[string]any]()
	cachesAllM         = safemap.New[dbCache, []map[string]any]()
	DefaultDB          = ""
)

const (
	MIGRATION_FOLDER = "migrations"
	CACHE_TOPIC      = "internale-db-cache"
)
const (
	SQLITE   = "sqlite"
	POSTGRES = "postgres"
	MYSQL    = "mysql"
	MARIA    = "maria"
)

type TableEntity struct {
	Pk         string
	Name       string
	Columns    []string
	Types      map[string]string
	ModelTypes map[string]string
	Tags       map[string][]string
}

type DatabaseEntity struct {
	Name    string
	Conn    *sql.DB
	Dialect string
	Tables  []TableEntity
}

func InitDB() error {
	var err error
	var dsn string
	if settings.Config.Db.DSN == "" {
		settings.Config.Db.Type = SQLITE
		if settings.Config.Db.Name == "" {
			settings.Config.Db.Name = "db"
		}
	}
	if strings.HasPrefix(settings.Config.Db.Type, "cockroach") {
		settings.Config.Db.Type = POSTGRES
	}

	switch settings.Config.Db.Type {
	case POSTGRES:
		dsn = fmt.Sprintf("postgres://%s/%s?sslmode=disable", settings.Config.Db.DSN, settings.Config.Db.Name)
	case MYSQL, MARIA, "mariadb":
		if strings.Contains(settings.Config.Db.DSN, "tcp(") {
			dsn = settings.Config.Db.DSN + "/" + settings.Config.Db.Name
		} else {
			split := strings.Split(settings.Config.Db.DSN, "@")
			if len(split) > 2 {
				return errors.New("there is 2 or more @ symbol in dsn")
			}
			dsn = split[0] + "@" + "tcp(" + split[1] + ")/" + settings.Config.Db.Name
		}
		settings.Config.Db.Type = MARIA
	case SQLITE, "sqlite3":
		dsn = settings.Config.Db.Name + ".sqlite?_pragma=foreign_keys(1)"
		if settings.Config.Db.Name == "" {
			dsn = "db.sqlite?_pragma=foreign_keys(1)"
		}
	default:
		dsn = settings.Config.Db.Name + ".sqlite?_pragma=foreign_keys(1)"
		if settings.Config.Db.Name == "" {
			dsn = "db.sqlite?_pragma=foreign_keys(1)"
		}
	}
	isMariaDb := settings.Config.Db.Type == MARIA || settings.Config.Db.Type == "mariadb"
	DefaultDB = settings.Config.Db.Name
	dbType := settings.Config.Db.Type
	if isMariaDb {
		dbType = "mysql"
	}
	dbConn, err := sql.Open(dbType, dsn)
	if logger.CheckError(err) {
		return err
	}
	err = dbConn.Ping()
	if logger.CheckError(err) {
		logger.Info("check if env is loaded", dsn)
		return err
	}
	// if db exist return
	_, err = GetMemoryDatabase(DefaultDB)
	if err == nil {
		return nil
	}

	databases = append(databases, DatabaseEntity{
		Name:    DefaultDB,
		Conn:    dbConn,
		Dialect: settings.Config.Db.Type,
		Tables:  []TableEntity{},
	})
	mDbNameConnection[DefaultDB] = dbConn
	mDbNameDialect[DefaultDB] = settings.Config.Db.Type

	dbConn.SetMaxOpenConns(5)
	dbConn.SetMaxIdleConns(2)
	dbConn.SetConnMaxLifetime(30 * time.Minute)
	dbConn.SetConnMaxIdleTime(10 * time.Second)
	eventbus.Subscribe(CACHE_TOPIC, func(data map[string]string) {
		handleCache(data)
	})

	go utils.RunEvery(30*time.Minute, func() {
		eventbus.Publish(CACHE_TOPIC, map[string]string{
			"type": "clean",
		})
	})
	return nil
}

func NewDatabaseFromDSN(dbType, dbName string, dbDSN ...string) error {
	var dsn string
	if strings.HasPrefix(dbType, "cockroach") {
		dbType = POSTGRES
	}
	switch dbType {
	case POSTGRES:
		if len(dbDSN) == 0 {
			return errors.New("dbDSN for mysql cannot be empty")
		}
		dsn = fmt.Sprintf("postgres://%s/%s?sslmode=disable", dbDSN[0], dbName)
	case MYSQL:
		if len(dbDSN) == 0 {
			return errors.New("dbDSN for mysql cannot be empty")
		}
		if strings.Contains(dbDSN[0], "tcp(") {
			dsn = dbDSN[0] + "/" + dbName
		} else {
			split := strings.Split(dbDSN[0], "@")
			if len(split) > 2 {
				return errors.New("there is 2 or more @ symbol in dsn")
			}
			dsn = split[0] + "@" + "tcp(" + split[1] + ")/" + dbName
		}
	case MARIA, "mariadb":
		if len(dbDSN) == 0 {
			return errors.New("dbDSN for mysql cannot be empty")
		}
		if strings.Contains(dbDSN[0], "tcp(") {
			dsn = dbDSN[0] + "/" + dbName
		} else {
			split := strings.Split(dbDSN[0], "@")
			if len(split) > 2 {
				return errors.New("there is 2 or more @ symbol in dsn")
			}
			dsn = split[0] + "@" + "tcp(" + split[1] + ")/" + dbName
		}
		settings.Config.Db.Type = MARIA
	case SQLITE, "":
		if dsn == "" {
			dsn = "db.sqlite"
		}
		if !strings.Contains(dbName, SQLITE) {
			dsn = dbName + ".sqlite"
		} else {
			dsn = dbName
		}
	default:
		logger.Info(dbType, "not handled, choices are: postgres,mysql,sqlite")
		dsn = dbName + ".sqlite"
		if dsn == "" {
			dsn = "db.sqlite"
		}
	}
	if dbType == SQLITE {
		dsn += "?_pragma=foreign_keys(1)"
	}

	if dbType == MARIA || dbType == "mariadb" {
		dbType = "mysql"
	}
	conn, err := sql.Open(dbType, dsn)
	if logger.CheckError(err) {
		return err
	}
	err = conn.Ping()
	if logger.CheckError(err) {
		logger.Info("check if env is loaded", dsn)
		return err
	}
	for _, dbb := range databases {
		if dbb.Name == dbName {
			if err := conn.Close(); err != nil {
				logger.Error(err)
			}
			return errors.New("another database with the same name already registered")
		}
	}
	mDbNameConnection[dbName] = conn
	mDbNameDialect[dbName] = dbType
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(3 * time.Hour)
	conn.SetConnMaxIdleTime(10 * time.Second)
	return nil
}

func NewDatabaseFromConnection(dbType, dbName string, conn *sql.DB) error {
	if strings.HasPrefix(dbType, "cockroach") {
		dbType = POSTGRES
	}
	err := conn.Ping()
	if logger.CheckError(err) {
		return err
	}
	for _, dbb := range databases {
		if dbb.Name == dbName {
			if err := conn.Close(); err != nil {
				logger.Error(err)
			}
			return errors.New("another database with the same name already registered")
		}
	}
	if dbType == "mariadb" {
		settings.Config.Db.Type = MARIA
	}
	mDbNameConnection[dbName] = conn
	mDbNameDialect[dbName] = dbType
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(10 * time.Second)
	return nil
}

// GetConnection return default connection for orm.DefaultDatabase (if dbName not specified or empty or "default") else it return the specified one
func GetConnection(dbName ...string) *sql.DB {
	if len(dbName) > 0 {
		db, err := GetMemoryDatabase(dbName[0])
		if logger.CheckError(err) {
			return nil
		}
		return db.Conn
	} else {
		db, err := GetMemoryDatabase(settings.Config.Db.Name)
		if logger.CheckError(err) {
			return nil
		}
		return db.Conn
	}
}

func UseForAdmin(dbName string) {
	db, err := GetMemoryDatabase(dbName)
	if logger.CheckError(err) {
		logger.Error(dbName, "not found in connections list")
	} else {
		if db.Dialect != "" {
			if UseCache {
				eventbus.Publish(CACHE_TOPIC, map[string]string{
					"type":     "clean",
					"table":    "",
					"database": "",
				})
			}
			DefaultDB = db.Name
			settings.Config.Db.Name = db.Name
			settings.Config.Db.Type = db.Dialect
		}
	}
}

func GetAllTables(dbName ...string) []string {
	var name string
	if len(dbName) == 0 {
		name = settings.Config.Db.Name
	} else {
		name = dbName[0]
	}
	if UseCache {
		if v, ok := cacheGetAllTables.Get(name); ok {
			return v
		}
	}
	tables := []string{}
	tbs, err := GetMemoryTables(name)
	if err == nil {
		for _, t := range tbs {
			tables = append(tables, t.Name)
		}
		if len(tables) > 0 {
			if UseCache {
				cacheGetAllTables.Set(name, tables)
			}
			return tables
		}
	}

	conn := GetConnection(name)
	if conn == nil {
		logger.Error("connection is null")
		return nil
	}

	switch settings.Config.Db.Type {
	case POSTGRES:
		rows, err := conn.Query(`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname NOT IN ('pg_catalog','information_schema','crdb_internal','pg_extension') AND tableowner != 'node'`)
		if logger.CheckError(err) {
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			err := rows.Scan(&table)
			if logger.CheckError(err) {
				return nil
			}
			tables = append(tables, table)
		}
	case MYSQL, MARIA:
		rows, err := conn.Query("SELECT table_name,table_schema FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND table_schema ='" + name + "'")
		if logger.CheckError(err) {
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			var table_schema string
			err := rows.Scan(&table, &table_schema)
			if logger.CheckError(err) {
				return nil
			}
			tables = append(tables, table)
		}
	case SQLITE, "":
		rows, err := conn.Query(`SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';`)
		if logger.CheckError(err) {
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			err := rows.Scan(&table)
			if logger.CheckError(err) {
				return nil
			}
			tables = append(tables, table)
		}
	default:
		logger.Error("database type not supported, should be sqlite, postgres or mysql")
		os.Exit(0)
	}
	if UseCache {
		cacheGetAllTables.Set(name, tables)
	}
	return tables
}

func GetAllColumnsTypes(table string, dbName ...string) map[string]string {
	dName := settings.Config.Db.Name
	if len(dbName) > 0 {
		dName = dbName[0]
	}
	if UseCache {
		if v, ok := cacheGetAllColumns.Get(dName + "-" + table); ok {
			return v
		}
	}

	tb, err := GetMemoryTable(table, dName)
	if err == nil {
		if len(tb.Types) > 0 {
			if UseCache {
				cacheGetAllColumns.Set(dName+"-"+table, tb.Types)
			}
			return tb.Types
		}
	}

	dbType := settings.Config.Db.Type
	conn := GetConnection(dName)
	for _, d := range databases {
		if d.Name == dName {
			dbType = d.Dialect
			conn = d.Conn
		}
	}

	var statement string
	columns := map[string]string{}
	switch dbType {
	case POSTGRES:
		statement = "SELECT column_name,data_type FROM information_schema.columns WHERE table_name = '" + table + "'"
	case MYSQL, MARIA:
		statement = "SELECT column_name,data_type FROM information_schema.columns WHERE table_name = '" + table + "' AND TABLE_SCHEMA = '" + settings.Config.Db.Name + "'"
	default:
		statement = "PRAGMA table_info(" + table + ");"
		row, err := conn.Query(statement)
		if logger.CheckError(err) {
			return nil
		}
		defer row.Close()
		var num int
		var singleColName string
		var singleColType string
		var fake1 int
		var fake2 interface{}
		var fake3 int
		for row.Next() {
			err := row.Scan(&num, &singleColName, &singleColType, &fake1, &fake2, &fake3)
			if logger.CheckError(err) {
				return nil
			}
			columns[singleColName] = singleColType
		}
		if UseCache {
			cacheGetAllColumns.Set(dName+"-"+table, columns)
		}
		return columns
	}

	row, err := conn.Query(statement)

	if logger.CheckError(err) {
		return nil
	}
	defer row.Close()
	var singleColName string
	var singleColType string
	for row.Next() {
		err := row.Scan(&singleColName, &singleColType)
		if logger.CheckError(err) {
			return nil
		}
		columns[singleColName] = singleColType
	}
	if UseCache {
		cacheGetAllColumns.Set(dName+"-"+table, columns)
	}
	return columns
}

func GetMemoryTable(tbName string, dbName ...string) (TableEntity, error) {
	dName := settings.Config.Db.Name
	if len(dbName) > 0 {
		dName = dbName[0]
	}
	db, err := GetMemoryDatabase(dName)
	if err != nil {
		return TableEntity{}, err
	}
	for _, t := range db.Tables {
		if t.Name == tbName {
			return t, nil
		}
	}
	return TableEntity{}, errors.New("nothing found")
}

func GetMemoryTables(dbName ...string) ([]TableEntity, error) {
	dName := settings.Config.Db.Name
	if len(dbName) > 0 {
		dName = dbName[0]
	}
	db, err := GetMemoryDatabase(dName)
	if err != nil {
		return nil, err
	}
	return db.Tables, nil
}

func GetMemoryDatabases() []DatabaseEntity {
	return databases
}

// GetMemoryDatabase return the first connected database orm.DefaultDatabase if dbName "" or "default" else the matched db
func GetMemoryDatabase(dbName string) (*DatabaseEntity, error) {
	if DefaultDB == "" {
		DefaultDB = settings.Config.Db.Name
	}
	switch dbName {
	case "", "default":
		for i := range databases {
			if databases[i].Name == DefaultDB {
				return &databases[i], nil
			}
		}
		return nil, errors.New(dbName + "database not found")
	default:
		for i := range databases {
			if databases[i].Name == dbName {
				return &databases[i], nil
			}
		}
		return nil, errors.New(dbName + "database not found")
	}
}

func CreateUser(email, password string, isAdmin int, dbName ...string) error {
	if email == "" || password == "" {
		return errors.New("email and password cannot be empty")
	}
	uuid, err := GenerateUUID()
	if logger.CheckError(err) {
		return err
	}
	hash1, err := hash.GenerateHash(password)
	if logger.CheckError(err) {
		return err
	}
	name := settings.Config.Db.Name
	if len(dbName) > 0 {
		name = dbName[0]
	}
	_, err = Table("users").Database(name).Insert(
		"uuid,email,password,is_admin",
		[]any{uuid, email, hash1, isAdmin},
	)

	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}
