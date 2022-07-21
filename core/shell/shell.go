package shell

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/orm"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/input"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

const helpS string = `Commands :  
[databases, use, tables, columns, migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, q/quit/exit, help/commands]
  'databases':
	  list all connected databases

  'use':
	  use a specific database

  'tables':
	  list all tables in database

  'columns':
	  list all columns of a table

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

  'clear/cls':
	  clear console
`

const commandsS string = "Commands :  [databases, use, tables, columns, migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, q!/quit/exit, help/commands]"

// InitShell init the shell and return true if used to stop main
func InitShell() bool {
	args := os.Args
	if len(args) <2 {
		return false
	}
	
	switch args[1] {
	case "commands":
		fmt.Printf(logger.Yellow,"Shell Usage: go run main.go shell")
		fmt.Printf(logger.Yellow,commandsS)
		return true
	case "help":
		fmt.Printf(logger.Yellow,"Shell Usage: go run main.go shell")
		fmt.Printf(logger.Yellow,helpS)
		return true
	case "shell":
		databases := orm.GetDatabases()
		var conn *sql.DB
		if len(databases) > 1 {
			fmt.Printf(logger.Yellow,"-----------------------------------")
			fmt.Printf(logger.Blue,"Found many databases:")
			for _,db := range databases {
				fmt.Printf(logger.Blue,`  - `+db.Name)
			}
			dbName,err := input.String(input.Blue,"Enter Database Name to use: ")
			if logger.CheckError(err) {
				return true
			}
			if dbName == "" {return true}
			orm.UseForAdmin(dbName)
			conn = orm.GetConnection(dbName)
		} else {
			conn = orm.GetConnection()
		}
		defer conn.Close()

		fmt.Printf(logger.Yellow,commandsS)
		for {
			command,err := input.String(input.Blue,"> ")
			if err != nil {
				if errors.Is(err,io.EOF) {
					fmt.Printf(logger.Blue,"shell shutting down")
				}
				return true
			}

			switch command {
			case "quit","exit","q","q!":
				return true		
			case "clear","cls":
				input.Clear()	
				fmt.Printf(logger.Yellow,"Commands :  [migrate, createsuperuser, createuser, getall, get, drop, delete, clear/cls, quit/exit, help/commands]")
			case "help":
				fmt.Printf(logger.Yellow,helpS)
			case "commands":
				fmt.Printf(logger.Yellow,commandsS)
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
			case "databases":
				fmt.Printf(logger.Green,orm.GetDatabases())
			case "use":
				db := input.Input(input.Blue,"database name: ")
				orm.UseForAdmin(db)
				fmt.Printf(logger.Green,"you are using database "+db)
			case "tables":
				fmt.Printf(logger.Green,orm.GetAllTables(settings.GlobalConfig.DbName)) 
			case "columns":
				tb := input.Input(input.Blue,"Table name: ")
				mcols := orm.GetAllColumns(tb,settings.GlobalConfig.DbName)
				cols := []string{}
				for k := range mcols {
					cols = append(cols, k)
				}
				fmt.Printf(logger.Green,cols) 
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
	case "push":
		if len(args) > 2 {
			pushGit(args[2])
		} else {
			logger.Error("version tag cannot be empty")
		}
		return true
	default:
		return false
	}
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
	password := input.Hidden(input.Blue,"Password : ")
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

func pushGit(version string) {
	if strings.TrimSpace(version)  == "" {
		logger.Error("version tag cannot be empty")
		return
	} else {
		fmt.Printf(logger.Green,"VERSION: "+version)
	}
	if !strings.HasPrefix(version,"v") {
		version="v"+version
	}

	fmt.Printf(logger.Blue,"pushing to git repos:")
	fmt.Printf(input.Blue,"git add .")
	err := exec.Command("git", "add", ".").Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")
	
	fmt.Printf(input.Blue,"git reset kago.go")
	err = exec.Command("git", "reset", "kago.go").Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")

	commit,err := input.String(input.Blue,"commit message: ")
	if logger.CheckError(err) {return}
	fmt.Printf(input.Blue,"git commit -m \""+commit+"\"")
	err = exec.Command("git", "commit", "-m","\""+commit+"\"").Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")

	fmt.Printf(input.Blue,"git tag "+version)
	err = exec.Command("git", "tag", version).Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")

	fmt.Printf(input.Blue,"git push origin "+version)
	err = exec.Command("git", "push", "origin",version).Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")

	fmt.Printf(input.Blue,"git push")
	err = exec.Command("git", "push").Run()
	if logger.CheckError(err) {return}
	fmt.Printf(logger.Green,"  --> DONE")
	fmt.Printf(logger.Green,"pushed successfully to "+version)
}