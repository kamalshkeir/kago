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
	err := AutoMigrate[models.User]("users", settings.Config.Db.Name)
	if logger.CheckError(err) {
		return err
	}
	return nil
}

func checkUpdatedAtTrigger(dialect,tableName,col,pk string) map[string][]string {
	triggers := map[string][]string{}
	t := "(datetime('now','localtime'))"
	if dialect == "sqlite" {
		st:="CREATE TRIGGER "
		st+=tableName+"_update_trig AFTER UPDATE ON "+tableName
		st+=" BEGIN update "+tableName+ " SET "+ col + " = " +  t 
		st+=" WHERE " + pk + " = " + "NEW."+pk+";"
		st+="End;"
		triggers[col]=[]string{st}
	} else if dialect == "postgres" {	
		st:="CREATE OR REPLACE FUNCTION updated_at_trig() RETURNS trigger AS $$"
		st+=" BEGIN NEW."+col+" = now();RETURN NEW;"
		st+="END;$$ LANGUAGE plpgsql;"
		triggers[col]=[]string{st}
		trigCreate := "CREATE OR REPLACE TRIGGER "+tableName+"_update_trig"
		trigCreate += " BEFORE UPDATE ON public."+tableName
		trigCreate += " FOR EACH ROW EXECUTE PROCEDURE updated_at_trig();"
		triggers[col]=append(triggers[col], trigCreate)
	} else {
		return nil
	}
	return triggers
}

