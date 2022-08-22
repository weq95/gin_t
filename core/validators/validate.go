package validators

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	validatorTag = "validate"
)

// Validator 验证器接口
type Validator interface {
	Validate(any) error
}

// ValidatorFunc 验证器回调函数
type ValidatorFunc func(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error

// ValidatorLibrary 定义验证器管理器
type ValidatorLibrary map[string]ValidatorFunc

// validatorLibrary验证器管理器
var validatorLibrary ValidatorLibrary = make(map[string]ValidatorFunc, 0)

func (validate ValidatorLibrary) Validate(value any) error {
	var t = reflect.TypeOf(value)
	for t.Kind() == reflect.Ptr { //解引用(去指针化)
		t = t.Elem()
	}

	var v = reflect.ValueOf(value)
	for reflect.Ptr == v.Kind() { //解引用(去指针化)
		v = v.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		var tags, ok = t.Field(i).Tag.Lookup(validatorTag)
		if !ok {
			continue
		}

		// 根据分号分隔
		// required(m=姓名不能为空);max_length(m=姓名长度不能大于10,value=10)
		// required(m=姓名不能为空)  max_length(m=姓名长度不能大于10,value=10)
		for _, validateName := range strings.Split(tags, ";") {
			var matchList = validateParamRegexp.FindAllStringSubmatch(validateName, -1)
			if len(matchList) == 0 {
				continue
			}

			var result = matchList[0]
			var key = result[1]
			var validator, ok = validate[key]
			if !ok {
				return fmt.Errorf("%s validator does not exits", key)
			}

			var text = result[2]
			var list = strings.Split(text, ",")
			if len(list) > 2 {
				list[1] = strings.TrimPrefix(text, list[0]+",")
				list = list[:2]
			}

			var param Param
			for idx, val := range list {
				if strings.HasPrefix(val, " ") {
					list[idx] = strings.TrimPrefix(val, " ")
				}

				var item = strings.Split(list[idx], "=")
				if len(item) != 2 {
					return fmt.Errorf("%s syntax error", list[idx])
				}

				switch item[0] {
				case "m", "message":
					param.Message = item[1]
				case "v", "value":
					param.Value = item[1]
				default:
					return fmt.Errorf("unexpect param got %s", item[0])
				}
			}

			if err := validator(t, v, i, param); err != nil {
				return NewValidationError(err, t.Field(i).Name, key)
			}
		}
	}

	return nil
}

// RegisterValidator 注册验证器
func RegisterValidator(key string, val ValidatorFunc) error {
	if _, ok := validatorLibrary[key]; ok {
		return fmt.Errorf("key %s has already exits", key)
	}

	validatorLibrary[key] = val
	return nil
}

// isValid 检测值是否位空
func isValid(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.String: //字符串类型
		return value.String() != ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64: //number类型含负数
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64: //number 非负整数
		fallthrough
	case reflect.Float32, reflect.Float64: //number float类型
		return !value.IsZero()
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan: //其他类数组复合类型
		return value.Len() > 0
	case reflect.Struct: //结构体, 需要两项检测
		return !reflect.DeepEqual(value.Interface(), reflect.New(value.Type()).Elem().Interface())
	case reflect.Ptr, reflect.Interface: //指针类型, interface的指针类型
		return !value.IsNil()
	default:
		return true
	}
}

// Required 必填字段
func Required(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	//获取当前字段
	var field = valueOf.Field(index)
	if field.Kind() == reflect.String {
		var except, err = strconv.Atoi(param.Value)
		if err != nil {
			return err
		}

		if utf8.RuneCountInString(field.String()) >= except {
			return errors.New(param.Message) //返回用户自定义错误信息
		}
	}

	return nil
}
func MaxLength(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	var field = valueOf.Field(index)
	if field.Kind() == reflect.String {
		var except, err = strconv.Atoi(param.Value)
		if err != nil {
			return err
		}

		if utf8.RuneCountInString(field.String()) >= except {
			return errors.New(param.Message)
		}
	}

	return nil
}

// MinLength 字段最小长度
func MinLength(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	var field = valueOf.Field(index)
	if field.Kind() == reflect.String {
		var except, err = strconv.Atoi(param.Value)
		if err != nil {
			return err
		}

		if utf8.RuneCountInString(field.String()) <= except {
			return errors.New(param.Message)
		}
	}

	return nil
}

// isGt 数值比较大小
func isGt(field reflect.Value, v string) (bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		var value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Int() > value, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var value, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Uint() > value, nil

	case reflect.Float32, reflect.Float64:
		var value, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}

		return field.Float() > value, nil

	default:
		return false, unsupportedError
	}
}

