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

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

var (
	UseCache=true
	databases= []database{}
	mDbNameConnection = map[string]*sql.DB{}
	mDbNameDialect = map[string]string{}
	mModelTablename = map[any]string{}
	mModelDatabase = map[any]database{}
	cacheGetAllTables = safemap.New[string,[]string]()
	cacheGetAllColumns = safemap.New[string,map[string]string]()
	cachesOneM = safemap.New[dbCache,map[string]any]()
	cachesAllM = safemap.New[dbCache,[]map[string]any]()
)

const (
	MIGRATION_FOLDER = "migrations"
	CACHE_TOPIC = "internale-db-cache"
)

type database struct {
	name string
	conn *sql.DB
	dialect string
}

func InitDB() (error) {
	var err error
	var dsn string
	if settings.GlobalConfig.DbDSN == "" {
		if os.Getenv("DB_TYPE") != "" {
			settings.GlobalConfig.DbType=os.Getenv("DB_TYPE")
		} else {
			settings.GlobalConfig.DbType="sqlite"
		}
	}

	if settings.GlobalConfig.DbName == "" {
		if os.Getenv("DB_NAME") != "" {
			settings.GlobalConfig.DbName=os.Getenv("DB_NAME")
		}
		if settings.GlobalConfig.DbType == "sqlite" {
			if settings.GlobalConfig.DbName == "" {
				settings.GlobalConfig.DbName="db"
			}
		}
	}
	switch settings.GlobalConfig.DbType {
	case "postgres":
		dsn = fmt.Sprintf("postgres://%s/%s?sslmode=disable",settings.GlobalConfig.DbDSN,settings.GlobalConfig.DbName)
	case "mysql":
		if strings.Contains(settings.GlobalConfig.DbDSN,"tcp(") {
			dsn = settings.GlobalConfig.DbDSN + "/"+ settings.GlobalConfig.DbName
		} else {
			split := strings.Split(settings.GlobalConfig.DbDSN,"@")
			if len(split) > 2 {
				return errors.New("there is 2 or more @ symbol in dsn")
			}
			dsn = split[0]+"@"+"tcp("+split[1]+")/"+ settings.GlobalConfig.DbName
		}		
	case "sqlite","":
		dsn = settings.GlobalConfig.DbName+".sqlite"
		if dsn == "" {dsn="db.sqlite"}
	default:
		dsn = settings.GlobalConfig.DbName+".sqlite"
		if dsn == "" {dsn="db.sqlite"}
	}
	dbConn, err := sql.Open(settings.GlobalConfig.DbType, dsn)
	if logger.CheckError(err) {
		return err
	}
	err = dbConn.Ping()
	if logger.CheckError(err) {
		logger.Info("check if env is loaded", dsn)
		return err
	}
	if settings.GlobalConfig.DbType == "sqlite" {
		_, err = dbConn.Exec(`PRAGMA foreign_keys = ON`)
		if logger.CheckError(err) {
			return err
		}
	}
	
	databases = append(databases, database{
		name: settings.GlobalConfig.DbName,
		conn: dbConn,
		dialect: settings.GlobalConfig.DbType,
	})
	mDbNameConnection[settings.GlobalConfig.DbName]=dbConn
	mDbNameDialect[settings.GlobalConfig.DbName]=settings.GlobalConfig.DbType
	dbConn.SetMaxOpenConns(5)
	dbConn.SetMaxIdleConns(2)
	dbConn.SetConnMaxLifetime(30 * time.Minute)
	dbConn.SetConnMaxIdleTime(10 * time.Second)
	eventbus.Subscribe(CACHE_TOPIC,func(data map[string]string) {
		handleCache(data)
	})

	go utils.RunEvery(30 * time.Minute,func(){
		eventbus.Publish(CACHE_TOPIC,map[string]string{
			"type":"clean",
		})
	})
	return nil
}

func NewDb(dbType,dbName,dbDSN string) (error) {
	var dsn string
	switch dbType {
	case "postgres":
		dsn = fmt.Sprintf("postgres://%s/%s?sslmode=disable",dbDSN,dbName)
	case "mysql":
		if strings.Contains(dbDSN,"tcp(") {
			dsn = dbDSN + "/"+ dbName
		} else {
			split := strings.Split(dbDSN,"@")
			if len(split) > 2 {
				return errors.New("there is 2 or more @ symbol in dsn")
			}
			dsn = split[0]+"@"+"tcp("+split[1]+")/"+ dbName
		}		
	case "sqlite","":
		dsn = dbName+".sqlite"
		if dsn == "" {dsn="db.sqlite"}
	default:
		logger.Info(dbType,"not handled, choices are: postgres,mysql,sqlite")
		dsn = dbName+".sqlite"
		if dsn == "" {dsn="db.sqlite"}
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
	for _,dbb := range databases {
		if dbb.name == dbName {
			logger.Error("another database with the same name already registered")
			if err := conn.Close();err != nil {
				logger.Error(err)
			}
			return errors.New("another database with the same name already registered")
		}
	}
	databases = append(databases, database{
		name: dbName,
		conn: conn,
		dialect: dbType,
	})
	mDbNameConnection[dbName]=conn
	mDbNameDialect[dbName]=dbType
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)
	conn.SetConnMaxIdleTime(10 * time.Second)
	return nil
}