func autoMigrate[T comparable](db *DatabaseEntity, tableName string) error {
	dialect := db.Dialect
	s := reflect.ValueOf(new(T)).Elem()
	typeOfT := s.Type()
	mFieldName_Type := map[string]string{}
	mFieldName_Tags := map[string][]string{}
	cols := []string{}
	pk := ""
	for i := 0; i < s.NumField(); i++ {
		f := s.Field(i)
		fname := typeOfT.Field(i).Name
		fname = ToSnakeCase(fname)
		ftype := f.Type()
		cols = append(cols, fname)
		mFieldName_Type[fname] = ftype.Name()
		if ftag, ok := typeOfT.Field(i).Tag.Lookup("orm"); ok {
			tags := strings.Split(ftag, ";")
			for i,tag := range tags {
				if tag == "autoinc" || tag == "pk" {
					pk = fname
				}
				tags[i] = strings.TrimSpace(tags[i])
			}
			mFieldName_Tags[fname] = tags
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
				table: tableName,
				dialect:  dialect,
				fName:    fName,
				fType:    ty,
				fTags:    &mFieldName_Tags,
				fKeys:    &fkeys,
				res:      &res,
				indexes:  &indexes,
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
	statement := prepareCreateStatement(tableName, res, fkeys, cols, db, mFieldName_Tags)
	var triggers map[string][]string
	tbFound := false
	
	// check if table in memory
	for _, t := range db.Tables {
		if t.Name == tableName {
			tbFound = true
			if len(t.Columns) == 0 {
				t.Columns = cols
			}
			if len(t.Tags) == 0 {
				t.Tags = mFieldName_Tags
			}
			if len(t.ModelTypes) == 0 {
				t.Types = mFieldName_Type
			}
		}
	}
	// check for update field to create a trigger
	if db.Dialect != MYSQL {
		for col, tags := range mFieldName_Tags {
			for _, tag := range tags {
				if tag == "update" {
					triggers = checkUpdatedAtTrigger(db.Dialect,tableName,col,pk)	
				}
			}
		}
	}
	
	if !tbFound {
		db.Tables = append(db.Tables, TableEntity{
			Name:       tableName,
			Columns:    cols,
			Tags:       mFieldName_Tags,
			ModelTypes: mFieldName_Type,
			Pk:         pk,
		})
	}
	if Debug {
		fmt.Printf(logger.Blue, "statement: "+statement)
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
	if !strings.HasSuffix(tableName, "_temp") {
		if len(triggers) > 0 {
			for _,stats := range triggers {
				for _,st := range stats {
					if Debug {
						logger.Printfs("trigger updated_at %s: %s",tableName, st)
					}
					err := Exec(db.Name,st)
					if logger.CheckError(err) {
						logger.Printfs("rdtrigger updated_at %s: %s",tableName, st)
						return err
					}
				}
			}
		}
		statIndexes := ""
		if len(indexes) > 0 {
			if len(indexes) > 1 {
				logger.Error(mi.fName, "cannot have more than 1 index")
			} else {
				statIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)", tableName, indexes[0], tableName, indexes[0])
			}
		}
		mstatIndexes := ""
		if len(*mi.mindexes) > 0 {
			if len(*mi.mindexes) > 1 {
				logger.Error(mi.fName, "cannot have more than 1 multiple indexes")
			} else {
				for k, v := range *mi.mindexes {
					mstatIndexes = fmt.Sprintf("CREATE INDEX idx_%s_%s ON %s (%s)", tableName, k, tableName, k+","+v)
				}
			}
		}
		ustatIndexes := []string{}
		for col, tagValue := range *mi.uindexes {
			sp := strings.Split(tagValue,",")
			for i := range sp {
				if sp[i][0] == 'I' {
					sp[i] = "LOWER("+sp[i][1:]+")"
				}
			}
			res := strings.Join(sp,",")
			ustatIndexes = append(ustatIndexes, fmt.Sprintf("CREATE UNIQUE INDEX idx_%s_%s ON %s (%s)", tableName, col, tableName, res)) 
		}
		if statIndexes != "" {
			if Debug {
				logger.Printfs(statIndexes)
			}
			_, err := db.Conn.Exec(statIndexes)
			if logger.CheckError(err) {
				logger.Printfs("rdindexes: %s", statIndexes)
				return err
			}
		}
		if mstatIndexes != "" {
			if Debug {
				logger.Printfs("mindexes: %s", mstatIndexes)
			}
			_, err := db.Conn.Exec(mstatIndexes)
			if logger.CheckError(err) {
				logger.Printfs("rdmindexes: %s", mstatIndexes)
				return err
			}
		}
		if len(ustatIndexes) > 0 {
			for i := range ustatIndexes {
				if Debug {
					logger.Printfs("uindexes: %s", ustatIndexes[i])
				}
				_, err := db.Conn.Exec(ustatIndexes[i])
				if logger.CheckError(err) {
					logger.Printfs("rduindexes: %s", ustatIndexes)
					return err
				}
			}
			
		}
	}

	logger.Printfs("gr%s migrated successfully, restart the server", tableName)
	return nil
}

func AutoMigrate[T comparable](tableName string, dbName ...string) error {
	if _, ok := mModelTablename[*new(T)]; !ok {
		mModelTablename[*new(T)] = tableName
	}
	var db *DatabaseEntity
	var err error
	dbname := ""
	if len(dbName) > 0 {
		dbname = dbName[0]
		db, err = GetDatabase(dbname)
		if err != nil || db == nil {
			return errors.New("database not found")
		}
	} else {
		dbname = settings.Config.Db.Name
		db, err = GetDatabase(dbname)
		if err != nil || db == nil {
			return errors.New("database not found")
		}
	}

	tbFoundDB := false
	tables := GetAllTables(dbname)
	for _, t := range tables {
		if t == tableName {
			tbFoundDB = true
		}
	}

	tbFoundLocal := false
	if len(db.Tables) == 0 {
		if tbFoundDB {
			// found db not local
			linkModel[T](tableName, db)
			return nil
		} else {
			// not db and not local
			err := autoMigrate[T](db, tableName)
			if logger.CheckError(err) {
				return err
			}
			return nil
		}
	} else {
		// db have tables
		for _, t := range db.Tables {
			if t.Name == tableName {
				tbFoundLocal = true
			}
		}
	}
	if !tbFoundLocal {
		if tbFoundDB {
			linkModel[T](tableName, db)
			return nil
		} else {
			err := autoMigrate[T](db, tableName)
			if logger.CheckError(err) {
				return err
			}
		}
	}

	return nil
}

type migrationInput struct {
	table    string
	dialect  string
	fName    string
	fType    string
	fTags    *map[string][]string
	fKeys    *[]string
	res      *map[string]string
	indexes  *[]string
	mindexes *map[string]string
	uindexes *map[string]string
}

func handleMigrationInt(mi *migrationInput) {
	primary, index, autoinc, notnull, defaultt, checks, unique := "", "", "", "", "", []string{}, ""
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName] = ""
		return
	}
	for _, tag := range tags {
		if !strings.Contains(tag, ":") {
			switch tag {
			case "autoinc", "pk":
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
				*mi.indexes = append(*mi.indexes, mi.fName)
			case "unique":
				unique = " UNIQUE"
			case "default":
				defaultt = " DEFAULT 0"
			default:
				logger.Error(tag, "not handled for migration int")
			}
		} else {
			sp := strings.Split(tag, ":")
			tg := sp[0]
			switch tg {
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
						case "donothing", "noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull", "null":
							fkey += " ON DELETE SET NULL"
						case "setdefault", "default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled", sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing", "noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull", "null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault", "default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled", sp[3])
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
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", -1)
					case POSTGRES, MYSQL:
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", -1)
					default:
						logger.Error("check not handled for dialect:", mi.dialect)
					}
				}
				checks = append(checks, strings.TrimSpace(sp[1]))
			case "mindex":
				if v, ok := (*mi.mindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.mindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.mindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.mindexes)[mi.fName] = sp[1]
				}
			case "uindex":
				if v, ok := (*mi.uindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.uindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.uindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.uindexes)[mi.fName] = sp[1]
				}
			default:
				logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName, "for migration int")
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
		if len(checks) > 0 {
			joined := strings.TrimSpace(strings.Join(checks, " AND "))
			(*mi.res)[mi.fName] += " CHECK(" + joined + ")"
		}
	}
}

