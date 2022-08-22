package bind

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"reflect"
	"strconv"
	"time"
)

func bindSingle(field reflect.Value, formValue string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(formValue)
	case reflect.Int:
		return bindInt(field, formValue, 0)
	case reflect.Int8:
		return bindInt(field, formValue, 8)
	case reflect.Int16:
		return bindInt(field, formValue, 16)
	case reflect.Int32:
		return bindInt(field, formValue, 32)
	case reflect.Int64:
		return bindInt(field, formValue, 64)
	case reflect.Uint:
		return bindUint(field, formValue, 0)
	case reflect.Uint8:
		return bindUint(field, formValue, 8)
	case reflect.Uint16:
		return bindUint(field, formValue, 16)
	case reflect.Uint32:
		return bindUint(field, formValue, 32)
	case reflect.Uint64:
		return bindUint(field, formValue, 64)
	case reflect.Float32:
		return bindFloat(field, formValue, 32)
	case reflect.Float64:
		return bindFloat(field, formValue, 64)
	case reflect.Bool:
		return bindBool(field, formValue)
	case reflect.Struct:
		return json.Unmarshal([]byte(formValue), field.Addr().Interface())
	case reflect.Map:
		return json.Unmarshal([]byte(formValue), field.Addr().Interface())
	case reflect.Ptr:
		return json.Unmarshal([]byte(formValue), field.Addr().Interface())
	default:
		return errors.New("unknown type got")
	}
	return nil
}

func bind(field reflect.Value, formValues []string) error {
	switch field.Kind() {
	case reflect.Array:
		return bindArray(field, formValues)
	case reflect.Slice:
		return bindSlice(field, formValues)
	default:
		if len(formValues) > 0 {
			return bindSingle(field, formValues[0])
		}
	}

	return nil
}

func bindInt(field reflect.Value, formValue string, bitSize int) error {
	if formValue == "" {
		formValue = "0"
	}
	var v, err = strconv.ParseInt(formValue, 10, bitSize)
	if err != nil {
		return err
	}

	field.SetInt(v)
	return nil
}

func bindFloat(field reflect.Value, formValue string, bitSize int) error {
	if formValue == "" {
		formValue = "0.0"
	}

	var v, err = strconv.ParseFloat(formValue, bitSize)
	if err != nil {
		return err
	}

	field.SetFloat(v)
	return nil
}

func bindUint(field reflect.Value, formValue string, bitSize int) error {
	if formValue == "" {
		formValue = "0"
	}
	v, err := strconv.ParseUint(formValue, 10, bitSize)
	if err != nil {
		return err
	}
	field.SetUint(v)
	return nil
}

func bindBool(field reflect.Value, formValue string) error {
	if formValue == "" {
		formValue = "0"
	}
	v, err := strconv.ParseBool(formValue)
	if err != nil {
		return err
	}
	field.SetBool(v)
	return nil
}

// bindArray 绑定数组
func bindArray(field reflect.Value, formValues []string) error {
	var err error

	for i, value := range formValues {
		if err = bindSingle(field.Index(i), value); err != nil {
			return err
		}
	}

	return err
}

// bindSlice 绑定切片
func bindSlice(field reflect.Value, formValues []string) error {
	var length = len(formValues)
	var slice = reflect.MakeSlice(field.Type(), length, length)
	if err := bindArray(slice, formValues); err != nil {
		return err
	}

	field.Set(slice)
	return nil
}

// Int64TimeBinder 绑定整型字符串时间
func Int64TimeBinder() BindMethod {
	return func(value reflect.Value, strings []string) error {
		var str string
		if len(strings) > 0 {
			str = strings[0]
		}

		switch value.Interface().(type) {
		case time.Time:
			var i, err = strconv.ParseInt(str, 10, 64)
			if err != nil {
				return err
			}

			var t = time.Unix(i, 0)
			value.Set(reflect.ValueOf(t))
			return err
		}

		return errors.New("time.Time type required")
	}
}

func FormatTimeBinder(format string) BindMethod {
	return func(value reflect.Value, strings []string) error {
		var str string
		if len(strings) > 0 {
			str = strings[0]
		}

		switch value.Interface().(type) {
		case time.Time:
			var t, err = time.Parse(format, str)
			if err != nil {
				return err
			}

			value.Set(reflect.ValueOf(t))
			return nil
		}

		return errors.New("time.Time type required")
	}
}

// bindFile 绑定文件
func bindFile(field reflect.Value, files []*multipart.FileHeader) error {
	for i, file := range files { //过滤空文件
		if file == nil {
			files = append(files[:i], files[i+1:]...)
		}
	}

	switch field.Interface().(type) {
	case *multipart.FileHeader:
		if len(files) == 0 {
			return errors.New("no such file found")
		}

		//这里应该其实拿的地址
		field.Set(reflect.ValueOf(files[0]).Elem().Addr())

	case []*multipart.FileHeader:
		var length = len(files)
		var v = reflect.ValueOf(files)
		var slice = reflect.MakeSlice(v.Type(), length, length)
		for i, file := range files {
			slice.Index(i).Set(reflect.ValueOf(file).Elem().Addr())
		}
		field.Set(slice)

	default:
		return errors.New("unknown type got")
	}

	return nil
}
