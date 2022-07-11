package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		fmt.Printf(input.Blue,"available commands: 1:init, 2:file \n")
		choice := input.Input(input.Blue,"command> ")
		switch choice {
		case "file","2":
			path := input.Input(input.Blue,"path: ")
			err := migratefromfile(path)
			if logger.CheckError(err) {
				return true
			}
		case "init","1":
			err := orm.Migrate()
			if logger.CheckError(err) {
				return true
			}
			fmt.Printf(logger.Green,"users table migrated successfully")
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