func handleMigrationBool(mi *migrationInput) {
	defaultt := ""
	(*mi.res)[mi.fName] = "INTEGER NOT NULL CHECK (" + mi.fName + " IN (0, 1))"
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName] = ""
		return
	}
	for _, tag := range tags {
		if strings.Contains(tag, ":") {
			sp := strings.Split(tag, ":")
			switch sp[0] {
			case "default":
				if sp[1] != "" {
					if sp[1] == "true" {
						defaultt = " DEFAULT 1"
					} else if sp[1] == "false" {
						defaultt = " DEFAULT 0"
					} else {
						defaultt = " DEFAULT " + sp[1]
					}
				} else {
					defaultt = " DEFAULT false"
				}
			case "mindex":
				if v, ok := (*mi.mindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.mindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.mindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.mindexes)[mi.fName] = sp[1]
				}
			case "fk":
				ref := strings.Split(sp[1], ".")
				if len(ref) == 2 {
					fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
					if len(sp) > 2 {
						switch sp[2] {
						case "cascade":
							fkey += " ON DELETE CASCADE"
						case "donothing", "noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull", "null":
							fkey += " ON DELETE SET NULL"
						case "setdefault", "default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled", sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing", "noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull", "null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault", "default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled", sp[3])
							}
						}
					}
					*mi.fKeys = append(*mi.fKeys, fkey)
				} else {
					logger.Error("wtf ?, it should be fk:users.id:cascade/donothing")
				}
			default:
				logger.Error(sp[0], "not handled for", mi.fName, "migration bool")
			}
		} else {
			switch tag {
			case "index":
				*mi.indexes = append(*mi.indexes, mi.fName)
			case "default":
				defaultt = " DEFAULT 0"
			default:
				logger.Error(tag, "not handled in Migration Bool")
			}
		}
	}
	if defaultt != "" {
		(*mi.res)[mi.fName] += defaultt
	}
}

