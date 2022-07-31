package orm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kamalshkeir/kago/core/settings"
	"github.com/kamalshkeir/kago/core/utils"
	"github.com/kamalshkeir/kago/core/utils/eventbus"
	"github.com/kamalshkeir/kago/core/utils/logger"
	"github.com/kamalshkeir/kago/core/utils/safemap"
)

var cachesOneS = safemap.New[dbCache, any]()
var cachesAllS = safemap.New[dbCache, any]()

type Builder[T comparable] struct {
	debug      bool
	conn       *sql.DB
	limit      int
	page       int
	tableName  string
	selected   string
	orderBys   string
	whereQuery string
	query      string
	offset     string
	statement  string
	dialect    string
	database   string
	args       []any
	order      []string
	ctx        context.Context
}

func Model[T comparable]() *Builder[T] {
	tName := getTableName[T]()
	if tName == "" {
		logger.Error("unable to find tableName from model, restart the app if you just migrated")
		return nil
	}
	return &Builder[T]{
		tableName: tName,
	}
}

func (b *Builder[T]) Database(dbName string) *Builder[T] {
	if b.tableName == "" {
		logger.Error("you should specify the model before database")
		return nil
	}
	b.database = dbName
	return b
}

func (b *Builder[T]) Insert(model *T) (int, error) {
	tName := getTableName[T]()
	if tName == "" {
		return 0, errors.New("unable to find tableName from model, restart the app if you just migrated")
	}
	if b.database == "" {
		b.database = settings.GlobalConfig.DbName
	}
	if b.conn == nil {
		if con, ok := mDbNameConnection[b.database]; ok {
			b.conn = con
		} else {

			b.conn = databases[0].Conn
		}
	}
	if b.dialect == "" {
		if dial, ok := mDbNameDialect[b.database]; ok {
			b.dialect = dial
		} else {
			b.dialect = databases[0].Dialect
		}
	}

	if UseCache {
		eventbus.Publish(CACHE_TOPIC, map[string]string{
			"type":     "create",
			"table":    b.tableName,
			"database": b.database,
		})
	}

	names, mvalues, _, mtags := getStructInfos(model)
	values := []any{}
	if len(names) < len(mvalues) {
		return 0, errors.New("there is more values than fields")
	}
	placeholdersSlice := []string{}
	index := 999
	for i, name := range names {
		if v, ok := mvalues[name]; ok {
			values = append(values, v)
		} else {
			logger.Error(v, "not found in fields")
			return 0, errors.New("field not found")
		}
		if utils.SliceContains(mtags[name], "autoinc", "pk") || (strings.Contains(name, "id") && i == 0) {
			index = i
		} else {
			placeholdersSlice = append(placeholdersSlice, "?")
		}
	}
	if index != 999 {
		names = append(names[:index], names[index+1:]...)
		values = append(values[:index], values[index+1:]...)
		delete(mvalues, names[index])
	}

	placeholders := strings.Join(placeholdersSlice, ",")
	fields_comma_separated := strings.Join(names, ",")
	var affectedRows int
	stat := strings.Builder{}
	stat.WriteString("INSERT INTO " + b.tableName + " (")
	stat.WriteString(fields_comma_separated)
	stat.WriteString(") VALUES (")
	stat.WriteString(placeholders)
	stat.WriteString(")")
	b.statement = stat.String()
	adaptPlaceholdersToDialect(&b.statement, b.dialect)
	var res sql.Result
	var err error
	if b.ctx != nil {
		res, err = b.conn.ExecContext(b.ctx, b.statement, values...)
	} else {
		res, err = b.conn.Exec(b.statement, values...)
	}
	if err != nil {
		logger.Info(b.statement, values)
		return affectedRows, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return int(rows), err
	}
	return int(rows), nil
}

func (b *Builder[T]) Set(query string, args ...any) (int, error) {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return 0, errors.New("unable to find tableName from model")
		}
		b.tableName = tName
	}
	if b.database == "" {
		b.database = settings.GlobalConfig.DbName
	}
	if b.conn == nil {
		if con, ok := mDbNameConnection[b.database]; ok {
			b.conn = con
		} else {

			b.conn = databases[0].Conn
		}
	}
	if b.dialect == "" {
		if dial, ok := mDbNameDialect[b.database]; ok {
			b.dialect = dial
		} else {
			b.dialect = databases[0].Dialect
		}
	}
	if UseCache {
		eventbus.Publish(CACHE_TOPIC, map[string]string{
			"type":     "update",
			"table":    b.tableName,
			"database": b.database,
		})
	}
	if b.whereQuery == "" {
		return 0, errors.New("you should use Where before Update")
	}

	b.statement = "UPDATE " + b.tableName + " SET " + query + " WHERE " + b.whereQuery
	adaptPlaceholdersToDialect(&b.statement, b.dialect)
	args = append(args, b.args...)
	if b.debug {
		logger.Debug("statement:", b.statement)
		logger.Debug("args:", b.args)
	}

	var res sql.Result
	var err error
	if b.ctx != nil {
		res, err = b.conn.ExecContext(b.ctx, b.statement, args...)
	} else {
		res, err = b.conn.Exec(b.statement, args...)
	}
	if err != nil {
		return 0, err
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(aff), nil
}

