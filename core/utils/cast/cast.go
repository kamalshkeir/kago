package cast

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// FromTypeString cast a string value to the given type name.
func FromTypeString(value string, targetType string) (interface{}, error) {
	message := "cast: cannot cast `%v` to type `%v`"

	switch targetType {
	case "int":
		v, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return nil, fmt.Errorf(message, value, targetType)
		}
		return int(v), nil
	case "int8":
		v, err := strconv.ParseInt(value, 0, 8)
		if err != nil {
			return nil, err
		}
		return int8(v), nil
	case "int16":
		v, err := strconv.ParseInt(value, 0, 16)
		if err != nil {
			return nil, err
		}
		return int16(v), nil
	case "int32":
		v, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return int32(v), nil
	case "int64":
		v, err := strconv.ParseInt(value, 0, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case "uint":
		v, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint(v), nil
	case "uint8":
		v, err := strconv.ParseUint(value, 0, 8)
		if err != nil {
			return nil, err
		}
		return uint8(v), nil
	case "uint16":
		v, err := strconv.ParseUint(value, 0, 16)
		if err != nil {
			return nil, err
		}
		return uint16(v), nil
	case "uint32":
		v, err := strconv.ParseUint(value, 0, 32)
		if err != nil {
			return nil, err
		}
		return uint32(v), nil
	case "uint64":
		v, err := strconv.ParseUint(value, 0, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case "bool":
		v, err := strconv.ParseBool(value)
		if err != nil {
			return nil, err
		}
		return v, nil

	case "float32":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return float32(v), nil
	case "float64":
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return nil, err
		}
		return v, nil

	case "string":
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
