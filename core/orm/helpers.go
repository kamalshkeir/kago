package orm

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/input"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

type dbCache struct {
	limit      int
	page       int
	database   string
	table      string
	selected   string
	orderBys   string
	whereQuery string
	query      string
	offset     string
	statement  string
	args       string
}

// linkModel link a struct model to a  db_table_name
func linkModel[T comparable](to_table_name string, db *DatabaseEntity) {
	if db.Name == "" {
		db = &databases[0]
	}
	fields, _, ftypes, ftags := getStructInfos(new(T))
	// get columns from db
	colsNameType := GetAllColumns(to_table_name, db.Name)
	cols := []string{}
	for k := range colsNameType {
		cols = append(cols, k)
	}
	for _,list := range ftags {
		for i := range list {
			list[i]=strings.TrimSpace(list[i]) 
		}
	}

	diff := utils.Difference(fields, cols)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		// add or remove field from struct
		handleAddOrRemove[T](to_table_name, fields, cols, diff, db, ftypes, ftags)
		wg.Done()
	}()

	go func() {
		// rename field
		handleRename(to_table_name, fields, cols, diff, db,ftags)
		wg.Done()
	}()
	wg.Wait()
	pk := ""
	tFound := false
	for _, t := range db.Tables {
		if t.Name == to_table_name {
			tFound = true
		}
	}
	for col,tags := range ftags {
		for _,tag := range tags {
			if tag == "autoinc" || tag == "pk" {
				pk=col
				break
			}
		}
	}
	if !tFound {
		db.Tables = append(db.Tables, TableEntity{
			Name:       to_table_name,
			Columns:    cols,
			ModelTypes: ftypes,
			Types:      colsNameType,
			Tags:       ftags,
			Pk: pk,
		})
	}
}

