package orm

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/kamalshkeir/kago/core/admin/models"
	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils/logger"
)

func Migrate() error {
	err := AutoMigrate[models.User]("users",settings.Config.Db.Name)
	if logger.CheckError(err) {
		return err
	}
	return nil
}

func autoMigrate[T comparable](db *DatabaseEntity, tableName string) error {
	dialect := db.Dialect
	s := reflect.ValueOf(new(T)).Elem()
	typeOfT := s.Type()
	mFieldName_Type := map[string]string{}
	mFieldName_Tags := map[string][]string{}
	cols := []string{}

	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fname := typeOfT.Field(i).Name
		fname = ToSnakeCase(fname)
		ftype := f.Type()
		cols = append(cols,fname)
		mFieldName_Type[fname] = ftype.Name()
		if ftag, ok := typeOfT.Field(i).Tag.Lookup("orm"); ok {
			tags := strings.Split(ftag, ";")
			for i := range tags {
				tags[i]=strings.TrimSpace(tags[i])
			}
			mFieldName_Tags[fname] =  tags
		}
	}
	res := map[string]string{}
	fkeys := []string{}
	indexes := []string{}
	mindexes := map[string]string{}
	uindexes := map[string]string{}
	var mi *migrationInput
	for _, fName := range cols {
		if ty, ok := mFieldName_Type[fName]; ok {
			mi = &migrationInput{
				dialect: dialect,
				fName: fName,
				fType: ty,
				fTags: &mFieldName_Tags,
				fKeys: &fkeys,
				res: &res,
				indexes: &indexes,
				mindexes: &mindexes,
				uindexes: &uindexes,
			}
			switch ty {
			case "int", "uint", "int64", "uint64", "int32", "uint32":
				handleMigrationInt(mi)
			case "bool":
				handleMigrationBool(mi)
			case "string":
				handleMigrationString(mi)
			case "float64", "float32":
				handleMigrationFloat(mi)
			case "Time":
				handleMigrationTime(mi)
			default:
				logger.Error(fName, "of type", ty, "not handled")
			}
		}
	}
	statement := prepareCreateStatement(tableName, res, fkeys, cols,db,mFieldName_Tags)
	tbFound := false
	for _,t := range db.Tables {
		if t.Name == tableName {
			tbFound=true
			if len(t.Columns) == 0 {t.Columns=cols}
			if len(t.Tags) == 0 {t.Tags=mFieldName_Tags}
			if len(t.ModelTypes) == 0 {t.Types=mFieldName_Type}
		}
	}
	if !tbFound {
		db.Tables = append(db.Tables, TableEntity{
			Name: tableName,
			Columns: cols,
			Tags:mFieldName_Tags,
			ModelTypes: mFieldName_Type,
		})
	}
	if Debug {
		fmt.Printf(logger.Blue,"statement: "+ statement)
	}
	
	c, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()
	ress, err := db.Conn.ExecContext(c, statement)
	if err != nil {
		return err
	}
	_, err = ress.RowsAffected()
	if err != nil {
		return err
	}
	if !strings.HasSuffix(tableName,"_temp") {
		statIndexes := ""
		if len(indexes) > 0 {
			if len(indexes) > 1 {
				logger.Error(mi.fName,"cannot have more than 1 index")
			} else {
				statIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",tableName,indexes[0],tableName,indexes[0])
			}
		}
		mstatIndexes := ""
		if len(*mi.mindexes) > 0 {
			if len(*mi.mindexes) > 1 {
				logger.Error(mi.fName,"cannot have more than 1 multiple indexes")
			} else {
				for k,v := range *mi.mindexes {
					mstatIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)",tableName,k,tableName,k+","+v)
				}
			}
		}
		ustatIndexes := ""
		if len(*mi.uindexes) > 0 {
			if len(*mi.uindexes) > 1 {
				logger.Error(mi.fName,"cannot have more than 1 multiple unique indexes")
			} else {
				for k,v := range *mi.uindexes {
					reste := ","
					if v == "" {reste=v}
					ustatIndexes = fmt.Sprintf("CREATE UNIQUE INDEX idx_%s_%s ON %s (%s)",tableName,k,tableName,k+reste+v)
				}
			}
		}
		if statIndexes != "" {
			if Debug {
				logger.Printfs(statIndexes)
			}
			_, err := db.Conn.Exec(statIndexes)
			if logger.CheckError(err) {
				logger.Printfs("indexes: %s",statIndexes)
				return err
			}
		}
		if mstatIndexes != "" {
			if Debug {
				logger.Printfs("mindexes: %s",mstatIndexes)
			}
			_, err := db.Conn.Exec(mstatIndexes)
			if logger.CheckError(err) {
				logger.Printfs("mindexes: %s",mstatIndexes)
				return err
			}
		}
		if ustatIndexes != "" {
			if Debug {
				logger.Printfs("uindexes: %s",ustatIndexes)
			}
			_, err := db.Conn.Exec(ustatIndexes)
			if logger.CheckError(err) {
				logger.Printfs("uindexes: %s",ustatIndexes)
				return err
			}
		}
	}
	
	logger.Printfs("gr%s migrated successfully, restart the server",tableName)
	return nil
}