func GetConnection(dbName ...string) *sql.DB {
	if len(dbName) > 0 {
		for _,db := range databases {
			if db.name == dbName[0] {
				return db.conn
			}
		}
	}
	if len(databases) > 0 {
		return databases[0].conn
	}
	return nil
}

func GetAllTables(dbName ...string) []string {
	var name string
	if len(dbName) == 0 {
		name=settings.GlobalConfig.DbName
	} else {
		name=dbName[0]
	}
	if UseCache {
		if v,ok := cacheGetAllTables.Get(name);ok {
			return v
		}
	}

	conn := GetConnection(dbName...)
	
	tables := []string{}
	switch settings.GlobalConfig.DbType {
	case "postgres":
		rows,err := conn.Query(`SELECT tablename FROM pg_catalog.pg_tables WHERE schemaname != 'pg_catalog' AND schemaname != 'information_schema';`)
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
			tables = append(tables,table)
		}
	case "mysql":
		rows,err := conn.Query("SELECT table_name,table_schema FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_TYPE = 'BASE TABLE' AND table_schema ='"+name+"'")
		if logger.CheckError(err) {
			return nil
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			var table_schema string
			err := rows.Scan(&table,&table_schema)
			if logger.CheckError(err) {
				return nil
			}
			tables = append(tables,table)
		}
	case "sqlite","":
		rows,err := conn.Query(`SELECT name FROM sqlite_schema WHERE type ='table' AND name NOT LIKE 'sqlite_%';`)
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
			tables = append(tables,table)
		}
	default:
		logger.Error("database type not supported, should be sqlite, postgres or mysql")
		os.Exit(0)
	}
	if UseCache {
		cacheGetAllTables.Set(name,tables)
	}
	return tables
}

func GetAllColumns(table string, dbName ...string) map[string]string {
	dName := settings.GlobalConfig.DbType
	if len(dbName) > 0 {
		dName=dbName[0]
	}
	if UseCache {
		if v,ok := cacheGetAllColumns.Get(dName+"-"+table);ok {
			return v
		}
	}

	dbType := settings.GlobalConfig.DbType
	conn := GetConnection()
	
	
	for _,d := range databases {
		if d.name == dName {
			dbType=d.dialect
			conn=d.conn
		}
	}

	var statement string
	columns := map[string]string{}
	switch dbType {
	case "postgres":
		statement = "SELECT column_name,data_type FROM information_schema.columns WHERE table_name = '"+table+"'"
	case "mysql":
		statement = "SELECT column_name,data_type FROM information_schema.columns WHERE table_name = '"+table+"' AND TABLE_SCHEMA = '"+settings.GlobalConfig.DbName+"'"
	default:
		statement = "PRAGMA table_info("+table+");"
		row,err := conn.Query(statement)
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
			err := row.Scan(&num,&singleColName,&singleColType,&fake1,&fake2,&fake3)
			if logger.CheckError(err) {
				return nil
			}
			columns[singleColName]=singleColType
		}
		if UseCache {
			cacheGetAllColumns.Set(dName+"-"+table,columns)
		}
		return columns
	}


	row,err := conn.Query(statement)

	if logger.CheckError(err) {
		return nil
	}
	defer row.Close()
	var singleColName string
	var singleColType string
	for row.Next() {
		err := row.Scan(&singleColName,&singleColType)
		if logger.CheckError(err) {
			return nil
		}
		columns[singleColName]=singleColType
	}
	if UseCache {
		cacheGetAllColumns.Set(dName+"-"+table,columns)
	}
	return columns
}

func CreateUser(email,password string,isAdmin int) error {
	if email == "" || password == "" {
		return errors.New("email and password cannot be empty")
	}
	uuid,err := GenerateUUID()
	if logger.CheckError(err) {
		return err
	}
	hash1, err := hash.GenerateHash(password)
	if logger.CheckError(err) {
		return err
	}

	_,err = Database().Table("users").Insert(
		"uuid,email,password,is_admin",
		uuid,
		email,
		hash1,
		isAdmin,
	)
	
	if err != nil {
		logger.Error(err)
		return err
	}
	return nil
}

func handleCache(data map[string]string) {
	switch data["type"] {
	case "create","delete","update":
		go func() {	
			cachesAllM.Flush()
			cachesAllS.Flush()
			cachesOneM.Flush()
			cachesOneS.Flush()
		}()		
	case "drop":
		go func() {
			if v,ok := data["table"];ok {
				cacheGetAllColumns.Delete(v)
			}
			cacheGetAllTables.Flush()
			cachesAllM.Flush()
			cachesAllS.Flush()
			cachesOneM.Flush()
			cachesOneS.Flush()
		}()
	case "clean":
		go func() {
			cacheGetAllColumns.Flush()
			cacheGetAllTables.Flush()
			cachesAllM.Flush()
			cachesAllS.Flush()
			cachesOneM.Flush()
			cachesOneS.Flush()
		}()
	default:
		logger.Info("CACHE DB: default case triggered",data)
	}
}
