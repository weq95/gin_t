package bind

import (
	"errors"
	"fmt"
	"mime/multipart"
	"net/url"
	"reflect"
	"strings"
)

const (
	pass      = "-"
	formTag   = "form"
	bindTag   = "bind"
	fileTag   = "file"
	headerTag = "header"
	uriTag    = "url"
)

var EmptyMultipartFormError = errors.New("nil *multipart.Form got")

type BindMethod func(reflect.Value, []string) error

type formBinder interface {
	BindForm(values url.Values, v any) error
}

type multipartFormBinder interface {
	BindMultipartForm(from *multipart.Form, v any) error
}

type URLValueBinder struct {
	TagName     string
	BindTagName string
	BindMethods map[string]BindMethod
}

// BindForm 解析参数绑定到结构体上
func (u URLValueBinder) BindForm(form url.Values, v any) error {
	var values = reflect.ValueOf(v)
	if values.Kind() != reflect.Ptr {
		return errors.New("pointer type required")
	}

	var value = values.Elem()
	var t = reflect.TypeOf(v).Elem()
	for i := 0; i < value.NumField(); i++ {
		var field = t.Field(i)
		//获取tag关联的值
		var tag, ok = field.Tag.Lookup(u.TagName)
		if !ok {
			//默认标签值
			tag = field.Name
		}

		var tags = strings.Split(tag, ",")
		var formKey = tags[0]
		var defFormValue []string
		if len(tags) != 1 {
			defFormValue = tags[1:]
		}

		//不进行解析的字段
		if formKey == pass {
			continue
		}

		var formValue, exist = form[formKey]
		if !exist && len(defFormValue) > 0 {
			if err := bind(value.Field(i), defFormValue); err != nil {
				return err
			}
		} else {
			if customBindTag, ok01 := field.Tag.Lookup(u.BindTagName); ok01 {
				if method := u.BindMethods[customBindTag]; method != nil {
					if err := method(value.Field(i), formValue); err != nil {
						return err
					}

					continue
				}

				return errors.New("no method named " + customBindTag)
			}
		}

		if err := bind(value.Field(i), formValue); err != nil {
			if len(defFormValue) > 0 { //尝试绑定默认值
				if err = bind(value.Field(i), defFormValue); err != nil {
					return err
				}
			}

			return err
		}
	}

	return nil
}

// AddBindMethod 添加处理绑定的业务函数
func (u *URLValueBinder) AddBindMethod(name string, method BindMethod) error {
	if _, ok := u.BindMethods[name]; ok {
		return fmt.Errorf("%s already exist", name)
	}

	if u.BindMethods == nil {
		u.BindMethods = make(map[string]BindMethod)
	}

	u.BindMethods[name] = method
	return nil
}

type HttpMultipartFormBinder struct {
	URLValueBinder
	FieldTag string
}

func (h *HttpMultipartFormBinder) BindMultipartForm(form *multipart.Form, v any) error {
	if form == nil {
		return EmptyMultipartFormError
	}

	var value = reflect.ValueOf(v)
	if value.Kind() != reflect.Ptr {
		return errors.New("pointer type required")
	}

	value = value.Elem()
	var t = reflect.TypeOf(v).Elem()
	for i := 0; i < value.NumField(); i++ {
		var field = t.Field(i)
		var tag, ok = field.Tag.Lookup(h.TagName)
		if !ok { //设置默认tag
			tag = field.Name
		}

		var tags = strings.Split(tag, ",")
		var formKey = tags[0]
		var defFormValue []string

		if len(tags) != 1 {
			defFormValue = tags[1:]
		}
		if formKey == pass {
			continue
		}

		if formValue, exits := form.Value[formKey]; exits {
			var customBindTag, ok01 = field.Tag.Lookup(h.BindTagName)
			if ok01 {
				if method := h.BindMethods[customBindTag]; method != nil {
					if err := method(value.Field(i), formValue); err != nil {
						return err
					}
				}

				continue
			}

			if err := bind(value.Field(i), formValue); err != nil {
				if len(defFormValue) > 0 {
					if err = bind(value.Field(i), defFormValue); err != nil {
						return err
					}
				}
			}

			continue
		} else if len(defFormValue) > 0 {
			if err := bind(value.Field(i), defFormValue); err != nil {
				return err
			}
		}

		switch value.Field(i).Interface().(type) {
		case *multipart.FileHeader, []*multipart.FileHeader:
			var fileTagVal, ok01 = field.Tag.Lookup(h.FieldTag)
			if ok01 {
				fileTagVal = field.Name
			}
			if fileTagVal == pass {
				break
			}

			if files, ok02 := form.File[fileTagVal]; ok02 {
				if err := bindFile(value.Field(i), files); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
