package shell

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/input"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

// InitShell init the shell and return true if used to stop main
func InitShell() bool {
	args := os.Args
	if len(args) <2 {
		return false
	}
	
	switch args[1] {
	case "help":
		fmt.Printf(logger.Blue,"Shell Usage: go run main.go [migrate, createsuperuser, createuser, getall, get, drop, delete]")
		return true
	case "migrate":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		fmt.Printf(input.Blue,"available commands: 1:new, 2:up, 3:down, 4:file, 5:init \n")
		choice := input.Input(input.Blue,"command> ")
		switch choice {
		case "new","1":
			newmigration()
		case "up","2":
			migrationup()
		case "down","3":
			migrationdown()
		case "file","4":
			path := input.Input(input.Blue,"path: ")
			err := migratefromfile(path)
			if logger.CheckError(err) {
				return true
			}
		case "init","5":
			err := orm.Migrate()
			if logger.CheckError(err) {
				return true
			}
			fmt.Printf(logger.Green,"Users and schema_migrations migrated successfully")
		}
	case "createsuperuser":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		createsuperuser()
	case "createuser":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		createuser()				
	case "getall":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		getAll()	
	case "get":	
		_ = orm.InitDB()
		defer orm.GetConnection().Close()	
		getRow()			
	case "drop":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		dropTable()	
	case "delete":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		deleteRow()	
	default:
		return false	
	}
	return true
}

func getAll() {
	tableName,err := input.String(input.Blue,"Enter a table name: ")
	if err == nil {
		data,err := orm.Database().Table(tableName).All()
		if err == nil {
			d,_ := json.MarshalIndent(data,"","    ")
			fmt.Printf(logger.Green,string(d))
		} else {
			fmt.Printf(logger.Red,err.Error())
		}
	} else {
		fmt.Printf(logger.Red,"table name invalid")
	}
}

func getRow() {
	tableName := input.Input(input.Blue,"Table Name : ") 
	whereField := input.Input(input.Blue,"Where field : ") 
	equalTo := input.Input(input.Blue,"Equal to : ") 
	if tableName != "" && whereField != "" && equalTo != ""{
		var data map[string]interface{}
		var err error
		data,err = orm.Database().Table(tableName).Where(whereField+" = ?",equalTo).One()
		if err == nil {
			d,_ := json.MarshalIndent(data,"","    ")
			fmt.Printf(logger.Green,string(d))
		} else {
			fmt.Printf(logger.Red,"error: "+err.Error())
		}
	} else {
		fmt.Printf(logger.Red,"One or more field are empty")
	}
}

func createuser() {
	email := input.Input(input.Blue,"Email : ")
	password := input.Input(input.Blue,"Password : ")
	if email != "" && password != "" {
		err := orm.CreateUser(email,password,0)
		if err == nil {
			fmt.Printf(logger.Green,"User "+email+" created successfully")
		} else {
			fmt.Printf(logger.Red,"unable to create user:"+err.Error())
		}
	} else {
		fmt.Printf(logger.Red,"email or password invalid")
	}
}

func createsuperuser() {
	email := input.Input(input.Blue,"Email: ")
	password := input.Hidden(input.Blue,"Password: ")
	err := orm.CreateUser(email,password,1)
	if err==nil {
		fmt.Printf(logger.Green,"User "+email+" created successfully")
	} else {
		fmt.Printf(logger.Red,"error creating user :"+err.Error())
	}
}


func migratefromfile(path string) error {
	if settings.GlobalConfig.DbType != "postgres" && settings.GlobalConfig.DbType != "sqlite" && settings.GlobalConfig.DbType != "mysql" {
		logger.Error("database is neither postgres, sqlite or mysql !")
		return errors.New("database is neither postgres, sqlite or mysql !")
	}
	if path == "" {
		logger.Error("path cannot be empty !")
		return errors.New("path cannot be empty !")
	}
	statements := []string{}
	b,err := os.ReadFile(path)
	if err != nil {
		return errors.New("error reading from " +path +" "+err.Error())
	}
	splited := strings.Split(string(b),";")
	statements = append(statements, splited...)

	//exec migrations
	for i := range statements {
		_,err := orm.GetConnection().Exec(statements[i])
		if err != nil {
			return errors.New("error migrating from "+path + " "+ err.Error())
		}
	}
	return nil
}