func (b *Builder[T]) Delete() (int, error) {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return 0, errors.New("unable to find tableName from model")
		}
		b.tableName = tName
	}
	if b.database == "" {
		b.database = settings.GlobalConfig.DbName
	}
	if b.conn == nil {
		if con, ok := mDbNameConnection[b.database]; ok {
			b.conn = con
		} else {

			b.conn = databases[0].Conn
		}
	}
	if b.dialect == "" {
		if dial, ok := mDbNameDialect[b.database]; ok {
			b.dialect = dial
		} else {
			b.dialect = databases[0].Dialect
		}
	}
	if UseCache {
		eventbus.Publish(CACHE_TOPIC, map[string]string{
			"type":     "delete",
			"table":    b.tableName,
			"database": b.database,
		})
	}

	b.statement = "DELETE FROM " + b.tableName
	if b.whereQuery != "" {
		b.statement += " WHERE " + b.whereQuery
	} else {
		return 0, errors.New("no Where was given for this query:" + b.whereQuery)
	}
	adaptPlaceholdersToDialect(&b.statement, b.dialect)
	if b.debug {
		logger.Debug("statement:", b.statement)
		logger.Debug("args:", b.args)
	}

	var res sql.Result
	var err error

	if b.ctx != nil {
		res, err = b.conn.ExecContext(b.ctx, b.statement, b.args...)
	} else {
		res, err = b.conn.Exec(b.statement, b.args...)
	}
	if err != nil {
		return 0, err
	}
	affectedRows, err := res.RowsAffected()
	if err != nil {
		return int(affectedRows), err
	}
	return int(affectedRows), nil
}

func (b *Builder[T]) Drop() (int, error) {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			return 0, errors.New("unable to find tableName from model")
		}
		b.tableName = tName
	}
	if b.database == "" {
		b.database = settings.GlobalConfig.DbName
	}
	if b.conn == nil {
		if con, ok := mDbNameConnection[b.database]; ok {
			b.conn = con
		} else {

			b.conn = databases[0].Conn
		}
	}
	if b.dialect == "" {
		if dial, ok := mDbNameDialect[b.database]; ok {
			b.dialect = dial
		} else {
			b.dialect = databases[0].Dialect
		}
	}
	if UseCache {
		eventbus.Publish(CACHE_TOPIC, map[string]string{
			"type":     "drop",
			"table":    b.tableName,
			"database": b.database,
		})
	}
	b.statement = "DROP TABLE " + b.tableName
	var res sql.Result
	var err error
	if b.ctx != nil {
		res, err = b.conn.ExecContext(b.ctx, b.statement)
	} else {
		res, err = b.conn.Exec(b.statement)
	}
	if err != nil {
		return 0, err
	}
	aff, err := res.RowsAffected()
	if err != nil {
		return int(aff), err
	}
	return int(aff), err
}

func (b *Builder[T]) Select(columns ...string) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	s := []string{}
	s = append(s, columns...)
	b.selected = strings.Join(s, ",")
	b.order = append(b.order, "select")
	return b
}

func (b *Builder[T]) Where(query string, args ...any) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.whereQuery = query
	b.args = append(b.args, args...)
	b.order = append(b.order, "where")
	return b
}

func (b *Builder[T]) Query(query string, args ...any) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.query = query
	b.args = append(b.args, args...)
	b.order = append(b.order, "query")
	return b
}

func (b *Builder[T]) Limit(limit int) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.limit = limit
	b.order = append(b.order, "limit")
	return b
}

func (b *Builder[T]) Context(ctx context.Context) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.ctx = ctx
	return b
}

func (b *Builder[T]) Page(pageNumber int) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.page = pageNumber
	b.order = append(b.order, "page")
	return b
}

func (b *Builder[T]) OrderBy(fields ...string) *Builder[T] {
	if b.tableName == "" {
		tName := getTableName[T]()
		if tName == "" {
			logger.Error("unable to find tableName from model")
			return nil
		}
		b.tableName = tName
	}
	b.orderBys = "ORDER BY "
	orders := []string{}
	for _, f := range fields {
		if strings.HasPrefix(f, "+") {
			orders = append(orders, f[1:]+" ASC")
		} else if strings.HasPrefix(f, "-") {
			orders = append(orders, f[1:]+" DESC")
		} else {
			orders = append(orders, f+" ASC")
		}
	}
	b.orderBys += strings.Join(orders, ",")
	b.order = append(b.order, "order_by")
	return b
}

func (b *Builder[T]) Debug() *Builder[T] {
	b.debug = true
	return b
}