// handleAddOrRemove handle sync with db when adding or removing from a struct auto migrated
func handleAddOrRemove[T comparable](to_table_name string, fields, cols, diff []string, db *DatabaseEntity, ftypes map[string]string, ftags map[string][]string) {
	if len(cols) > len(fields) { // extra column db
		for _, d := range diff {
			if v,ok := ftags[d];ok && v[0] == "-"{
				continue
			}
			fmt.Println(" ")
			logger.Printfs("⚠️ found extra column '%s' in the database table '%s'", d, to_table_name)

			statement := "ALTER TABLE " + to_table_name + " DROP COLUMN " + d

			choice := input.Input(input.Yellow, "> do you want to remove '"+d+"' from database ? (Y/n): ")
			if utils.SliceContains([]string{"yes", "Y", "y"}, choice) {
				sst := "DROP INDEX IF EXISTS idx_"+to_table_name+"_"+d
				if len(databases) > 1 && db.Name == "" {
					ddb := input.Input(input.Blue, "> There are more than one database connected, enter database name: ")
					conn := GetConnection(ddb)
					if conn != nil {						
						if Debug {
							logger.Info(sst)
							logger.Info(statement)
						}
						_, err := conn.Exec(sst)
						if logger.CheckError(err) {
							logger.Info(sst)
							return
						}
						_, err = conn.Exec(statement)
						if err != nil {
							temp := to_table_name+"_temp"
							err := autoMigrate[T](db,temp)
							if logger.CheckError(err) {return}
							cls := strings.Join(fields,",")
							_, err =conn.Exec("INSERT INTO "+temp+" SELECT "+cls+" FROM "+to_table_name)
							if logger.CheckError(err) {return}
							_,err = Table(to_table_name).Database(db.Name).Drop()
							if logger.CheckError(err) {return}
							_, err =conn.Exec("ALTER TABLE "+temp+" RENAME TO "+to_table_name)
							if logger.CheckError(err) {return}
						}
						logger.Printfs("grDone, '%s' removed from '%s'", d, to_table_name)
						os.Exit(0)
					}
				} else {
					conn := db.Conn
					if conn != nil {
						_, err := conn.Exec(sst)
						if logger.CheckError(err) {
							logger.Info(sst)
							return
						}
						if Debug {
							logger.Info(sst)
							logger.Info(statement)
						}
						_, err = conn.Exec(statement)
						if err != nil {
							temp := to_table_name+"_temp"
							err := autoMigrate[T](db,temp)
							if logger.CheckError(err) {return}
							cls := strings.Join(fields,",")
							_, err =conn.Exec("INSERT INTO "+temp+" SELECT "+cls+" FROM "+to_table_name)
							if logger.CheckError(err) {return}
							_,err = Table(to_table_name).Database(db.Name).Drop()
							if logger.CheckError(err) {return}
							_, err =conn.Exec("ALTER TABLE "+temp+" RENAME TO "+to_table_name)
							if logger.CheckError(err) {return}	
						}
						logger.Printfs("grDone, '%s' removed from '%s'", d, to_table_name)
						os.Exit(0)
					}
				}
			} else {
				fmt.Printf(logger.Green, "Nothing changed.")
			}
		}
	} else if len(cols) < len(fields) { // missing column db
		for _, d := range diff {
			if v,ok := ftags[d]; ok && v[0] == "-"{
				continue
			}
			fmt.Println(" ")
			logger.Printfs("⚠️ column '%s' is missing from the database table '%s'", d, to_table_name)
			choice, err := input.String(input.Yellow, "> do you want to add '"+d+"' to the database ? (Y/n):")
			logger.CheckError(err)
			statement := "ALTER TABLE " + to_table_name + " ADD " + d + " "
			if ty, ok := ftypes[d]; ok {			
				res := map[string]string{}
				fkeys := []string{}
				indexes := []string{}
				mindexes := map[string]string{}
				uindexes := map[string]string{}
				mi := &migrationInput{
					dialect: db.Dialect,
					fName: d,
					fType: ty,
					fTags: &ftags,
					fKeys: &fkeys,
					res: &res,
					indexes: &indexes,
					mindexes: &mindexes,
					uindexes: &uindexes,
				}
				ty = strings.ToLower(ty)
				switch {
				case strings.Contains(ty, "str"):
					handleMigrationString(mi)
					var s string
					var fkey string
					if v, ok := res[d]; ok {
						s = v
						if strings.Contains(v,"UNIQUE") {
							s=strings.ReplaceAll(v,"UNIQUE","")
							uindexes[d]=d
						}
					} else {
						s = "VARCHAR(255)"
					}
					for _, fk := range fkeys {
						r := strings.Index(fk,"REFERENCE")
						fkey = fk[r:]
					}
					if fkey != "" {
						s += " " + fkey
					}
					statement += s
				case strings.Contains(ty, "bool"):
					handleMigrationBool(mi)
					var s string
					var fkey string
					if v, ok := res[d]; ok {
						s = v
						if !strings.Contains(v,"DEFAULT 0") {
							s += " DEFAULT 0"
						}
						if strings.Contains(v,"UNIQUE") {
							s=strings.ReplaceAll(v,"UNIQUE","")
							uindexes[d]=d
						}
					} else {
						s = "INTEGER NOT NULL CHECK (" + d + " IN (0, 1)) DEFAULT 0"
					}
					for _, fk := range fkeys {
						r := strings.Index(fk,"REFERENCE")
						fkey = fk[r:]
					}
					if fkey != "" {
						s += " " + fkey
					}
					statement += s
				case strings.Contains(ty, "int"):
					handleMigrationInt(mi)
					var s string
					var fkey string
					if v, ok := res[d]; ok {
						s = v
						if strings.Contains(v,"UNIQUE") {
							s=strings.ReplaceAll(v,"UNIQUE","")
							uindexes[d]=d
						}
					} else {
						s = "INTEGER"
					}
					for _, fk := range fkeys {
						r := strings.Index(fk,"REFERENCE")
						fkey = fk[r:]
					}
					if fkey != "" {
						s += " " + fkey
					}
					statement += s
				case strings.Contains(ty, "floa"):
					handleMigrationFloat(mi)
					var s string
					var fkey string
					if v, ok := res[d]; ok {
						s = v
						if strings.Contains(v,"UNIQUE") {
							s=strings.ReplaceAll(v,"UNIQUE","")
							uindexes[d]=d
						}
					} else {
						s = "DECIMAL(5,2)"
					}
					for _, fk := range fkeys {
						r := strings.Index(fk,"REFERENCE")
						fkey = fk[r:]
					}
					if fkey != "" {
						s += " " + fkey
					}
					statement += s
				case strings.Contains(ty, "time"):
					handleMigrationTime(mi)
					var s string
					var fkey string
					if v, ok := res[d]; ok {
						s = v
						if strings.Contains(v,"UNIQUE") {
							s=strings.ReplaceAll(v,"UNIQUE","")
							uindexes[d]=d
						}
					} else {
						if strings.Contains(db.Dialect, SQLITE) {
							s = "TEXT"
						} else {
							s = "TIMESTAMP"
						}
					}
					s = strings.ToLower(s)
					if strings.Contains(s, "default") {
						sp := strings.Split(s, " ")
						s = strings.Join(sp[:len(sp)-2], " ")
					}
					if strings.Contains(s, "not null") {
						s = strings.ReplaceAll(s, "not null", "")
					}
					for _, fk := range fkeys {
						r := strings.Index(fk,"REFERENCE")
						fkey = fk[r:]
					}
					if fkey != "" {
						s += " " + fkey
					}
					statement += s
				default:
					logger.Info("case not handled:", ty)
					return
				}

				statIndexes := ""
				if len(indexes) > 0 {
					if len(indexes) > 1 {
						logger.Error(mi.fName,"cannot have more than 1 index")
					} else {
						statIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",to_table_name,d,to_table_name,indexes[0])
					}
				}
				mstatIndexes := ""
				if len(*mi.mindexes) > 0 {
					if len(*mi.mindexes) > 1 {
						logger.Error(mi.fName,"cannot have more than 1 multiple indexes")
					} else {
						if v,ok := (*mi.mindexes)[mi.fName];ok {
							mstatIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",to_table_name,d,to_table_name,d+","+v)
						}
					}
				}
				ustatIndexes := ""
				if len(*mi.uindexes) > 0 {
					if len(*mi.uindexes) > 1 {
						logger.Error(mi.fName,"cannot have more than 1 multiple indexes")
					} else {
						if v,ok := (*mi.uindexes)[mi.fName];ok {
							reste := ","
							if v == "" {reste=v}
							mstatIndexes = fmt.Sprintf("CREATE UNIQUE INDEX idx_%s_%s ON %s (%s)",to_table_name,d,to_table_name,d+reste+v)
						}
					}
				}
				
				if utils.SliceContains([]string{"yes", "Y", "y"}, choice) {
					if len(databases) > 1 && db.Name == "" {
						ddb := input.Input(input.Blue, "> There are more than one database connected, database name:")
						conn := GetConnection(ddb)
						if conn != nil {
							_, err := conn.Exec(statement)
							if logger.CheckError(err) {
								logger.Info(statement)
								return
							}
							if statIndexes != "" {
								_, err := conn.Exec(statIndexes)
								if logger.CheckError(err) {
									logger.Info(statIndexes)
									return
								}
							}
							if mstatIndexes != "" {
								_, err := conn.Exec(mstatIndexes)
								if logger.CheckError(err) {
									logger.Info(mstatIndexes)
									return
								}
							}
							if ustatIndexes != "" {
								_, err := conn.Exec(ustatIndexes)
								if logger.CheckError(err) {
									logger.Info(ustatIndexes)
									return
								}
							}
							if (Debug) {
								logger.Printfs("ylstatement: %s",statement)
								logger.Printfs("ylstatIndexes: %s",statIndexes)
								logger.Printfs("ylmstatIndexes: %s",mstatIndexes)
								logger.Printfs("ylustatIndexes: %s",ustatIndexes)
							}
							logger.Printfs("grDone, '%s' added to '%s', you may want to restart your server", d, to_table_name)
						}
					} else {
						conn := GetConnection(db.Name)
						if conn != nil {
							_, err := conn.Exec(statement)
							if logger.CheckError(err) {
								logger.Info(statement)
								return
							}
							if statIndexes != "" {
								_, err := conn.Exec(statIndexes)
								if logger.CheckError(err) {
									logger.Info(statIndexes)
									return
								}
							}
							if mstatIndexes != "" {
								_, err := conn.Exec(mstatIndexes)
								if logger.CheckError(err) {
									logger.Info(mstatIndexes)
									return
								}
							}
							if ustatIndexes != "" {
								_, err := conn.Exec(ustatIndexes)
								if logger.CheckError(err) {
									logger.Info(ustatIndexes)
									return
								}
							}
							if (Debug) {
								logger.Printfs("ylstatement: %s",statement)
								logger.Printfs("ylstatIndexes: %s",statIndexes)
								logger.Printfs("ylmstatIndexes: %s",mstatIndexes)
								logger.Printfs("ylmstatIndexes: %s",mstatIndexes)
							}
							logger.Printfs("grDone, '%s' added to '%s', you may want to restart your server", d, to_table_name)
						}
					}
				} else {
					logger.Printfs("grNothing changed")
				}
			} else {
				logger.Info("case not handled:", ty, ftypes[d])
			}
		}
	}
}

