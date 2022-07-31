package cast

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/kamalshkeir/kago/core/utils/logger"
)

const (
	Uint8  = "uint8"
	Uint16 = "uint16"
	Uint32 = "uint32"
	Uint64 = "uint64"
	Uint   = "uint"

	Int8  = "int8"
	Int16 = "int16"
	Int32 = "int32"
	Int64 = "int64"
	Int   = "int"

	Float32 = "float32"
	Float64 = "float64"

	Bool = "bool"

	String = "string"
)

// FromTypeString cast a string value to the given type name.
func FromTypeString(value string, targetType string) (interface{}, error) {
	message := "cast: cannot cast `%v` to type `%v`"

	switch targetType {
	case Int:
		v, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return nil, fmt.Errorf(message, value, targetType)
		}
		return int(v), nil
	case Int8:
		v, err := strconv.ParseInt(value, 0, 8)
		if err != nil {
			return nil, err
		}
		return int8(v), nil
	case Int16:
		v, err := strconv.ParseInt(value, 0, 16)
		if err != nil {
			return nil, err
		}
		return int16(v), nil
	case Int32:
		v, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return int32(v), nil
	case Int64:
		v, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case Uint:
		v, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint(v), nil
	case Uint8:
		v, err := strconv.ParseUint(value, 0, 8)
		if err != nil {
			return nil, err
		}
		return uint8(v), nil
	case Uint16:
		v, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return nil, err
		}
		return uint16(v), nil
	case Uint32:
		v, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint32(v), nil
	case Uint64:
		v, err := strconv.ParseUint(value, 0, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case Bool:
		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return v, nil

	case Float32:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return float32(v), nil
	case Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case String:
		return value, nil
	}

	return nil, fmt.Errorf("cast: type %v is not supported", targetType)
}

// FromTypeReflect casts a string value to the given reflected type.
func FromTypeReflect(value string, targetType reflect.Type) (interface{}, error) {
	var typeName = targetType.String()

	if strings.HasPrefix(typeName, "[]") {
		itemType := typeName[2:]
		array := reflect.New(targetType).Elem()

		for _, v := range strings.Split(value, ",") {
			if item, err := FromTypeString(strings.Trim(v, " \n\r"), itemType); err != nil {
				return array.Interface(), err
			} else {
				array = reflect.Append(array, reflect.ValueOf(item))
			}
		}

		return array.Interface(), nil
	}

	return FromTypeString(value, typeName)
}


func FillStructFromValues(structure interface{},values ...any) error {
	inputType := reflect.TypeOf(structure)
	if inputType != nil {
		if inputType.Kind() == reflect.Ptr {
			if inputType.Elem().Kind() == reflect.Struct {
				return fillStructFromValues(reflect.ValueOf(structure).Elem(),values...)
			} else {
				return errors.New("env: element is not pointer to struct")
			}
		}
	}
	return errors.New("env: invalid structure")
}


func fillStructFromValues(s reflect.Value,values ...any) error {
	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)
		fType := s.Type()
		val := values[i]
		if reflect.TypeOf(val).Kind() == fType.Kind() {
			field.Set(reflect.ValueOf(val))
		} else if fType.Kind() == reflect.Struct {
			ptr := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
			ptr.Set(reflect.ValueOf(val))
		} else if fType.Kind() == reflect.Ptr {
			if !field.IsZero() && field.Elem().Type().Kind() == reflect.Struct {
				if err := fillStructFromValues(field.Elem(),val); err != nil {
					return err
				}
			}
		} else {
			v, err := FromTypeReflect(fmt.Sprintf("%v",val), fType)
			if err != nil {
				return err
			}
			ptr := reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
			ptr.Set(reflect.ValueOf(v))
		}
	}
	return nil
}


func FillStructOld[T comparable](struct_to_fill *T, values_to_fill ...any) {
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
			switch reflect.ValueOf(values_to_fill[i]).Kind() {
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
				fmt.Printf("%T\n", valueToSet.Interface())
			}
		case reflect.Float64:
			if v, ok := valueToSet.Interface().(float64); ok {
				field.SetFloat(v)
			}
		case reflect.SliceOf(reflect.TypeOf("")).Kind():
			field.SetString(strings.Join(valueToSet.Interface().([]string), ","))
		case reflect.Struct:
			switch v := valueToSet.Interface().(type) {
			case string:
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
			case time.Time:
				field.Set(reflect.ValueOf(v))
			default:
				logger.Error("field struct with value", v, "not handled")
			}
		default:
			field.Set(reflect.ValueOf(valueToSet.Interface()))
			logger.Error("type not handled for fieldType", field.Kind(), "value=", valueToSet.Interface())
		}
	}
}