var newmigrationWG sync.WaitGroup
func newmigration() {
	if settings.GlobalConfig.DbType != "postgres" && settings.GlobalConfig.DbType != "sqlite" && settings.GlobalConfig.DbType != "mysql" {
		logger.Error("database is neither postgres, sqlite or mysql !")
		os.Exit(1)
	}
	if _,err := os.Stat(orm.MIGRATION_FOLDER);err != nil {
		err := os.Mkdir(orm.MIGRATION_FOLDER,0770)
		if err != nil {
			logger.Error(err)
			os.Exit(1)
		}
	}

	var version int
	var up_path string
	var down_path string
	mg_name := input.Input(input.Blue,"migration name : ")
	if mg_name == "" {
		logger.Error("migration name cannot be empty !")
		mg_name = input.Input(input.Blue,"migration name : ")
	}

	//check if migration with the same name exist in orm
	s,err := orm.Database().Table("schema_migrations").Where("name = ?",mg_name).One()
	if err == nil {
		logger.Error("schema_migrations with the same name already exist",s)
		mg_name = input.Input(input.Blue,"migration name : ")
	}
	
	// get last version
	row := orm.GetConnection().QueryRow("select version from schema_migrations order by -version limit 1")
	err = row.Scan(&version)
	if errors.Is(err,sql.ErrNoRows) {
		up_path = orm.MIGRATION_FOLDER + fmt.Sprintf("/%d-%s-up.sql",1,mg_name)
		down_path = orm.MIGRATION_FOLDER + fmt.Sprintf("/%d-%s-down.sql",1,mg_name)
		newmigrationWG.Add(3)
		go func() {
			_,err := orm.Database().Table("schema_migrations").Insert(
				"version,name,up_path,down_path,executed",
				1,mg_name,up_path,down_path,false,
			)
			logger.CheckError(err)
			newmigrationWG.Done()
		}()

		go func() {
			defer newmigrationWG.Done()
			f,err := os.Create(up_path)
			logger.CheckError(err)
			defer f.Close()
		}()

		go func() {
			defer newmigrationWG.Done()
			f,err := os.Create(down_path)
			logger.CheckError(err)
			defer f.Close()
		}()
		newmigrationWG.Wait()
	} else if err != nil {
		logger.Error("row scan error :",err)
		os.Exit(1)
	} else {
		up_path = orm.MIGRATION_FOLDER + fmt.Sprintf("/%d-%s-up.sql",version+1,mg_name)
		down_path = orm.MIGRATION_FOLDER + fmt.Sprintf("/%d-%s-down.sql",version+1,mg_name)
		newmigrationWG.Add(3)
		go func() {
			_,err := orm.Database().Table("schema_migrations").Insert(
				"version,name,up_path,down_path,executed",
				version+1,mg_name,up_path,down_path,false,
			)
			logger.CheckError(err)
			newmigrationWG.Done()
		}()

		go func() {
			defer newmigrationWG.Done()
			f,err := os.Create(up_path)
			logger.CheckError(err)
			defer f.Close()
		}()

		go func() {
			defer newmigrationWG.Done()
			f,err := os.Create(down_path)
			logger.CheckError(err)
			defer f.Close()
		}()
		newmigrationWG.Wait()		
	}
}

func migrationup() {
	if settings.GlobalConfig.DbType != "postgres" && settings.GlobalConfig.DbType != "sqlite" && settings.GlobalConfig.DbType != "mysql" {
		logger.Error("database is neither postgres, sqlite or mysql !")
		os.Exit(1)
	}
	if _,err := os.Stat(orm.MIGRATION_FOLDER);err != nil {
		logger.Error("migrations folder not found !, execute migrate new before migrate up")
		os.Exit(0)
	}
	migrations,err := orm.Database().Table("schema_migrations").All()
	if err != nil {
		logger.Error("getall migration error:",err)
		os.Exit(1)
	}

	notExecutedList := []map[string]interface{}{}
	for _,mg := range migrations {
		if mg["executed"] == int64(0) || mg["executed"] == 0 || mg["executed"] == false {
			notExecutedList = append(notExecutedList, mg)
		}
	}

	if len(notExecutedList)  > 0 {
		fmt.Printf(logger.Blue,"version | name")
		for _,mg := range notExecutedList {
			fmt.Printf(logger.Red,fmt.Sprintf("   %v    |  %v    DOWN",mg["version"],mg["name"]))
		}
		v := input.Input(input.Blue,"choose version to migrate up: ")
		if v == "" {return}
		mg,err := orm.Database().Table("schema_migrations").Where("version = ?",v).One()
		if logger.CheckError(err) {return}
		if path,ok := mg["up_path"];ok {
			if v,ok := path.(string);ok {
				migratefromfile(v)
				_,err := orm.Database().Table("schema_migrations").Where("up_path = ?",v).Set("executed = ?",1)
				if !logger.CheckError(err) {
					fmt.Printf(logger.Green,v+" Migrated Up successfully")
				}
			}
		} else {
			logger.Error("no up_path found in db for :",mg)
			return
		}
	} else {
		fmt.Printf(logger.Green,"All migrations are already executed, nothing change")
		return
	}
}