func AutoMigrate[T comparable](tableName string, dbName ...string) error {
	if _,ok := mModelTablename[*new(T)];!ok {
		mModelTablename[*new(T)]=tableName
	}
	var db *DatabaseEntity
	var err error
	dbname := ""
	if len(dbName) > 0 {
		dbname = dbName[0]
		db,err = GetDatabase(dbname)
		if err != nil || db == nil {
			return errors.New("database not found")
		}
	} else {
		dbname = settings.Config.Db.Name
		db,err = GetDatabase(dbname)
		if err != nil || db == nil {
			return errors.New("database not found")
		}
	}
	
	tbFoundDB := false
	tables := GetAllTables(dbname)
	for _, t := range tables {
		if t == tableName {
			tbFoundDB=true
		}
	}
	
	tbFoundLocal := false
	if len(db.Tables) == 0 {
		if tbFoundDB {
			// found db not local
			linkModel[T](tableName,db)
			return nil
		} else {
			// not db and not local
			err := autoMigrate[T](db,tableName)
			if logger.CheckError(err) {
				return err
			}
			return nil
		}
	} else {
		// db have tables
		for _,t := range db.Tables {
			if t.Name == tableName {
				tbFoundLocal=true
			}
		}
	} 	
	if !tbFoundLocal {
		if tbFoundDB {
			linkModel[T](tableName,db)
			return nil
		} else {
			err := autoMigrate[T](db,tableName)
			if logger.CheckError(err) {
				return err
			}
		}
	} 

	return nil
}

type migrationInput struct {
	dialect string
	fName string
	fType string
	fTags *map[string][]string
	fKeys *[]string
	res *map[string]string
	indexes *[]string
	mindexes *map[string]string
	uindexes *map[string]string
}

