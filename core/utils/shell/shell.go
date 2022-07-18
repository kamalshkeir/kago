package shell

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
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
	case "commands":
		fmt.Printf(logger.Yellow,"Shell Usage: go run main.go shell")
		fmt.Printf(logger.Yellow,"Commands :  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]")
		return true
	case "help":
		fmt.Printf(logger.Yellow,"Shell Usage: go run main.go shell")
		fmt.Printf(logger.Yellow,`Commands :  
  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]
	
	'migrate':
		migrate initial users to env database

	'createsuperuser':
		create a admin user

	'createuser':
		create a regular user

	'getall':
		get all rows given a table name

	'get':
		get single row wher field equal_to

	'delete':
		delete rows where field equal_to

	'drop':
		drop a table given table name
				`)
		return true
	case "shell":
		_ = orm.InitDB()
		defer orm.GetConnection().Close()
		fmt.Printf(logger.Yellow,"Commands :  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]")
		for {
			command,err := input.String(input.Blue,"> ")
			if err != nil {
				if errors.Is(err,io.EOF) {
					fmt.Printf(logger.Blue,"shell shutting down")
					os.Exit(0)
				}
				return true
			}
			switch command {
			case "quit","exit":
				return true		
			case "clear","cls":
				input.Clear()	
				fmt.Printf(logger.Yellow,"Commands :  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]")
			case "help":
		fmt.Printf(logger.Yellow,`Commands :  
  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]
	
	'migrate':
		migrate initial users to env database

	'createsuperuser':
		create a admin user

	'createuser':
		create a regular user

	'getall':
		get all rows given a table name

	'get':
		get single row wher field equal_to

	'delete':
		delete rows where field equal_to
		
	'drop':
		drop a table given table name
				`)
			case "commands":
				fmt.Printf(logger.Yellow,"Commands :  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]")
			case "migrate":
				fmt.Printf(logger.Blue,"available commands:")
				fmt.Printf(logger.Blue,"1 : init")
				fmt.Printf(logger.Blue,"2 : file")
				choice := input.Input(input.Blue,"command> ")
				switch choice {
				case "file","2":
					path := input.Input(input.Blue,"path: ")
					err := migratefromfile(path)
					if !logger.CheckError(err) {
						fmt.Printf(logger.Green,"migrated successfully")
					}
				case "init","1":
					err := orm.Migrate()
					if !logger.CheckError(err) {
						fmt.Printf(logger.Green,"users table migrated successfully")
					}
				}
			case "createsuperuser":
				createsuperuser()
			case "createuser":
				createuser()
			case "getall":
				getAll()	
			case "get":		
				getRow()			
			case "drop":
				dropTable()	
			case "delete":
				deleteRow()	
			default:
				fmt.Printf(logger.Red,"command not handled, use 'help' or 'commands' to list available commands ")
			}
		}
	default:
		fmt.Printf(logger.Red,"command not handled, available commands : 'shell' , 'help', 'commands'")	
	}
	return true
}

func getAll() {
	tableName,err := input.String(input.Blue,"Enter a table name: ")
	if err == nil {
		data,err := orm.Table(tableName).Database(settings.GlobalConfig.DbName).All()
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
		data,err = orm.Table(tableName).Database(settings.GlobalConfig.DbName).Where(whereField+" = ?",equalTo).One()
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
	if !utils.SliceContains([]string{"postgres","sqlite","mysql"},settings.GlobalConfig.DbType) {
		logger.Error("database is neither postgres, sqlite or mysql ")
		return errors.New("database is neither postgres, sqlite or mysql ")
	}
	if path == "" {
		logger.Error("path cannot be empty ")
		return errors.New("path cannot be empty ")
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
		_,err := orm.Table(tableName).Database(settings.GlobalConfig.DbName).Drop()
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
			_,err := orm.Table(tableName).Database(settings.GlobalConfig.DbName).Where(whereField + " = ?",equalTo).Delete()
			if err == nil {
				fmt.Printf(logger.Green,tableName+"with"+whereField+"="+equalTo+"deleted.")
			} else {
				fmt.Printf(logger.Red,"error deleting row: "+err.Error())
			}
		} else {
			_,err = orm.Table(tableName).Where(whereField+" = ?",equal).Delete()
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