func GreaterThan(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var ok, err = isGt(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New(param.Message)
	}

	return nil
}

// isLt 数值比较大小
func isLt(field reflect.Value, v string) (bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		var value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Int() < value, err

	case reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64:
		var value, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Uint() < value, err

	case reflect.Float32, reflect.Float64:
		var value, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}

		return field.Float() < value, nil
	default:
		return false, unsupportedError
	}
}

func LowerThan(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var ok, err = isLt(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}

	if !ok {
		return errors.New(param.Message)
	}

	return err
}

func isGe(field reflect.Value, v string) (bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		var value, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Int() >= value, err
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var value, err = strconv.ParseUint(v, 10, 64)
		if err != nil {
			return false, err
		}

		return field.Uint() >= value, err
	case reflect.Float32, reflect.Float64:
		var value, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}

		return field.Float() >= value, err
	default:
		return false, unsupportedError
	}
}

func GreaterEqual(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}
	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var ok, err = isGe(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(param.Value)
	}

	return err
}

func isLe(field reflect.Value, v string) (bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Int() <= value, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Uint() <= value, nil
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}
		return field.Float() <= value, nil
	}
	return false, unsupportedError
}

func LowerEqual(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	ok, err := isLe(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(param.Message)
	}
	return nil
}

func isEqual(field reflect.Value, v string) (bool, error) {
	switch field.Kind() {
	case reflect.String:
		return field.String() == v, nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Int() == value, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Uint() == value, nil
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return false, err
		}
		return field.Float() == value, nil
	case reflect.Bool:
		value, err := strconv.ParseBool(v)
		if err != nil {
			return false, err
		}
		return field.Bool() == value, nil
	}
	return false, unsupportedError
}

func Equal(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	ok, err := isEqual(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(param.Message)
	}
	return nil
}

func EqualField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	f := valueOf.FieldByName(param.Value)
	if !f.IsValid() {
		return fmt.Errorf("no field named %s", param.Value)
	}
	if !reflect.DeepEqual(valueOf.Field(index).Interface(), f.Interface()) {
		return errors.New(param.Message)
	}
	return nil
}

func Method(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var method = valueOf.MethodByName(param.Value)
	if method.IsValid() {
		return fmt.Errorf("no methods name %s", param.Value)
	}

	if fn, ok := method.Interface().(ValidatorFunc); ok {
		return fn(typeOf, valueOf, index, param)
	}

	return fmt.Errorf("method %s must be `validate.ValidatorFunc` type ", param.Value)
}

func Regexp(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}
	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var field = valueOf.Field(index)
	var reg = regexp.MustCompile(param.Value)
	if field.Kind() != reflect.String {
		return fmt.Errorf("Regexp only support string type")
	}

	if !reg.MatchString(field.String()) {
		return errors.New(param.Message)
	}

	return nil
}

func Email(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}

	var field = valueOf.Field(index)
	if field.Kind() != reflect.String {
		return errors.New("Email only support string type")
	}

	if !emailRegex.MatchString(field.String()) {
		return errors.New(param.Message)
	}
	return nil
}

func UUID(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	field := valueOf.Field(index)
	if field.Kind() != reflect.String {
		return errors.New("UUID only support string type")
	}
	if !uuidRegexp.MatchString(field.String()) {
		return errors.New(param.Message)
	}
	return nil
}

func Phone(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	field := valueOf.Field(index)
	if field.Kind() != reflect.String {
		return errors.New("Phone only support string type")
	}
	if !phoneRegexp.MatchString(field.String()) {
		return errors.New(param.Message)
	}
	return nil
}

// isBetween 范围判断 min<=T<=max
func isBetween(field reflect.Value, min, max string) (bool, error) {
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		mi, err := strconv.ParseInt(min, 10, 64)
		if err != nil {
			return false, err
		}
		ma, err := strconv.ParseInt(max, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Int() >= mi && field.Int() <= ma, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		mi, err := strconv.ParseUint(min, 10, 64)
		if err != nil {
			return false, err
		}
		ma, err := strconv.ParseUint(max, 10, 64)
		if err != nil {
			return false, err
		}
		return field.Uint() >= mi && field.Uint() <= ma, nil
	case reflect.Float32, reflect.Float64:
		mi, err := strconv.ParseFloat(min, 64)
		if err != nil {
			return false, err
		}
		ma, err := strconv.ParseFloat(max, 64)
		if err != nil {
			return false, err
		}
		return field.Float() >= mi && field.Float() <= ma, nil
	default:
		return false, nil
	}
}