func handleMigrationInt(mi *migrationInput) {
	primary,index, autoinc, notnull, defaultt, check,unique := "", "", "", "", "", "",""
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName]=""
		return
	}
	for _, tag := range tags {
		switch tag {
		case "unique":
			unique = " UNIQUE"
		case "pk":
			primary = " PRIMARY KEY"
		case "autoinc":
			switch mi.dialect {
			case SQLITE, "":
				autoinc = "INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT"
			case POSTGRES:
				autoinc = "SERIAL NOT NULL PRIMARY KEY"
			case MYSQL:
				autoinc = "INTEGER NOT NULL PRIMARY KEY AUTO_INCREMENT"
			default:
				logger.Error("dialect can be sqlite, postgres or mysql only, not ", mi.dialect)
			}
		case "notnull":
			notnull = "NOT NULL"
		case "index":
			*mi.indexes=append(*mi.indexes,mi.fName)
		default:
			if strings.Contains(tag, ":") {
				sp := strings.Split(tag, ":")
				switch sp[0] {
				case "default":
					defaultt = " DEFAULT " + sp[1]
				case "fk":
					ref := strings.Split(sp[1], ".")
					if len(ref) == 2 {
						fkey := "FOREIGN KEY (" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
						if len(sp) > 2 {
							switch sp[2] {
							case "cascade":
								fkey += " ON DELETE CASCADE"
							case "donothing","noaction":
								fkey += " ON DELETE NO ACTION"
							case "setnull","null":
								fkey += " ON DELETE SET NULL"
							case "setdefault","default":
								fkey += " ON DELETE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled",sp[2])
							}
							if len(sp) > 3 {
								switch sp[3] {
								case "cascade":
									fkey += " ON UPDATE CASCADE"
								case "donothing","noaction":
									fkey += " ON UPDATE NO ACTION"
								case "setnull","null":
									fkey += " ON UPDATE SET NULL"
								case "setdefault","default":
									fkey += " ON UPDATE SET DEFAULT"
								default:
									logger.Printf("rdfk %s not handled",sp[3])
								}
							}
						}
						*mi.fKeys = append(*mi.fKeys, fkey)
					} else {
						logger.Error("allowed options cascade/donothing/noaction")
					}
				case "check":
					if strings.Contains(strings.ToLower(sp[1]), "len") {
						switch mi.dialect {
						case SQLITE, "":
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", 1)
						case POSTGRES, MYSQL:
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", 1)
						default:
							logger.Error("check not handled for dialect:", mi.dialect)
						}
					}
					check = " CHECK (" + sp[1] + ")"
				case "mindex":
					if v,ok := (*mi.mindexes)[mi.fName];ok {
						if v == "" {
							(*mi.mindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.mindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.mindexes)[mi.fName]=sp[1]
					}
				case "uindex":
					if v,ok := (*mi.uindexes)[mi.fName];ok {
						if v == "" {
							(*mi.uindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.uindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.uindexes)[mi.fName]=sp[1]
					}
				default:
					logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName)
				}

			} else {
				logger.Error("tag", tag, "not handled for", mi.fName, "of type", mi.fType)
			}
		}
	}
	
	if autoinc != "" {
		// integer auto increment
		(*mi.res)[mi.fName] = autoinc
	} else {
		// integer normal
		(*mi.res)[mi.fName] = "INTEGER"
		if primary != "" {
			(*mi.res)[mi.fName] += primary
		} else {
			if notnull != "" {
				(*mi.res)[mi.fName] += notnull
			}
		}
		if unique != "" {
			(*mi.res)[mi.fName] += unique
		}
		if index != "" {
			(*mi.res)[mi.fName] += index
		}
		if defaultt != "" {
			(*mi.res)[mi.fName] += defaultt
		}
		if check != "" {
			(*mi.res)[mi.fName] += check
		}
	}
}

func handleMigrationBool(mi *migrationInput) {
	defaultt := ""
	(*mi.res)[mi.fName] = "INTEGER NOT NULL CHECK (" + mi.fName + " IN (0, 1))"
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName]=""
		return
	}
	for _, tag := range tags {
		if strings.Contains(tag, ":") {
			sp := strings.Split(tag, ":")
			switch sp[0] {
			case "mindex":
				if v,ok := (*mi.mindexes)[mi.fName];ok {
					if v == "" {
						(*mi.mindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1],",") {
						(*mi.mindexes)[mi.fName] += ","+sp[1]
					} else {
						logger.Error("mindex not working for",mi.fName,sp[1])
					}
				} else {
					(*mi.mindexes)[mi.fName]=sp[1]
				}
			case "uindex":
				if v,ok := (*mi.uindexes)[mi.fName];ok {
					if v == "" {
						(*mi.uindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1],",") {
						(*mi.uindexes)[mi.fName] += ","+sp[1]
					} else {
						logger.Error("mindex not working for",mi.fName,sp[1])
					}
				} else {
					(*mi.uindexes)[mi.fName]=sp[1]
				}
			case "fk":
				ref := strings.Split(sp[1], ".")
				if len(ref) == 2 {
					fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
					if len(sp) > 2 {
						switch sp[2] {
						case "cascade":
							fkey += " ON DELETE CASCADE"
						case "donothing","noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull","null":
							fkey += " ON DELETE SET NULL"
						case "setdefault","default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled",sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing","noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull","null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault","default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled",sp[3])
							}
						}
					}
					*mi.fKeys = append(*mi.fKeys, fkey)
				} else {
					logger.Error("wtf ?, it should be fk:users.id:cascade/donothing")
				}
			}
		} else if tag == "index" {
			*mi.indexes=append(*mi.indexes,mi.fName)
		} else {
			logger.Error("tag", tag, "not handled for", mi.fName, "of type", mi.fType)		
		}
		if defaultt != "" {
			(*mi.res)[mi.fName] += defaultt
		}
	}
}