func handleMigrationString(mi *migrationInput) {
	unique, notnull, text, defaultt, size, pk, checks := "", "", "", "", "", "", []string{}
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName] = ""
		return
	}
	for _, tag := range tags {
		if !strings.Contains(tag, ":") {
			switch tag {
			case "text":
				text = "TEXT"
			case "notnull":
				notnull = " NOT NULL"
			case "index":
				*mi.indexes = append(*mi.indexes, mi.fName)
			case "unique":
				unique = " UNIQUE"
			case "iunique":
				(*mi.uindexes)[mi.fName] = "I"+mi.fName
			case "default":
				defaultt = " DEFAULT ''"
			default:
				logger.Error(tag, "not handled for migration string")
			}
		} else {
			sp := strings.Split(tag, ":")
			switch sp[0] {
			case "default":
				defaultt = " DEFAULT " + sp[1]
			case "mindex":
				if v, ok := (*mi.mindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.mindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.mindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.mindexes)[mi.fName] = sp[1]
				}
			case "uindex":
				if v, ok := (*mi.uindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.uindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.uindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.uindexes)[mi.fName] = sp[1]
				}
			case "fk":
				ref := strings.Split(sp[1], ".")
				if len(ref) == 2 {
					fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
					if len(sp) > 2 {
						switch sp[2] {
						case "cascade":
							fkey += " ON DELETE CASCADE"
						case "donothing", "noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull", "null":
							fkey += " ON DELETE SET NULL"
						case "setdefault", "default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled", sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing", "noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull", "null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault", "default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled", sp[3])
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
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", -1)
					case POSTGRES, MYSQL:
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", -1)
					default:
						logger.Error("check not handled for dialect:", mi.dialect)
					}
				}
				checks = append(checks, strings.TrimSpace(sp[1]))
			default:
				logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName, "migration string")
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
	if len(checks) > 0 {
		joined := strings.TrimSpace(strings.Join(checks, " AND "))
		(*mi.res)[mi.fName] += " CHECK(" + joined + ")"
	}
}