func (b *Builder[T]) All() ([]T, error) {
	if b.database == "" {
		b.database=settings.GlobalConfig.DbName
	}
	if b.tableName == "" {
		return nil, errors.New("error: this model is not linked, execute orm.AutoMigrate before")
	}
	c := dbCache{
		database:   b.database,
		table:      b.tableName,
		selected:   b.selected,
		statement:  b.statement,
		orderBys:   b.orderBys,
		whereQuery: b.whereQuery,
		query:      b.query,
		offset:     b.offset,
		limit:      b.limit,
		page:       b.page,
		args:       fmt.Sprintf("%v", b.args...),
	}
	if UseCache {
		if v, ok := cachesAllS.Get(c); ok {
			return v.([]T), nil
		}
	}

	if b.tableName == "" {
		return nil, errors.New("unable to find model, try orm.LinkModel before")
	}

	if b.selected != "" && b.selected != "*" {
		b.statement = "select " + b.selected + " from " + b.tableName
	} else {
		b.statement = "select * from " + b.tableName
	}

	if b.whereQuery != "" {
		b.statement += " WHERE " + b.whereQuery
	}
	if b.query != "" {
		b.limit = 0
		b.orderBys = ""
		b.statement = b.query
	}

	if b.orderBys != "" {
		b.statement += " " + b.orderBys
	}

	if b.limit > 0 {
		i := strconv.Itoa(b.limit)
		b.statement += " LIMIT " + i
		if b.page > 0 {
			o := strconv.Itoa((b.page - 1) * b.limit)
			b.statement += " OFFSET " + o
		}
	}

	if b.debug {
		logger.Debug("statement:", b.statement)
		logger.Debug("args:", b.args)
	}

	models, err := b.queryS(b.statement, b.args...)
	if err != nil {
		return nil, err
	}
	if UseCache {
		cachesAllS.Set(c, models)
	}
	return models, nil
}

func (b *Builder[T]) One() (T, error) {
	if b.database == "" {
		b.database=settings.GlobalConfig.DbName
	}
	if b.tableName == "" {
		return *new(T), errors.New("error: this model is not linked, execute orm.AutoMigrate first")
	}
	c := dbCache{
		database:   b.database,
		table:      b.tableName,
		selected:   b.selected,
		statement:  b.statement,
		orderBys:   b.orderBys,
		whereQuery: b.whereQuery,
		query:      b.query,
		offset:     b.offset,
		limit:      b.limit,
		page:       b.page,
		args:       fmt.Sprintf("%v", b.args...),
	}
	if UseCache {
		if v, ok := cachesOneS.Get(c); ok {
			return v.(T), nil
		}
	}

	if b.tableName == "" {
		return *new(T), errors.New("unable to find model, try orm.LinkModel before")
	}

	if b.selected != "" && b.selected != "*" {
		b.statement = "select " + b.selected + " from " + b.tableName
	} else {
		b.statement = "select * from " + b.tableName
	}

	if b.whereQuery != "" {
		b.statement += " WHERE " + b.whereQuery
	}

	if b.query != "" {
		b.limit = 0
		b.orderBys = ""
		b.statement = b.query
	}

	if b.orderBys != "" {
		b.statement += " " + b.orderBys
	}

	if b.limit > 0 {
		i := strconv.Itoa(b.limit)
		b.statement += " LIMIT " + i
	}

	if b.debug {
		logger.Debug("statement:", b.statement)
		logger.Debug("args:", b.args)
	}

	models, err := b.queryS(b.statement, b.args...)
	if err != nil {
		return *new(T), err
	}
	if UseCache {
		cachesOneS.Set(c, models[0])
	}
	return models[0], nil
}

func (b *Builder[T]) queryS(query string, args ...any) ([]T, error) {
	if b.database == "" {
		b.database=settings.GlobalConfig.DbName
	}
	db,err := GetDatabase(b.database)
	if logger.CheckError(err) {
		return nil,err
	}

	adaptPlaceholdersToDialect(&query, db.Dialect)
	res := make([]T, 0)
	
	var rows *sql.Rows
	if b.ctx != nil {
		rows, err = db.Conn.QueryContext(b.ctx, query, args...)
	} else {
		rows, err = db.Conn.Query(query, args...)
	}

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no data found")
	} else if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []string
	if b.selected != "" && b.selected != "*" {
		cols = strings.Split(b.selected, ",")
	} else {
		cols, err = rows.Columns()
		if err != nil {
			return nil, err
		}
	}

	columns_ptr_to_values := make([]interface{}, len(cols))
	values := make([]interface{}, len(cols))
	for rows.Next() {
		for i := range values {
			columns_ptr_to_values[i] = &values[i]
		}

		err := rows.Scan(columns_ptr_to_values...)
		if err != nil {
			return nil, err
		}

		if b.dialect == "mysql" {
			for i := range values {
				if v, ok := values[i].([]byte); ok {
					values[i] = string(v)
				}
			}
		}

		row := new(T)
		if b.selected != "" && b.selected != "*" {
			fillStructColumns(row, b.selected, values...)
		} else {
			fillStruct(row,values...)
		}

		res = append(res, *row)
	}

	if len(res) == 0 {
		return nil, errors.New("no data found")
	}
	return res, nil
}
