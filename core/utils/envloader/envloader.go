package envloader

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"
	"unsafe"

	"github.com/kamalshkeir/kago/core/utils/cast"
	"github.com/kamalshkeir/kago/core/utils/logger"
)



func Load(envFiles ...string) {
	var wg sync.WaitGroup
	if len(envFiles) == 0 {
		Load(".env")
	}
	wg.Add(len(envFiles))
	for _,env := range envFiles {
		go func(env string) {
			defer wg.Done()
			f,err := os.OpenFile(env,os.O_RDONLY, os.ModePerm)
			if err != nil {
				fmt.Println(err)
				return
			}
			defer f.Close()
			r := bufio.NewScanner(f)
			
			for r.Scan() {
				sp := strings.Split(r.Text(),"=")
				if len(sp) != 2 || r.Text()[0] == '#'{
					continue
				}
				sp[0]=strings.TrimSpace(sp[0])
				sp[1]=strings.TrimSpace(sp[1])
				err := os.Setenv(sp[0],sp[1])
				logger.CheckError(err)
			}
		}(env)
	}
	wg.Wait()
}



// FillStructFromEnv fill the struct from env
func FillStruct(structure interface{}) error {
	inputType := reflect.TypeOf(structure)
	if inputType != nil {
		if inputType.Kind() == reflect.Ptr {
			if inputType.Elem().Kind() == reflect.Struct {
				return fillStructFromEnv(reflect.ValueOf(structure).Elem())
			} else {
				return errors.New("env: element is not pointer to struct")
			}
		}
	}
	return errors.New("env: invalid structure")
}

// fillStructFromEnv sets a reflected struct fields with the equivalent OS environment variables.
func fillStructFromEnv(s reflect.Value) error {
	for i := 0; i < s.NumField(); i++ {
		if t, exist := s.Type().Field(i).Tag.Lookup("env"); exist {
			// tag exist
			tag := t
			defau := ""
			if strings.Contains(t,"|") {
				sp := strings.Split(t,"|")
				if len(sp) == 2 {
					tag=sp[0] 
					defau=sp[1]
					if defau == "" {defau="|"}
				}
			} 

			if osv := os.Getenv(strings.TrimSpace(tag)); osv != "" {
				v, err := cast.FromTypeReflect(osv, s.Type().Field(i).Type)
				if err != nil {
					return fmt.Errorf("env: cannot set `%v` field; err: %v", s.Type().Field(i).Name, err)
				}

				ptr := reflect.NewAt(s.Field(i).Type(), unsafe.Pointer(s.Field(i).UnsafeAddr())).Elem()
				ptr.Set(reflect.ValueOf(v))
			} else {
				if defau != "|" && defau != "" {
					defau=strings.TrimSpace(defau)
					v, err := cast.FromTypeReflect(defau, s.Type().Field(i).Type)
					if err != nil {
						return fmt.Errorf("env: cannot set `%v` field; err: %v", s.Type().Field(i).Name, err)
					}
					ptr := reflect.NewAt(s.Field(i).Type(), unsafe.Pointer(s.Field(i).UnsafeAddr())).Elem()
					ptr.Set(reflect.ValueOf(v))
				}
			}
		} else if s.Type().Field(i).Type.Kind() == reflect.Struct {
			// tag not exist, check if struct
			if err := fillStructFromEnv(s.Field(i)); err != nil {
				return err
			}
		} else if s.Type().Field(i).Type.Kind() == reflect.Ptr {
			if !s.Field(i).IsZero() && s.Field(i).Elem().Type().Kind() == reflect.Struct {
				if err := fillStructFromEnv(s.Field(i).Elem()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}