// Between 在 T 范围内
func Between(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	value := strings.Split(param.Value, ",")
	if len(value) != 2 {
		return fmt.Errorf("%s syntax error", param.Value)
	}
	min := strings.ReplaceAll(value[0], " ", "")
	max := strings.ReplaceAll(value[1], " ", "")
	ok, err := isBetween(valueOf.Field(index), min, max)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(param.Message)
	}
	return nil
}

// NotBetween 不在 T 范围内
func NotBetween(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	value := strings.Split(param.Value, ",")
	if len(value) != 2 {
		return fmt.Errorf("%s syntax error", param.Value)
	}
	min := strings.ReplaceAll(value[0], " ", "")
	max := strings.ReplaceAll(value[1], " ", "")
	ok, err := isBetween(valueOf.Field(index), min, max)
	if err != nil {
		return err
	}
	if ok {
		return errors.New(param.Message)
	}
	return nil
}
func NotEqual(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	ok, err := isEqual(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if ok {
		return errors.New(param.Message)
	}
	return nil
}

func NotEqualField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	f := valueOf.FieldByName(param.Value)
	if !f.IsValid() {
		return fmt.Errorf("no field named %s", param.Value)
	}
	if reflect.DeepEqual(valueOf.Field(index).Interface(), f.Interface()) {
		return errors.New(param.Message)
	}
	return nil
}

// Url 超链接判断
func Url(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	field := valueOf.Field(index)
	if field.Kind() != reflect.String {
		return errors.New("Url only support string type")
	}
	if !urlRegexp.MatchString(field.String()) {
		return errors.New(param.Message)
	}
	return nil
}

func isContains(field reflect.Value, value string) (bool, error) {
	var arrayStr = strings.Split(value, ",")
	switch field.Kind() {
	case reflect.String:
		var v = field.String()
		for _, str := range arrayStr {
			if str == v {
				return true, nil
			}
		}

		return false, nil
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		var v = field.Int()
		for _, item := range arrayStr {
			var num, err = strconv.ParseInt(strings.ReplaceAll(item, " ", ""), 10, 64)
			if err != nil {
				return false, err
			}
			if v == num {
				return true, nil
			}
		}
		return false, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var v = field.Uint()
		for _, item := range arrayStr {
			var num, err = strconv.ParseUint(strings.ReplaceAll(item, " ", ""), 10, 64)
			if err != nil {
				return false, err
			}
			if v == num {
				return true, nil
			}
		}
	case reflect.Float32, reflect.Float64:
		var v = field.Float()
		for _, item := range arrayStr {
			var num, err = strconv.ParseFloat(strings.ReplaceAll(item, " ", ""), 64)
			if err != nil {
				return false, err
			}
			if num == v {
				return true, nil
			}
		}

		return false, nil

	}
	return false, unsupportedError
}

func Contains(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	ok, err := isContains(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New(param.Message)
	}
	return nil
}

func NotContains(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	ok, err := isContains(valueOf.Field(index), param.Value)
	if err != nil {
		return err
	}
	if ok {
		return errors.New(param.Message)
	}
	return nil
}

func DateFormat(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}
	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var field = valueOf.Field(index)
	if field.Kind() != reflect.String {
		return errors.New("DateFormat only support string type")
	}

	if _, err := time.Parse(param.Value, field.String()); err != nil {
		return errors.New(param.Message)
	}

	return nil
}

func GreatThanField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}
	if len(param.Value) == 0 {
		return messageParamRequiredError
	}

	var field = valueOf.Field(index)
	var anotherField = valueOf.FieldByName(param.Value)
	if !anotherField.IsValid() {
		return errors.New("no field name " + param.Value)
	}
	if field.Kind() != anotherField.Kind() {
		return fmt.Errorf("%s and %s are different kind", typeOf.Field(index).Name, param.Value)
	}

	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		if field.Int() > anotherField.Int() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() > anotherField.Uint() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Float32, reflect.Float64:
		if field.Float() > anotherField.Float() {
			return nil
		}
		return errors.New(param.Message)
	}

	return errors.New("unsupported type")
}
func GreatEqualField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	field := valueOf.Field(index)
	anotherField := valueOf.FieldByName(param.Value)
	if !anotherField.IsValid() {
		return errors.New("no field name " + param.Value)
	}
	if field.Kind() != anotherField.Kind() {
		return fmt.Errorf("%s and %s are different kind", typeOf.Field(index).Name, param.Value)
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		if field.Int() >= anotherField.Int() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() >= anotherField.Uint() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Float32, reflect.Float64:
		if field.Float() >= anotherField.Float() {
			return nil
		}
		return errors.New(param.Message)
	}
	return errors.New("unsupported type")
}

