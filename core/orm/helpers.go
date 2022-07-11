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

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/logger"
)


type dbCache struct {
	table string
	selected string
	limit int
	page int
	orderBys string
	whereQuery string
	query      string
	offset     string
	statement  string
	args string
}

// LinkModel link a struct model to a  db_table_name
// Usage: korm.LinkModel[User]("users")
func LinkModel[T comparable](to_table_name string, dbNames ...string) {
	var dbName string
	if len(dbNames) == 0 {
		dbName=settings.GlobalConfig.DbName
	} else {
		dbName=dbNames[0]
	}

	fields, _,fieldsTags := getFieldTypesAndTags[T]()


	
	var dialect string
	conn := GetConnection(dbNames...)
	if conn != nil {
		if dialct,ok := mDbNameDialect[dbName];ok {
			dialect=dialct
		} else {
			logger.Error("unable to find dialect")
		}
	}
	
	tables := GetAllTables(dbName)	
	found := false
	for _,t := range tables {
		if t == to_table_name {
			found = true
		}
	}	
	if !found {
		logger.Warn("table",to_table_name,"not found in database",dbName)
		return
	}

	// table found in db
	names := []string{}
	mModelTablename[*new(T)]=to_table_name

	// get columns from db
	colsNameType := GetAllColumns(to_table_name)
	for k := range colsNameType {
		names = append(names, k)
	}
	// check if not the same list as struct fields
	if !utils.IsSameSlice(names,fields) {
		logger.Error("your model doesn't match with the table in the database, all struct fields names to snakeCase should match column name from the table")
		logger.Info("struct fields:",fields)
		logger.Info("db columns:",names)
		return
	}

	if v,ok := mModelDatabase[*new(T)];ok {
		v.tables = append(v.tables, table{
			name: to_table_name,
			columnsType: colsNameType,
			columns: names,
			columnsTags: fieldsTags,
		})
		v.conn=conn
		v.dialect=dialect
		v.name=dbName
	} else {
		mModelDatabase[*new(T)]=database{
			name: dbName,
			tables: []table{
				{
					name: to_table_name,
					columnsType: colsNameType,
					columns: names,
				},
			},
			conn: conn,
			dialect: dialect,
		}
	}
}

func getTableName[T comparable]() string {
	if v,ok := mModelTablename[*new(T)];ok {
		return v
	}
	return ""
}

