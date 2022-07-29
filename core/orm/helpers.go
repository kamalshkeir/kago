package orm

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
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
	fields, _, ftypes, ftags := getStructInfos(new(T))
	// get columns from db
	colsNameType := GetAllColumns(to_table_name,db.Name)
	names := []string{}
	var pkTable string
	for k := range colsNameType {
		names = append(names, k)
	}
	tFound := false
	for _,t := range db.Tables {
		if t.Name == to_table_name {
			tFound=true
		}
	}
	fkkeys := []string{}
	for col,tgs := range ftags {
		for _,t := range tgs {
			if t == "autoinc" {
				pkTable=col
			}
		}	
	}

	// check if not the same list as struct fields
	diff := utils.Difference(fields,names)
	if len(names) > len(fields) {
		for _,d := range diff {
			fmt.Println(" ")
			logger.Printfs("/!\\ found extra column '%s' in the database table '%s'",d,to_table_name)
			statement := "ALTER TABLE "+to_table_name+" DROP COLUMN "+d
			choice := input.Input(input.Yellow,"> do you want to remove it ? (Y/n):")
			if utils.SliceContains([]string{"yes","Y","y"},choice) {
				if len(databases) > 1 {
					ddb := input.Input(input.Blue,"> There are more than one database connected, database name:")
					conn := GetConnection(ddb)
					if conn != nil {
						_,err := conn.Exec(statement)
						if logger.CheckError(err) {
							return 
						}
					}
				} else {
					conn := GetConnection()
					if conn != nil {
						_,err := conn.Exec(statement)
						if logger.CheckError(err) {
							return 
						}
						fmt.Printf(logger.Green,"Done, you may want to restart your server")
					}
				}
			}
		}
	} else if len(names) < len(fields) {
		for _,d := range diff {
			fmt.Println(" ")
			logger.Printfs("/!\\ column '%s' is missing from the database table '%s'",d,to_table_name)
			choice,err := input.String(input.Yellow,"> do you want to add it ? (Y/n):")
			logger.CheckError(err)
			statement := "ALTER TABLE "+to_table_name+" ADD "+d+" "
			if ty,ok := ftypes[d];ok {
				ty = strings.ToLower(ty)
				switch  {
				case strings.Contains(ty,"str"):
					res := map[string]string{}
					fkeys := []string{}
					handleMigrationString(db.Dialect,d,ty,&ftags,&fkeys,&res)
					var s string
					var fkey string
					if v,ok := res[d];ok {
						s=v
					} else {
						s="VARCHAR(255)"
					}
					for _,fk := range fkeys {
						sp := strings.Split(fk," ")
						fkey = strings.Join(sp[2:]," ") 
					}				
					if fkey != "" {
						s += " "+fkey
						fkkeys = append(fkkeys, d)
					}
					statement += s
				case strings.Contains(ty,"bool"):
					res := map[string]string{}
					fkeys := []string{}
					handleMigrationBool(db.Dialect,d,ty,&ftags,&fkeys,&res)
					var s string
					var fkey string
					if v,ok := res[d];ok {
						s=v
					} else {
						s="INTEGER NOT NULL CHECK (" + d + " IN (0, 1)) DEFAULT 0"
					}
					for _,fk := range fkeys {
						sp := strings.Split(fk," ")
						fkey = strings.Join(sp[2:]," ") 
					}
					if fkey != "" {
						s += " "+fkey
						fkkeys = append(fkkeys, d)
					}
					statement += s
				case strings.Contains(ty,"int"):
					res := map[string]string{}
					fkeys := []string{}
					handleMigrationInt(db.Dialect,d,ty,&ftags,&fkeys,&res)
					var s string
					var fkey string
					if v,ok := res[d];ok {
						s=v
					} else {
						s="INTEGER"
					}
					for _,fk := range fkeys {
						sp := strings.Split(fk," ")
						fkey = strings.Join(sp[2:]," ") 
					}
					if fkey != "" {
						s += " "+fkey
						fkkeys = append(fkkeys, d)
					}
					statement += s
				case strings.Contains(ty,"floa"):
					res := map[string]string{}
					fkeys := []string{}
					handleMigrationFloat(db.Dialect,d,ty,&ftags,&fkeys,&res)
					var s string
					var fkey string
					if v,ok := res[d];ok {
						s=v
					} else {
						s="DECIMAL(5,2)"
					}
					for _,fk := range fkeys {
						sp := strings.Split(fk," ")
						fkey = strings.Join(sp[2:]," ") 
					}
					if fkey != "" {
						s += " "+fkey
						fkkeys = append(fkkeys, d)
					}
					statement += s
				case strings.Contains(ty,"time"):
					res := map[string]string{}
					fkeys := []string{}
					handleMigrationTime(db.Dialect,d,ty,&ftags,&fkeys,&res)
					var s string
					var fkey string
					if v,ok := res[d];ok {
						s=v
					} else {
						if strings.Contains(db.Dialect,"sqlite") {
							s="TEXT"
						}  else {
							s="TIMESTAMP"
						}
					}
					s = strings.ToLower(s)
					if strings.Contains(s,"default") {
						sp := strings.Split(s," ")
						s = strings.Join(sp[:len(sp)-2]," ")
					}
					if strings.Contains(s,"not null") {
						s = strings.ReplaceAll(s,"not null","")
					}
					for _,fk := range fkeys {
						sp := strings.Split(fk," ")
						fkey = strings.Join(sp[2:]," ") 
					}
					if fkey != "" {
						s += " "+fkey
						fkkeys = append(fkkeys, d)
					}
					statement += s
				default:
					logger.Info("case not handled:",ty)
					return
				}

				if utils.SliceContains([]string{"yes","Y","y"},choice) {
					if len(databases) > 1 {
						ddb := input.Input(input.Blue,"> There are more than one database connected, database name:")
						conn := GetConnection(ddb)
						if conn != nil {
							_,err := conn.Exec(statement)
							if logger.CheckError(err) {
								return 
							}
							logger.Printfs("Done, you may want to restart your server")
						}
					} else {
						conn := GetConnection()
						if conn != nil {
							_,err := conn.Exec(statement)
							if logger.CheckError(err) {
								logger.Info(statement)
								return 
							}
							logger.Printfs("Done, you may want to restart your server")
						}
					}
				} else {
					logger.Info("Nothing changed")
				}
			} else {
				logger.Info("case not handled:",ty,ftypes[d])
			}
		}
	} 
	
	
	// set maps
	if !tFound {
		db.Tables=append(db.Tables, TableEntity{
			Name: to_table_name,
			Columns: names,
			ModelTypes: ftypes,
			Types: colsNameType,
			Tags: ftags,
			Fkeys: fkkeys,
			Pk: pkTable,
		})
	}
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
	}
	return ""
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
		//logger.Info(tt.FieldByIndex([]int{i}).Name,"fieldType=", field.Kind(),",valueType=",reflect.ValueOf(values_to_fill[i]).Kind())
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
			default:
				logger.Error("not handled:", v)
				field.Set(reflect.ValueOf(valueToSet.Interface()))
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
			default:
				field.Set(reflect.ValueOf(valueToSet.Interface()))
				logger.Error(v, "not handled")
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
			default:
				logger.Error("not handled BOOL")
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
			default:
				logger.Error("type of valueToSet:")
				fmt.Printf("%T\n", valueToSet.Interface())
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
			default:
				logger.Error("type of valueToSet:")
				fmt.Printf("%T\n", valueToSet.Interface())
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
					if utils.StringContains(v,":","-") {
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
				default:
					logger.Error("field struct with value", v, "not handled")
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
	if strings.Contains(*query, "?") && (dialect == "postgres" || dialect == "sqlite") {
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