func LowerThanField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	field := valueOf.Field(index)
	anotherField := valueOf.FieldByName(param.Value)
	if !anotherField.IsValid() {
		return errors.New("no field name " + param.Value)
	}
	if field.Kind() != anotherField.Kind() {
		return fmt.Errorf("%s and %s are different kind", typeOf.Field(index).Name, param.Value)
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		if field.Int() < anotherField.Int() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() < anotherField.Uint() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Float32, reflect.Float64:
		if field.Float() < anotherField.Float() {
			return nil
		}
		return errors.New(param.Message)
	}
	return errors.New("unsupported type")
}
func LowerEqualField(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if param.Message == "" {
		return messageParamRequiredError
	}
	if param.Value == "" {
		return valueParamRequiredError
	}
	field := valueOf.Field(index)
	anotherField := valueOf.FieldByName(param.Value)
	if !anotherField.IsValid() {
		return errors.New("no field name " + param.Value)
	}
	if field.Kind() != anotherField.Kind() {
		return fmt.Errorf("%s and %s are different kind", typeOf.Field(index).Name, param.Value)
	}
	switch field.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64:
		if field.Int() <= anotherField.Int() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if field.Uint() <= anotherField.Uint() {
			return nil
		}
		return errors.New(param.Message)
	case reflect.Float32, reflect.Float64:
		if field.Float() <= anotherField.Float() {
			return nil
		}
		return errors.New(param.Message)
	}
	return errors.New("unsupported type")
}

func Round(typeOf reflect.Type, valueOf reflect.Value, index int, param Param) error {
	if len(param.Message) == 0 {
		return messageParamRequiredError
	}
	if len(param.Value) == 0 {
		return valueParamRequiredError
	}

	var field = valueOf.Field(index)
	var v, err = strconv.Atoi(param.Value)
	if err != nil {
		return err
	}

	var text string
	switch field.Kind() {
	case reflect.Float32:
		text = strconv.FormatFloat(field.Float(), 'f', -1, 32)
	case reflect.Float64:
		text = strconv.FormatFloat(field.Float(), 'f', -1, 64)
	default:
		return errors.New("round support float type only")
	}

	var s = strings.Split(text, ".")
	if len(s) == 2 && len(s[1]) > v {
		return errors.New(param.Message)
	}

	return nil
}

func init() {
	_ = RegisterValidator("required", Required)
	_ = RegisterValidator("max_length", MaxLength)
	_ = RegisterValidator("min_length", MinLength)
	_ = RegisterValidator("gt", GreaterThan)
	_ = RegisterValidator("lt", LowerThan)
	_ = RegisterValidator("ge", GreaterEqual)
	_ = RegisterValidator("le", LowerEqual)
	_ = RegisterValidator("equal", Equal)
	_ = RegisterValidator("eq", Equal)
	_ = RegisterValidator("equal_field", EqualField)
	_ = RegisterValidator("ef", EqualField)
	_ = RegisterValidator("method", Method)
	_ = RegisterValidator("regexp", Regexp)
	_ = RegisterValidator("email", Email)
	_ = RegisterValidator("uuid", UUID)
	_ = RegisterValidator("phone", Phone)
	_ = RegisterValidator("between", Between)
	_ = RegisterValidator("not_between", NotBetween)
	_ = RegisterValidator("not_equal", NotEqual)
	_ = RegisterValidator("ne", NotEqual)
	_ = RegisterValidator("not_equal_field", NotEqualField)
	_ = RegisterValidator("nef", NotEqualField)
	_ = RegisterValidator("url", Url)
	_ = RegisterValidator("contains", Contains)
	_ = RegisterValidator("not_contains", NotContains)
	_ = RegisterValidator("date_format", DateFormat)
	_ = RegisterValidator("great_than_field", GreatThanField)
	_ = RegisterValidator("gtf", GreatThanField)
	_ = RegisterValidator("lower_than_field", LowerThanField)
	_ = RegisterValidator("ltf", LowerThanField)
	_ = RegisterValidator("great_equal_field", GreatEqualField)
	_ = RegisterValidator("gef", GreatEqualField)
	_ = RegisterValidator("lower_equal_field", LowerEqualField)
	_ = RegisterValidator("lef", LowerEqualField)
	_ = RegisterValidator("round", Round)
}