func handleMigrationString(mi *migrationInput) {
	unique, notnull, text, defaultt, size, pk, check := "", "", "", "", "", "", ""
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName]=""
		return
	}
	for _, tag := range tags {
		switch tag {
		case "unique":
			unique = " UNIQUE"
		case "text":
			text = " TEXT"
		case "notnull":
			notnull = " NOT NULL"
		case "pk":
			pk = " PRIMARY KEY"
		case "index":
			*mi.indexes=append(*mi.indexes,mi.fName)
		default:
			if strings.Contains(tag, ":") {
				sp := strings.Split(tag, ":")
				switch sp[0] {
				case "default":
					if sp[1] != "" {
						defaultt = " DEFAULT " + sp[1]
					} else {
						defaultt = " DEFAULT ''" 
					}
				case "mindex":
					if v,ok := (*mi.mindexes)[mi.fName];ok {
						if v == "" {
							(*mi.mindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.mindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.mindexes)[mi.fName]=sp[1]
					}
				case "uindex":
					if v,ok := (*mi.uindexes)[mi.fName];ok {
						if v == "" {
							(*mi.uindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.uindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.uindexes)[mi.fName]=sp[1]
					}
				case "fk":
					ref := strings.Split(sp[1], ".")
					if len(ref) == 2 {
						fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
						if len(sp) > 2 {
							switch sp[2] {
							case "cascade":
								fkey += " ON DELETE CASCADE"
							case "donothing","noaction":
								fkey += " ON DELETE NO ACTION"
							case "setnull","null":
								fkey += " ON DELETE SET NULL"
							case "setdefault","default":
								fkey += " ON DELETE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled",sp[2])
							}
							if len(sp) > 3 {
								switch sp[3] {
								case "cascade":
									fkey += " ON UPDATE CASCADE"
								case "donothing","noaction":
									fkey += " ON UPDATE NO ACTION"
								case "setnull","null":
									fkey += " ON UPDATE SET NULL"
								case "setdefault","default":
									fkey += " ON UPDATE SET DEFAULT"
								default:
									logger.Printf("rdfk %s not handled",sp[3])
								}
							}
						}
						*mi.fKeys = append(*mi.fKeys, fkey)
					} else {
						logger.Error("foreign key should be like fk:table.column:[cascade/donothing]")
					}
				case "size":
					sp := strings.Split(tag, ":")
					if sp[0] == "size" {
						size = sp[1]
					}
				case "check":
					if strings.Contains(strings.ToLower(sp[1]), "len") {
						switch mi.dialect {
						case SQLITE, "":
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", 1)
						case POSTGRES, MYSQL:
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", 1)
						default:
							logger.Error("check not handled for dialect:", mi.dialect)
						}
					}
					check = " CHECK (" + sp[1] + ")"
				default:
					logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName)
				}
			} else {
				logger.Error("tag", tag, "not handled for", mi.fName, "of type", mi.fType)
			}
		}
	}

	if text != "" {
		(*mi.res)[mi.fName] = text
	} else {
		if size != "" {
			(*mi.res)[mi.fName] = "VARCHAR(" + size + ")"
		} else {
			(*mi.res)[mi.fName] = "VARCHAR(255)"
		}
	}

	if unique != "" && pk == "" {
		(*mi.res)[mi.fName] += unique
	}
	if notnull != "" && pk == "" {
		(*mi.res)[mi.fName] += notnull
	}
	if pk != "" {
		(*mi.res)[mi.fName] += pk
	}
	if defaultt != "" {
		(*mi.res)[mi.fName] += defaultt
	}
	if check != "" {
		(*mi.res)[mi.fName] += check
	}
}