func fillStruct[T comparable](struct_to_fill *T,values_to_fill ...any) {
	rs := reflect.ValueOf(struct_to_fill).Elem()
	if len(values_to_fill) != rs.NumField() {
		logger.Error("values to fill and struct fields are not the same length")
		logger.Info("len(values_to_fill)=",len(values_to_fill))
		logger.Info("NumField=",rs.NumField())
		return
	}
	for i := 0; i < rs.NumField() ;i++  {
		field := rs.Field(i) 
		valueToSet := reflect.ValueOf(&values_to_fill[i]).Elem()
		//logger.Info(tt.FieldByIndex(append([]int{},i)).Name,"fieldType=", field.Kind(),",valueType=",reflect.ValueOf(values_to_fill[i]).Kind()) 
		switch field.Kind() {
			case reflect.String:
				switch v := valueToSet.Interface().(type) {
				case string:
					field.SetString(v)
				case time.Time:
					field.SetString(v.String())
				
				}
			case reflect.Int:
				switch v := valueToSet.Interface().(type) {
				case int64:
					field.SetInt(v)
				case string:
					if v,err := strconv.Atoi(v);err == nil {
						field.SetInt(int64(v))
					}
				case int:
					field.SetInt(int64(v))
				default:
					logger.Info("field kind:",field.Kind())
					logger.Error("not handled:",v)
				}
			case reflect.Int64:
				switch v := valueToSet.Interface().(type) {
				case int64:
					field.SetInt(v)
				case string:
					if v,err := strconv.Atoi(v);err == nil {
						field.SetInt(int64(v))
					}
				case []byte:
					if v,err := strconv.Atoi(string(v));err != nil {
						field.SetInt(int64(v))
					}
				case int:
					field.SetInt(int64(v))
				default:
					logger.Error(v,"not handled")
				}
			case reflect.Bool:
				if v,ok := valueToSet.Interface().(bool);!ok {
					switch v := valueToSet.Interface().(type) {
					case string:
						if v == "0" {field.SetBool(false)}
						if v == "1" {field.SetBool(true)}
					case int32,int64,int,uint,uint64:
						if v == 0 {field.SetBool(false)}
						if v == 1 {field.SetBool(true)}
					default:
						logger.Error(v,"not handled")
					}
				} else {
					field.SetBool(v)
				}
			case reflect.Uint:
				if v,ok := valueToSet.Interface().(uint);ok {
					field.SetUint(uint64(v))
				}
			case reflect.Uint64:
				if v,ok := valueToSet.Interface().(uint64);ok {
					field.SetUint(v)
				}
			case reflect.Float64:
				if v,ok := valueToSet.Interface().(float64);ok {
					field.SetFloat(v)
				}			
			case reflect.SliceOf(reflect.TypeOf("")).Kind():
				field.SetString(strings.Join(valueToSet.Interface().([]string),","))
			case reflect.Struct:
				if v,ok := valueToSet.Interface().(string);ok {
					l := len("2006-01-02T15:04")
					if strings.Contains(v[:l],"T") {
						if len(v) >= l {
							t,err := time.Parse("2006-01-02T15:04",v[:l])
							if !logger.CheckError(err) {
								field.Set(reflect.ValueOf(t))
							}
						} 	
					} else if len(v) >= len("2006-01-02 15:04:05"){
						t,err := time.Parse("2006-01-02 15:04:05",v[:len("2006-01-02 15:04:05")])
						if !logger.CheckError(err) {
							field.Set(reflect.ValueOf(t))
						}
					} else {
						logger.Info("doesn't match any case",v)
					}
				} else {
					logger.Error("field struct not string time.Time",v)
				}
				
			default:
				field.Set(reflect.ValueOf(valueToSet.Interface()))
				logger.Error("type not handled for fieldType",field.Kind(),"value=",valueToSet.Interface())
		}
	}
}

func fillStructColumns[T comparable](struct_to_fill *T,columns_to_fill string,values_to_fill ...any) {
	cols := strings.Split(columns_to_fill,",")
	if len(values_to_fill) != len(cols) && columns_to_fill != "*" && columns_to_fill != ""{
		logger.Error("len(values_to_fill) not the same of len(struct fields)")
		return
	}

	// pointer struct
	ps := reflect.ValueOf(struct_to_fill)
	// struct
	s := ps.Elem()
	//typeOfS := s.Type()

	if s.Kind() == reflect.Struct {
		for i,col := range cols {
			valueToSet := reflect.ValueOf(&values_to_fill[i]).Elem()

			var fieldToUpdate reflect.Value
			if f := s.FieldByName(SnakeCaseToCamelCase(col)); f.IsValid() && f.CanSet() {
				fieldToUpdate = f
			} else if  f := s.FieldByName(col); f.IsValid() &&  f.CanSet(){
				// usually here
				fieldToUpdate=f
			} else if f := s.FieldByName(ToSnakeCase(col)); f.IsValid() && f.CanSet(){
				fieldToUpdate=f
			}

			if &fieldToUpdate != nil {
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
						logger.Error("String doeesn't match anything")
					}
				case reflect.Int:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.String:
						i,err := strconv.Atoi(values_to_fill[i].(string))
						if err == nil {
							fieldToUpdate.SetInt(int64(i))
						}
					case reflect.Int64:
						fieldToUpdate.SetInt(values_to_fill[i].(int64))
					case reflect.Uint64:
						fieldToUpdate.SetInt(int64(values_to_fill[i].(uint64)))
					default:
						fmt.Printf("%T\n",values_to_fill[i])
						logger.Error("Int doeesn't match anything,type should be",fieldToUpdate.Kind(),"but got",values_to_fill[i])
					}
				case reflect.Int64:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.String:
						i,err := strconv.Atoi(values_to_fill[i].(string))
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
				case reflect.Struct:
					if v,ok := valueToSet.Interface().(string);ok {
						if strings.Contains(v,"T") && len(v) == len("2006-01-02T15:04") {
							t,err := time.Parse("2006-01-02T15:04",v)
							if !logger.CheckError(err) {
								fieldToUpdate.Set(reflect.ValueOf(t))
							}
						} else if len(v) == len("2006-01-02 15:04:05"){
							t,err := time.Parse("2006-01-02 15:04:05",v)
							if !logger.CheckError(err) {
								fieldToUpdate.Set(reflect.ValueOf(t))
							}
						} else {
							logger.Info("timestamp doesn't match any case")
						}
					}
				case reflect.Bool:
					switch reflect.ValueOf(values_to_fill[i]).Kind() {
					case reflect.Int:
						if values_to_fill[i] == 1 {fieldToUpdate.SetBool(true)}
					case reflect.Int64:
						if values_to_fill[i] == int64(1) {fieldToUpdate.SetBool(true)}
					case reflect.Uint64:
						if values_to_fill[i] == uint64(1) {fieldToUpdate.SetBool(true)}
					case reflect.String:
						if values_to_fill[i] == "1" {fieldToUpdate.SetBool(true)}
					default:
						logger.Error("not handled BOOL")
					}	
				default:
					switch v := valueToSet.Interface().(type) {
					case []byte:
						fieldToUpdate.SetString(string(v))
					default:
						logger.Error("case not handled , unable to fill struct,field kind:", fieldToUpdate.Kind(),",value to fill:",values_to_fill[i])
					}
				}
			}
		}
	} else {
		logger.Error("struct_to_fill is not struct :)")
	}
}