// handleRename handle sync with db when renaming fields struct
func handleRename(to_table_name string, fields, cols, diff []string, db *DatabaseEntity,ftags map[string][]string) {
	// rename field
	old := []string{}
	new := []string{}
	if len(fields) == len(cols) && len(diff)%2 == 0 && len(diff) > 0 {
		for _, d := range diff {
			if v,ok := ftags[d];ok && v[0] == "-"{
				continue
			}
			if !utils.SliceContains(cols, d) { // d is new
				new = append(new, d)
			} else { // d is old
				old = append(old, d)
			}
		}
	}
	if len(new) > 0 && len(new) == len(old) {
		if len(new) == 1 {
			choice := input.Input(input.Yellow, "⚠️ you renamed '"+old[0]+"' to '"+new[0]+"', execute these changes to db ? (Y/n):")
			if utils.SliceContains([]string{"yes", "Y", "y"}, choice) {
				statement := "ALTER TABLE " + to_table_name + " RENAME COLUMN " + old[0] + " TO " + new[0]
				if len(databases) > 1 && db.Name == "" {
					ddb := input.Input(input.Blue, "> There are more than one database connected, database name:")
					conn := GetConnection(ddb)
					if conn != nil {
						if Debug {
							logger.Info("statement:",statement)
						}
						_, err := conn.Exec(statement)
						if logger.CheckError(err) {
							logger.Info("statement:",statement)
							return
						}
						logger.Printfs("grDone, '%s' has been changed to %s", old[0], new[0])
					}
				} else {
					conn := db.Conn
					if conn != nil {
						if Debug {
							logger.Info("statement:",statement)
						}
						_, err := conn.Exec(statement)
						if logger.CheckError(err) {
							logger.Info("statement:",statement)
							return
						}
						logger.Printfs("grDone, '%s' has been changed to %s", old[0], new[0])
					}
				}
			} else {
				logger.Printfs("grNothing changed")
			}
		} else {
			for _, n := range new {
				for _, o := range old {
					if strings.HasPrefix(n, o) || strings.HasPrefix(o, n) {
						choice := input.Input(input.Yellow, "⚠️ you renamed '"+o+"' to '"+n+"', execute these changes to db ? (Y/n):")
						if utils.SliceContains([]string{"yes", "Y", "y"}, choice) {
							statement := "ALTER TABLE " + to_table_name + " RENAME COLUMN " + o + " TO " + n
							if len(databases) > 1 && db.Name == "" {
								ddb := input.Input(input.Blue, "> There are more than one database connected, database name:")
								conn := GetConnection(ddb)
								if conn != nil {
									if Debug {
										logger.Info("statement:",statement)
									}
									_, err := conn.Exec(statement)
									if logger.CheckError(err) {
										logger.Info("statement:",statement)
										return
									}
								}
							} else {
								if Debug {
									logger.Info("statement:",statement)
								}
								conn := GetConnection(db.Name)
								if conn != nil {
									_, err := conn.Exec(statement)
									if logger.CheckError(err) {
										logger.Info("statement:",statement)
										return
									}
								}
							}
							logger.Printfs("grDone, '%s' has been changed to %s", o, n)
						}
					}
				}
			}
		}
	}
}