func handleMigrationFloat(mi *migrationInput) {
	mtags := map[string]string{}
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName]=""
		return
	}
	for _, tag := range tags {
		switch tag {
		case "notnull":
			mtags["notnull"] = " NOT NULL"
		case "unique":
			mtags["unique"] = " UNIQUE"
		case "pk":
			mtags["pk"] = " PRIMARY KEY"
		case "index":
			*mi.indexes=append(*mi.indexes,mi.fName)
		default:
			if strings.Contains(tag, ":") {
				sp := strings.Split(tag, ":")
				switch sp[0] {
				case "default":
					if sp[1] != "" {
						mtags["default"] = " DEFAULT " + sp[1]
					}
				case "fk":
					ref := strings.Split(sp[1], ".")
					if len(ref) == 2 {
						fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
						if len(sp) > 2 {
							switch sp[2] {
							case "cascade":
								fkey += " ON DELETE CASCADE"
							case "donothing","noaction":
								fkey += " ON DELETE NO ACTION"
							case "setnull","null":
								fkey += " ON DELETE SET NULL"
							case "setdefault","default":
								fkey += " ON DELETE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled",sp[2])
							}
							if len(sp) > 3 {
								switch sp[3] {
								case "cascade":
									fkey += " ON UPDATE CASCADE"
								case "donothing","noaction":
									fkey += " ON UPDATE NO ACTION"
								case "setnull","null":
									fkey += " ON UPDATE SET NULL"
								case "setdefault","default":
									fkey += " ON UPDATE SET DEFAULT"
								default:
									logger.Printf("rdfk %s not handled",sp[3])
								}
							}
						}
						*mi.fKeys = append(*mi.fKeys, fkey)
					} else {
						logger.Error("foreign key should be like fk:table.column:[cascade/donothing]")
					}
				case "mindex":
					if v,ok := (*mi.mindexes)[mi.fName];ok {
						if v == "" {
							(*mi.mindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.mindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.mindexes)[mi.fName]=sp[1]
					}
				case "uindex":
					if v,ok := (*mi.uindexes)[mi.fName];ok {
						if v == "" {
							(*mi.uindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.uindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.uindexes)[mi.fName]=sp[1]
					}
				case "check":
					if strings.Contains(strings.ToLower(sp[1]), "len") {
						switch mi.dialect {
						case SQLITE, "":
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", 1)
						case POSTGRES, MYSQL:
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", 1)
						default:
							logger.Error("check not handled for dialect:", mi.dialect)
						}
					}
					mtags["check"] = " CHECK (" + sp[1] + ")"
				default:
					logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName)
				}
			}
		}

		(*mi.res)[mi.fName] = "DECIMAL(5,2)"
		for k, v := range mtags {
			switch k {
			case "pk":
				(*mi.res)[mi.fName] += v
			case "notnull":
				if _, ok := mtags["pk"]; !ok {
					(*mi.res)[mi.fName] += v
				}
			case "unique":
				if _, ok := mtags["pk"]; !ok {
					(*mi.res)[mi.fName] += v
				}
			case "default":
				(*mi.res)[mi.fName] += v
			case "check":
				(*mi.res)[mi.fName] += v
			default:
				logger.Error("case", k, "not handled")
			}
		}
	}
}