// getStructInfos very useful to access all struct fields data using reflect package
func getStructInfos[T comparable](strct *T) (fnames []string,fvalues []any,ftypes []reflect.Type,ftags []string) {
	s := reflect.ValueOf(strct).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		if ftag,ok := typeOfT.Field(i).Tag.Lookup("korm");ok {
			ftags = append(ftags, ftag)
		} else {
			ftags = append(ftags, "")
		}
		fname := typeOfT.Field(i).Name
		ftype := f.Type()
		fvalue := f.Interface()
		fnames = append(fnames, fname)
		ftypes = append(ftypes, ftype)
		fvalues = append(fvalues, fvalue)
	}
	return fnames,fvalues,ftypes,ftags
}

// getStructInfos very useful to access all struct fields data using reflect package
func getFieldTypesAndTags[T comparable]() (fields []string,fieldType map[string]string,fieldTags map[string][]string) {
	fields= []string{}
	fieldType = map[string]string{}
	fieldTags = map[string][]string{}
	s := reflect.ValueOf(new(T)).Elem()
	typeOfT := s.Type()
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fname := typeOfT.Field(i).Name
		fname = ToSnakeCase(fname)
		ftype := f.Type().Name()

		fields = append(fields, fname)
		fieldType[fname]=ftype
		if ftag,ok := typeOfT.Field(i).Tag.Lookup("korm");ok {
			tags := strings.Split(ftag,";")
			fieldTags[fname]=tags
		} 
	}
	return fields,fieldType,fieldTags
}


func adaptPlaceholdersToDialect(query *string,dialect string) *string {
	if  strings.Contains(*query,"?") && (dialect == "postgres" || dialect == "sqlite") {
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

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
    snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
    snake  = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
    return strings.ToLower(snake)
}

func SnakeCaseToCamelCase(inputUnderScoreStr string) (camelCase string) {
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

func ShutdownDatabases(databasesName ...string) error {
	if len(databasesName) > 0 {
		for i := range databasesName {
			for _,db := range databases {
				if db.name == databasesName[i] {
					if err := db.conn.Close();err != nil {
						return err
					}
				}
			}
		}
	} else {
		for i := range databases {
			if err := databases[i].conn.Close();err != nil {
				return err
			}
		}
	}
	return nil
}