func GetConstraints(db *DatabaseEntity, tableName string) map[string][]string {
	res := map[string][]string{}
	switch db.Dialect {
	case SQLITE:
		st := "select sql from sqlite_master where type='table' and name='" + tableName + "';"
		d, err := Query(db.Name, st)
		if logger.CheckError(err) {
			return nil
		}
		sqlStat := d[0]["sql"]
		if _, after, found := strings.Cut(sqlStat.(string), "("); found {
			lines := strings.Split(after[:len(after)-1], ",")
			for _, l := range lines {
				sp := strings.Split(l, " ")
				if len(sp) > 1 && sp[1] != "" {
					col := sp[0]
					tags := sp[1:]
					if col != "" && len(tags) > 1 {
						for _, t := range tags {
							switch t {
							case "PRIMARY", "PRIMARY KEY":
								res[col] = append(res[col], "pkey")
							case "NOT NULL", "NULL":
								res[col] = append(res[col], "notnull")
							case "FOREIGN", "FOREIGN KEY":
								res[col] = append(res[col], "fkey")
							case "CHECK":
								res[col] = append(res[col], "chk")
							case "UNIQUE":
								res[col] = append(res[col], "key")
							default:
								if t == "KEY" && col == "FOREIGN" {
									col := tags[1][1 : len(tags[1])-1]
									res[col] = append(res[col], "fkey")
								}
							}
						}
					}
				}
			}
		}
	case POSTGRES, MYSQL:
		st := "select table_name,constraint_type,constraint_name from INFORMATION_SCHEMA.TABLE_CONSTRAINTS where table_name='" + tableName + "';"
		d, err := Query(db.Name, st)
		if !logger.CheckError(err) {
			for _, dd := range d {
				logger.Success(dd)
				switch {
				case strings.HasPrefix(dd["constraint_name"].(string), "chk_"):
					ln := len("chk_") + len(tableName) + 1
					col := dd["constraint_name"].(string)[ln:]
					res[col] = append(res[col], "chk")
				case strings.HasSuffix(dd["constraint_name"].(string), "_pkey"):
					sp := strings.Split(dd["constraint_name"].(string), "_")
					sp = sp[:len(sp)-1]
					col := strings.Join(sp, "_")
					res[col] = append(res[col], "pkey")
				case strings.HasSuffix(dd["constraint_name"].(string), "_fkey"):
					if constraintName, ok := dd["constraint_name"].(string); ok {
						sp := strings.Split(constraintName, "_")
						table := sp[0]
						if table != tableName {
							for i := 2; true; i++ {
								table = strings.Join(sp[0:i], "_")
								if table == tableName {
									break
								}
							}

						}
						ln := len(table) + 2
						col := constraintName[ln:(len(constraintName) - len("_fkey"))]
						res[col] = append(res[col], "fkey")

					}
				case strings.HasSuffix(dd["constraint_name"].(string), "_key"):
					if constraintName, ok := dd["constraint_name"].(string); ok {
						// users_email_key
						sp := strings.Split(constraintName, "_")
						table := sp[0]
						if table != tableName {
							for i := 2; true; i++ {
								table = strings.Join(sp[0:i], "_")
								if table == tableName {
									break
								}
							}

						}
						ln := len(table) + 2
						col := constraintName[ln : len(constraintName)-len("_key")]
						res[col] = append(res[col], "key")

					}
				default:
				}
			}
		}
	}
	return res
}