func migrationdown() {
	if settings.GlobalConfig.DbType != "postgres" && settings.GlobalConfig.DbType != "sqlite" && settings.GlobalConfig.DbType != "mysql" {
		logger.Error("database is neither postgres, sqlite or mysql !")
		os.Exit(1)
	}
	if _,err := os.Stat(orm.MIGRATION_FOLDER);err != nil {
		logger.Error("migrations folder not found !, execute migrate new before migrate up")
		os.Exit(0)
	}
	migrations,err := orm.Database().Table("schema_migrations").All()
	if err != nil {
		logger.Error("getall migration error:",err)
		os.Exit(1)
	}
	executedList := []map[string]interface{}{}
	for _,mg := range migrations {
		if mg["executed"] == 1 || mg["executed"] == int64(1) || mg["executed"] == true  {
			executedList = append(executedList, mg)
		} 
	}

	if len(executedList)  > 0 {
		fmt.Printf(logger.Blue,"version | name")
		for _,mg := range executedList {
			fmt.Printf(logger.Green,fmt.Sprintf("   %v    |  %v   UP",mg["version"],mg["name"]))
		}
		v := input.Input(input.Blue,"choose version to migrate down: ")
		if v == "" {return}
		mg,err := orm.Database().Table("schema_migrations").Where("version = ?",v).One()
		if logger.CheckError(err) {return}
		if path,ok := mg["down_path"];ok {
			if v,ok := path.(string);ok {
				migratefromfile(v)
				_,err := orm.Database().Table("schema_migrations").Where("down_path = ?",v).Set("executed = ?",0)
				if !logger.CheckError(err) {
					fmt.Printf(logger.Green,v+" Migrated Down successfully")
				}
			}
		} else {
			logger.Error("no down_path found in db for :",mg)
			return
		}
	} else {
		fmt.Printf(logger.Green,"All migrations are already executed, nothing change")
		return
	}
}

func dropTable() {
	tableName := input.Input(input.Blue,"Table to drop : ") 
	if tableName != "" {
		_,err := orm.Database().Table(tableName).Drop()
		if err != nil {
			fmt.Printf(logger.Red,"error dropping table :"+err.Error())
		} else {
			fmt.Printf(logger.Green,tableName+" dropped with success")
		}
	} else {
		fmt.Printf(logger.Red,"table is empty")
	}
}

func deleteRow() {
	tableName := input.Input(input.Blue,"Table Name: ")
	whereField := input.Input(input.Blue,"Where Field: ")
	equalTo := input.Input(input.Blue,"Equal to: ")
	if tableName != "" && whereField != "" && equalTo != "" {
		equal,err := strconv.Atoi(equalTo)
		if err != nil {
			_,err := orm.Database().Table(tableName).Where(whereField + " = ?",equalTo).Delete()
			if err == nil {
				fmt.Printf(logger.Green,tableName+"with"+whereField+"="+equalTo+"deleted.")
			} else {
				fmt.Printf(logger.Red,"error deleting row: "+err.Error())
			}
		} else {
			_,err = orm.Database().Table(tableName).Where(whereField+" = ?",equal).Delete()
			if err == nil {
				fmt.Printf(logger.Green,tableName+" with "+whereField+" = "+equalTo+" deleted.")
			} else {
				fmt.Printf(logger.Red,"error deleting row: "+err.Error())
			}
		}		
	} else {
		fmt.Printf(logger.Red,"some of args are empty")
	}
}