func handleMigrationTime(mi *migrationInput) {
	defaultt, notnull, check := "", "", ""
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName]=""
		return
	}
	for _, tag := range tags {
		switch tag {
		case "now":
			switch mi.dialect {
			case SQLITE, "":
				defaultt = "TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP"
			case POSTGRES:
				defaultt = "TIMESTAMP NOT NULL DEFAULT (now())"
			case MYSQL:
				defaultt = "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"
			default:
				logger.Error("not handled Time for ", mi.fName, mi.fType)
			}
		case "notnull":
			if defaultt != "" {
				notnull = " NOT NULL"
			}
		case "index":
			*mi.indexes=append(*mi.indexes,mi.fName)
		default:
			if strings.Contains(tag, ":") {
				sp := strings.Split(tag, ":")
				switch sp[0] {
				case "mindex":
					if v,ok := (*mi.mindexes)[mi.fName];ok {
						if v == "" {
							(*mi.mindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.mindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.mindexes)[mi.fName]=sp[1]
					}
				case "uindex":
					if v,ok := (*mi.uindexes)[mi.fName];ok {
						if v == "" {
							(*mi.uindexes)[mi.fName] = sp[1]
						} else if strings.Contains(sp[1],",") {
							(*mi.uindexes)[mi.fName] += ","+sp[1]
						} else {
							logger.Error("mindex not working for",mi.fName,sp[1])
						}
					} else {
						(*mi.uindexes)[mi.fName]=sp[1]
					}
				case "fk":
					ref := strings.Split(sp[1], ".")
					if len(ref) == 2 {
						fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
						if len(sp) > 2 {
							switch sp[2] {
							case "cascade":
								fkey += " ON DELETE CASCADE"
							case "donothing","noaction":
								fkey += " ON DELETE NO ACTION"
							case "setnull","null":
								fkey += " ON DELETE SET NULL"
							case "setdefault","default":
								fkey += " ON DELETE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled",sp[2])
							}
							if len(sp) > 3 {
								switch sp[3] {
								case "cascade":
									fkey += " ON UPDATE CASCADE"
								case "donothing","noaction":
									fkey += " ON UPDATE NO ACTION"
								case "setnull","null":
									fkey += " ON UPDATE SET NULL"
								case "setdefault","default":
									fkey += " ON UPDATE SET DEFAULT"
								default:
									logger.Printf("rdfk %s not handled",sp[3])
								}
							}
						}
						*mi.fKeys = append(*mi.fKeys, fkey)
					} else {
						logger.Error("wtf ?, it should be fk:users.id:cascade/donothing")
					}
				case "check":
					if strings.Contains(strings.ToLower(sp[1]), "len") {
						switch mi.dialect {
						case SQLITE, "":
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", 1)
						case POSTGRES, MYSQL:
							sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", 1)
						default:
							logger.Error("check not handled for dialect:", mi.dialect)
						}
					}
					check = " CHECK (" + sp[1] + ")"
				case "default":
					if sp[1] != "" {
						switch mi.dialect {
						case SQLITE, "":
							defaultt = "TEXT NOT NULL DEFAULT " + sp[1]
						case POSTGRES:
							defaultt = "TIMESTAMP with time zone NOT NULL DEFAULT " + sp[1]
						case MYSQL:
							defaultt = "TIMESTAMP with time zone NOT NULL DEFAULT " + sp[1]
						default:
							logger.Error("default for field", mi.fName, "not handled")
						}
					}
				default:
					logger.Error("case", sp[0], "not handled")
				}
			}
		}
	}
	if defaultt != "" {
		(*mi.res)[mi.fName] = defaultt
	} else {
		if mi.dialect == "" || mi.dialect == SQLITE {
			(*mi.res)[mi.fName] = "TEXT"
		} else {
			(*mi.res)[mi.fName] = "TIMESTAMP with time zone"
		}

		if notnull != "" {
			(*mi.res)[mi.fName] += notnull
		}
		if check != "" {
			(*mi.res)[mi.fName] += check
		}
	}
}

func prepareCreateStatement(tbName string, fields map[string]string, fkeys, cols []string,db *DatabaseEntity,ftags map[string][]string) string {
	st := "CREATE TABLE IF NOT EXISTS "
	st += tbName + " ("
	for i, col := range cols {
		fName := col
		fType := fields[col]
		if fType == "" {
			continue
		}
		reste := ","
		if i == len(fields)-1 {
			reste = ""
		}
		st += fName + " " + fType + reste
	}
	if len(fkeys) > 0 {
		st += ","
	}
	for i, k := range fkeys {
		st += k
		if i < len(fkeys)-1 {
			st += ","
		}
	}
	st = strings.TrimSuffix(st,",")
	st += ");"
	return st
}