func ShutdownDatabases(databasesName ...string) error {
	if len(databasesName) > 0 {
		for i := range databasesName {
			for _, db := range databases {
				if db.Name == databasesName[i] {
					if err := db.Conn.Close(); err != nil {
						return err
					}
				}
			}
		}
	} else {
		for i := range databases {
			if err := databases[i].Conn.Close(); err != nil {
				return err
			}
		}
	}
	return nil
}

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func SnakeCaseToTitle(inputUnderScoreStr string) (camelCase string) {
	//snake_case to camelCase
	isToUpper := false
	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if isToUpper {
				camelCase += strings.ToUpper(string(v))
				isToUpper = false
			} else {
				if v == '_' {
					isToUpper = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return
}

func getTableName[T comparable]() string {
	if v, ok := mModelTablename[*new(T)]; ok {
		return v
	} else {
		return ""
	}
}

func fillStruct[T comparable](struct_to_fill *T, values_to_fill ...any) {
	rs := reflect.ValueOf(struct_to_fill).Elem()
	if len(values_to_fill) != rs.NumField() {
		logger.Error("values to fill and struct fields are not the same length")
		logger.Info("len(values_to_fill)=", len(values_to_fill))
		logger.Info("NumField=", rs.NumField())
		return
	}

	//tt := reflect.TypeOf(struct_to_fill).Elem()
	for i := 0; i < rs.NumField(); i++ {
		field := rs.Field(i)
		valueToSet := reflect.ValueOf(values_to_fill[i])
		//logger.Info(tt.FieldByIndex([]int{i}).Name, "fieldType=", field.Kind(), ",valueType=", reflect.ValueOf(values_to_fill[i]).Kind())
		switch field.Kind() {
		case reflect.ValueOf(values_to_fill[i]).Kind():
			field.Set(valueToSet)
		case reflect.String:
			switch valueToSet.Kind() {
			case reflect.String:
				field.SetString(valueToSet.String())
			case reflect.Struct:
				field.SetString(valueToSet.String())
			default:
				if valueToSet.IsValid() {
					field.Set(valueToSet)
				} else {
					field.SetString("")
				}
			}
		case reflect.Int:
			//logger.Info("INT", tt.Field(i).Name, ":", valueToSet.Interface())
			switch v := valueToSet.Interface().(type) {
			case int64:
				field.SetInt(v)
			case string:
				if v, err := strconv.Atoi(v); err == nil {
					field.SetInt(int64(v))
				}
			case int:
				field.SetInt(int64(v))
			}
		case reflect.Int64:
			switch v := valueToSet.Interface().(type) {
			case int64:
				field.SetInt(v)
			case string:
				if v, err := strconv.Atoi(v); err == nil {
					field.SetInt(int64(v))
				}
			case []byte:
				if v, err := strconv.Atoi(string(v)); err != nil {
					field.SetInt(int64(v))
				}
			case int:
				field.SetInt(int64(v))
			}
		case reflect.Bool:
			//logger.Info("Bool", tt.Field(i).Name, ":", valueToSet.Interface(), fmt.Sprintf("%T", valueToSet.Interface()))
			switch valueToSet.Kind() {
			case reflect.Int:
				if values_to_fill[i] == 1 {
					field.SetBool(true)
				}
			case reflect.Int64:
				if values_to_fill[i] == int64(1) {
					field.SetBool(true)
				}
			case reflect.Uint64:
				if values_to_fill[i] == uint64(1) {
					field.SetBool(true)
				}
			case reflect.String:
				if values_to_fill[i] == "1" {
					field.SetBool(true)
				}
			}
		case reflect.Uint:
			switch v := valueToSet.Interface().(type) {
			case uint:
				field.SetUint(uint64(v))
			case uint64:
				field.SetUint(v)
			case int64:
				field.SetUint(uint64(v))
			case int:
				field.SetUint(uint64(v))
			}
		case reflect.Uint64:
			switch v := valueToSet.Interface().(type) {
			case uint:
				field.SetUint(uint64(v))
			case uint64:
				field.SetUint(v)
			case int64:
				field.SetUint(uint64(v))
			case int:
				field.SetUint(uint64(v))
			}
		case reflect.Float64:
			if v, ok := valueToSet.Interface().(float64); ok {
				field.SetFloat(v)
			}
		case reflect.SliceOf(reflect.TypeOf("")).Kind():
			field.SetString(strings.Join(valueToSet.Interface().([]string), ","))
		case reflect.Struct:
			if valueToSet.IsValid() {
				switch v := valueToSet.Interface().(type) {
				case string:
					if utils.StringContains(v, ":", "-") {
						l := len("2006-01-02T15:04")
						if strings.Contains(v[:l], "T") {
							if len(v) >= l {
								t, err := time.Parse("2006-01-02T15:04", v[:l])
								if !logger.CheckError(err) {
									field.Set(reflect.ValueOf(t))
								}
							}
						} else if len(v) >= len("2006-01-02 15:04:05") {
							t, err := time.Parse("2006-01-02 15:04:05", v[:len("2006-01-02 15:04:05")])
							if !logger.CheckError(err) {
								field.Set(reflect.ValueOf(t))
							}
						} else {
							logger.Info("doesn't match any case", v)
						}
					}
				case time.Time:
					logger.Info(v)
					field.Set(reflect.ValueOf(v))
				}
			}
		default:
			field.Set(reflect.ValueOf(valueToSet.Interface()))
			logger.Error("type not handled for fieldType", field.Kind(), "value=", valueToSet.Interface())
		}
	}
}

func fillStructColumns[T comparable](struct_to_fill *T, columns_to_fill string, values_to_fill ...any) {
	cols := strings.Split(columns_to_fill, ",")
	if len(values_to_fill) != len(cols) && columns_to_fill != "*" && columns_to_fill != "" {
		logger.Error("len(values_to_fill) not the same of len(struct fields)")
		return
	}

	// pointer struct
	ps := reflect.ValueOf(struct_to_fill)
	// struct
	s := ps.Elem()
	//typeOfS := s.Type()

	if s.Kind() == reflect.Struct {
		for i, col := range cols {
			valueToSet := reflect.ValueOf(&values_to_fill[i]).Elem()
			var fieldToUpdate *reflect.Value
			if f := s.FieldByName(SnakeCaseToTitle(col)); f.IsValid() && f.CanSet() {
				fieldToUpdate = &f
			} else if f := s.FieldByName(col); f.IsValid() && f.CanSet() {
				// usually here
				fieldToUpdate = &f
			} else if f := s.FieldByName(ToSnakeCase(col)); f.IsValid() && f.CanSet() {
				fieldToUpdate = &f
			}

			if fieldToUpdate != nil {
				//logger.Info(typeOfS.FieldByIndex([]int{i}).Name,"fieldType=", fieldToUpdate.Kind(),",valueType=",reflect.ValueOf(values_to_fill[i]).Kind())
				switch fieldToUpdate.Kind() {
				case reflect.ValueOf(values_to_fill[i]).Kind():
					fieldToUpdate.Set(reflect.ValueOf(valueToSet.Interface()))
				case reflect.String:
					switch v := valueToSet.Interface().(type) {
					case string:
						fieldToUpdate.SetString(v)
					case time.Time:
						fieldToUpdate.SetString(v.String())
					case bool:
						fieldToUpdate.SetString(strconv.FormatBool(v))
					default:
						if !valueToSet.IsZero() {
							fieldToUpdate.Set(reflect.ValueOf(v))
						} else {
							fieldToUpdate.SetString("")
						}
					}
				case reflect.Int:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.String:
						i, err := strconv.Atoi(values_to_fill[i].(string))
						if err == nil {
							fieldToUpdate.SetInt(int64(i))
						}
					case reflect.Int64:
						fieldToUpdate.SetInt(values_to_fill[i].(int64))
					case reflect.Uint64:
						fieldToUpdate.SetInt(int64(values_to_fill[i].(uint64)))
					default:
						fmt.Printf("%T\n", values_to_fill[i])
						logger.Error("Int doeesn't match anything,type should be", fieldToUpdate.Kind(), "but got", values_to_fill[i])
					}
				case reflect.Int64:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.String:
						i, err := strconv.Atoi(values_to_fill[i].(string))
						if err == nil {
							fieldToUpdate.SetInt(int64(i))
						}
					case reflect.Int:
						fieldToUpdate.SetInt(int64(values_to_fill[i].(int)))
					case reflect.Uint64:
						fieldToUpdate.SetInt(int64(values_to_fill[i].(uint64)))
					default:
						logger.Error("Int64 doeesn't match anything")
					}
				case reflect.Uint:
					switch v := valueToSet.Interface().(type) {
					case uint:
						fieldToUpdate.SetUint(uint64(v))
					case uint64:
						fieldToUpdate.SetUint(v)
					case int64:
						fieldToUpdate.SetUint(uint64(v))
					case int:
						fieldToUpdate.SetUint(uint64(v))
					default:
						logger.Error("type of valueToSet:")
						fmt.Printf("%T\n", valueToSet.Interface())
					}
				case reflect.Uint64:
					switch v := valueToSet.Interface().(type) {
					case uint:
						fieldToUpdate.SetUint(uint64(v))
					case uint64:
						fieldToUpdate.SetUint(v)
					case int64:
						fieldToUpdate.SetUint(uint64(v))
					case int:
						fieldToUpdate.SetUint(uint64(v))
					default:
						logger.Error("type of valueToSet:")
						fmt.Printf("%T\n", valueToSet.Interface())
					}
				case reflect.Float64:
					if v, ok := valueToSet.Interface().(float64); ok {
						fieldToUpdate.SetFloat(v)
					}
				case reflect.Struct:
					if valueToSet.IsValid() {
						switch v := valueToSet.Interface().(type) {
						case string:
							l := len("2006-01-02T15:04")
							if strings.Contains(v[:l], "T") {
								if len(v) >= l {
									t, err := time.Parse("2006-01-02T15:04", v[:l])
									if !logger.CheckError(err) {
										fieldToUpdate.Set(reflect.ValueOf(t))
									}
								}
							} else if len(v) >= len("2006-01-02 15:04:05") {
								t, err := time.Parse("2006-01-02 15:04:05", v[:len("2006-01-02 15:04:05")])
								if !logger.CheckError(err) {
									fieldToUpdate.Set(reflect.ValueOf(t))
								}
							} else {
								logger.Info("v string doesn't match any case,v=", v)
							}
						case time.Time:
							fieldToUpdate.Set(reflect.ValueOf(v))
						default:
							logger.Error("field struct with value", v, "not handled")
						}
					}
				case reflect.Bool:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.Int:
						if values_to_fill[i] == 1 {
							fieldToUpdate.SetBool(true)
						}
					case reflect.Int64:
						if values_to_fill[i] == int64(1) {
							fieldToUpdate.SetBool(true)
						}
					case reflect.Uint64:
						if values_to_fill[i] == uint64(1) {
							fieldToUpdate.SetBool(true)
						}
					case reflect.String:
						if values_to_fill[i] == "1" {
							fieldToUpdate.SetBool(true)
						}
					default:
						logger.Error("not handled BOOL")
					}
				default:
					switch v := valueToSet.Interface().(type) {
					case []byte:
						fieldToUpdate.SetString(string(v))
					default:
						logger.Error("case not handled , unable to fill struct,field kind:", fieldToUpdate.Kind(), ",value to fill:", values_to_fill[i])
					}
				}
			}
		}
	} else {
		logger.Error("struct_to_fill is not struct :)")
	}
}

