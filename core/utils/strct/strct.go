package strct

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func FillValues(struct_to_fill any, values_to_fill ...any) {
	rs := reflect.ValueOf(struct_to_fill)
	if rs.Kind() == reflect.Pointer {
		rs = reflect.ValueOf(struct_to_fill).Elem()
	}
	//tt := reflect.TypeOf(struct_to_fill).Elem()
	for i := 0; i < rs.NumField(); i++ {
		field := rs.Field(i)
		if field.IsValid() {
			SetFieldValue(field,values_to_fill[i])
		}
	}
}

func FillSelectedValues(struct_to_fill any, fields_comma_separated string, values_to_fill ...any) {
	cols := strings.Split(fields_comma_separated, ",")
	if len(values_to_fill) != len(cols) {
		fmt.Println("error FillSelectedValues: len(values_to_fill) and len(struct fields) should be the same",len(values_to_fill),len(cols))
		return
	}
	rs := reflect.ValueOf(struct_to_fill)
	if rs.Kind() == reflect.Pointer {
		rs = reflect.ValueOf(struct_to_fill).Elem()
	}
	
	for i, col := range cols {
		var fieldToUpdate *reflect.Value
		if f := rs.FieldByName(SnakeCaseToTitle(col)); f.IsValid() && f.CanSet() {
			fieldToUpdate = &f
		} else if f := rs.FieldByName(col); f.IsValid() && f.CanSet() {
			// usually here
			fieldToUpdate = &f
		} else if f := rs.FieldByName(ToSnakeCase(col)); f.IsValid() && f.CanSet() {
			fieldToUpdate = &f
		}

		if fieldToUpdate.IsValid() {
			SetFieldValue(*fieldToUpdate,values_to_fill[i])
		}
	}
}

func SetFieldValue(fld reflect.Value, value any) {
	valueToSet := reflect.ValueOf(value)
	switch fld.Kind() {
	case valueToSet.Kind():
		fld.Set(valueToSet)
	case reflect.Ptr:
		unwrapped := fld.Elem()
		if !unwrapped.IsValid() {
			newUnwrapped := reflect.New(fld.Type().Elem())
			SetFieldValue(newUnwrapped,value)
			fld.Set(newUnwrapped)
			return
		}
		SetFieldValue(unwrapped,value)
	case reflect.Interface:
		unwrapped := fld.Elem()
		SetFieldValue(unwrapped,value)
	case reflect.Struct:
		switch v := value.(type) {
		case string:
			if strings.Contains(v, ":") || strings.Contains(v, "-") {
				l := len("2006-01-02T15:04")
				if strings.Contains(v[:l], "T") {
					if len(v) >= l {
						t, err := time.Parse("2006-01-02T15:04", v[:l])
						if err != nil {
							fld.Set(reflect.ValueOf(t))
						}
					}
				} else if len(v) >= len("2006-01-02 15:04:05") {
					t, err := time.Parse("2006-01-02 15:04:05", v[:len("2006-01-02 15:04:05")])
					if err == nil {
						fld.Set(reflect.ValueOf(t))
					}
				} else {
					fmt.Println("SetFieldValue Struct: doesn't match any case", v)
				}
			}
		case time.Time:
			fld.Set(valueToSet)
		case []any:
			// walk the fields
			for i := 0; i < fld.NumField(); i++ {
				SetFieldValue(fld.Field(i),v[i])
			}
		}	
	case reflect.String:
		switch valueToSet.Kind() {
		case reflect.String:
			fld.SetString(valueToSet.String())
		case reflect.Struct:
			fld.SetString(valueToSet.String())
		default:
			if valueToSet.IsValid() {
				fld.Set(valueToSet)
			} else {
				fmt.Println("value",valueToSet.Interface(),"is not valid")
			}
		}
	case reflect.Int:
		switch v := value.(type) {
		case int64:
			fld.SetInt(v)
		case string:
			if v, err := strconv.Atoi(v); err == nil {
				fld.SetInt(int64(v))
			}
		case int:
			fld.SetInt(int64(v))
		}
	case reflect.Int64:
		switch v := value.(type) {
		case int64:
			fld.SetInt(v)
		case string:
			if v, err := strconv.Atoi(v); err == nil {
				fld.SetInt(int64(v))
			}
		case []byte:
			if v, err := strconv.Atoi(string(v)); err != nil {
				fld.SetInt(int64(v))
			}
		case int:
			fld.SetInt(int64(v))
		}
	case reflect.Bool:
		switch valueToSet.Kind() {
		case reflect.Int:
			if value == 1 {
				fld.SetBool(true)
			}
		case reflect.Int64:
			if value == int64(1) {
				fld.SetBool(true)
			}
		case reflect.Uint64:
			if value == uint64(1) {
				fld.SetBool(true)
			}
		case reflect.String:
			if value == "1" {
				fld.SetBool(true)
			} else if value == "true" {
				fld.SetBool(true)
			}
		}
	case reflect.Uint:
		switch v := value.(type) {
		case uint:
			fld.SetUint(uint64(v))
		case uint64:
			fld.SetUint(v)
		case int64:
			fld.SetUint(uint64(v))
		case int:
			fld.SetUint(uint64(v))
		}
	case reflect.Uint64:
		switch v := value.(type) {
		case uint:
			fld.SetUint(uint64(v))
		case uint64:
			fld.SetUint(v)
		case int64:
			fld.SetUint(uint64(v))
		case int:
			fld.SetUint(uint64(v))
		}
	case reflect.Float64:
		if v, ok := value.(float64); ok {
			fld.SetFloat(v)
		}
	case reflect.Slice:
		targetType := fld.Type()
		typeName := targetType.String()
		if strings.HasPrefix(typeName, "[]") {		
			array := reflect.New(targetType).Elem()
			for _, v := range strings.Split(valueToSet.String(), ",") {
				array = reflect.Append(array, reflect.ValueOf(v))
			}
			fld.Set(array)
		}
	default:
		switch v := value.(type) {
		case []byte:
			fld.SetString(string(v))
		default:
			fmt.Println("setFieldValue: case not handled , unable to fill struct,field kind:", fld.Kind(), ",value to fill:", value)
		}
	}
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