func handleMigrationFloat(mi *migrationInput) {
	mtags := map[string]string{}
	tags := (*mi.fTags)[mi.fName]
	if len(tags) == 1 && tags[0] == "-" {
		(*mi.res)[mi.fName] = ""
		return
	}
	for _, tag := range tags {
		if !strings.Contains(tag, ":") {
			switch tag {
			case "notnull":
				mtags["notnull"] = " NOT NULL"
			case "index":
				*mi.indexes = append(*mi.indexes, mi.fName)
			case "unique":
				(*mi.uindexes)[mi.fName] = " UNIQUE"
			case "default":
				mtags["default"] = " DEFAULT 0.00"
			default:
				logger.Error(tag, "not handled for migration float")
			}
		} else {
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
						case "donothing", "noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull", "null":
							fkey += " ON DELETE SET NULL"
						case "setdefault", "default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled", sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing", "noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull", "null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault", "default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled", sp[3])
							}
						}
					}
					*mi.fKeys = append(*mi.fKeys, fkey)
				} else {
					logger.Error("foreign key should be like fk:table.column:[cascade/donothing]")
				}
			case "mindex":
				if v, ok := (*mi.mindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.mindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.mindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.mindexes)[mi.fName] = sp[1]
				}
			case "uindex":
				if v, ok := (*mi.uindexes)[mi.fName]; ok {
					if v == "" {
						(*mi.uindexes)[mi.fName] = sp[1]
					} else if strings.Contains(sp[1], ",") {
						(*mi.uindexes)[mi.fName] += "," + sp[1]
					} else {
						logger.Error("mindex not working for", mi.fName, sp[1])
					}
				} else {
					(*mi.uindexes)[mi.fName] = sp[1]
				}
			case "check":
				if strings.Contains(strings.ToLower(sp[1]), "len") {
					switch mi.dialect {
					case SQLITE, "":
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", -1)
					case POSTGRES, MYSQL:
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", -1)
					default:
						logger.Error("check not handled for dialect:", mi.dialect)
					}
				}
				if v, ok := mtags["check"]; ok && v != "" {
					mtags["check"] += " AND " + strings.TrimSpace(sp[1])
				} else if v == "" {
					mtags["check"] = strings.TrimSpace(sp[1])
				}
			default:
				logger.Error("not handled", sp[0], "for", tag, ",field:", mi.fName, "field float")
			}
		}

		(*mi.res)[mi.fName] = "DECIMAL(10,2)"
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
				(*mi.res)[mi.fName] += " CHECK(" + v + ")"
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
		(*mi.res)[mi.fName] = ""
		return
	}
	for _, tag := range tags {
		if !strings.Contains(tag, ":") {
			switch tag {
			case "now":
				switch mi.dialect {
				case SQLITE, "":
					defaultt = "TEXT NOT NULL DEFAULT (datetime('now','localtime'))"
				case POSTGRES:
					defaultt = "TIMESTAMP NOT NULL DEFAULT (now())"
				case MYSQL:
					defaultt = "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP"
				default:
					logger.Error("not handled Time for ", mi.fName, mi.fType)
				}
			case "update":
				switch mi.dialect {
				case SQLITE, "":
					defaultt = "TEXT NOT NULL DEFAULT (datetime('now','localtime'))"
				case POSTGRES:
					defaultt = "TIMESTAMP NOT NULL DEFAULT (now())"
				case MYSQL:
					defaultt = "TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP"
				default:
					logger.Error("not handled Time for ", mi.fName, mi.fType)
				}
			default:
				logger.Error(tag, "tag not handled for time")
			}
		} else {
			sp := strings.Split(tag, ":")
			switch sp[0] {
			case "fk":
				ref := strings.Split(sp[1], ".")
				if len(ref) == 2 {
					fkey := "FOREIGN KEY(" + mi.fName + ") REFERENCES " + ref[0] + "(" + ref[1] + ")"
					if len(sp) > 2 {
						switch sp[2] {
						case "cascade":
							fkey += " ON DELETE CASCADE"
						case "donothing", "noaction":
							fkey += " ON DELETE NO ACTION"
						case "setnull", "null":
							fkey += " ON DELETE SET NULL"
						case "setdefault", "default":
							fkey += " ON DELETE SET DEFAULT"
						default:
							logger.Printf("rdfk %s not handled", sp[2])
						}
						if len(sp) > 3 {
							switch sp[3] {
							case "cascade":
								fkey += " ON UPDATE CASCADE"
							case "donothing", "noaction":
								fkey += " ON UPDATE NO ACTION"
							case "setnull", "null":
								fkey += " ON UPDATE SET NULL"
							case "setdefault", "default":
								fkey += " ON UPDATE SET DEFAULT"
							default:
								logger.Printf("rdfk %s not handled", sp[3])
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
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "length", -1)
					case POSTGRES, MYSQL:
						sp[1] = strings.Replace(strings.ToLower(sp[1]), "len", "char_length", -1)
					default:
						logger.Error("check not handled for dialect:", mi.dialect)
					}
				}
				if check != "" {
					check += " AND " + strings.TrimSpace(sp[1])
				} else {
					check += sp[1]
				}
			default:
				logger.Error("case", sp[0], "not handled for time")
			}
		}
	}
	if defaultt != "" {
		(*mi.res)[mi.fName] = defaultt
	} else {
		if mi.dialect == "" || mi.dialect == SQLITE {
			(*mi.res)[mi.fName] = "TEXT"
		} else {
			(*mi.res)[mi.fName] = "TIMESTAMP"
		}

		if notnull != "" {
			(*mi.res)[mi.fName] += notnull
		}
		if check != "" {
			(*mi.res)[mi.fName] += " CHECK(" + check + ")"
		}
	}
}

func prepareCreateStatement(tbName string, fields map[string]string, fkeys, cols []string, db *DatabaseEntity, ftags map[string][]string) string {
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
	st = strings.TrimSuffix(st, ",")
	st += ");"
	return st
}