// getStructInfos very useful to access all struct fields data using reflect package
func getStructInfos[T comparable](strct *T) (fields []string, fValues map[string]any, fTypes map[string]string, fTags map[string][]string) {
	fields = []string{}
	fValues = map[string]any{}
	fTypes = map[string]string{}
	fTags = map[string][]string{}

	s := reflect.ValueOf(strct).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fname := typeOfT.Field(i).Name
		fname = ToSnakeCase(fname)
		fvalue := f.Interface()
		ftype := f.Type().Name()

		fields = append(fields, fname)
		fTypes[fname] = ftype
		fValues[fname] = fvalue
		if ftag, ok := typeOfT.Field(i).Tag.Lookup("orm"); ok {
			tags := strings.Split(ftag, ";")
			fTags[fname] = tags
		}
	}
	return fields, fValues, fTypes, fTags
}

func adaptPlaceholdersToDialect(query *string, dialect string) *string {
	if strings.Contains(*query, "?") && (dialect == POSTGRES || dialect == SQLITE) {
		split := strings.Split(*query, "?")
		counter := 0
		for i := range split {
			if i < len(split)-1 {
				counter++
				split[i] = split[i] + "$" + strconv.Itoa(counter)
			}
		}
		*query = strings.Join(split, "")
	}
	return query
}

func GenerateUUID() (string, error) {
	var uuid [16]byte
	_, err := io.ReadFull(rand.Reader, uuid[:])
	if err != nil {
		return "", err
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:]), nil
}
func encodeHex(dst []byte, uuid [16]byte) {
	hex.Encode(dst, uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}
