package validators

import "regexp"

// 必须包含validate 或 params字段
var validateParamRegexp = regexp.MustCompile(`(?P<validator>\w+)\((?P<params>.*)\)`)

type Param struct {
	Message string //用户自定义错误提示
	Value   string
}

func Validate(value any) error {
	return validatorLibrary.Validate(value)
}

type Default struct{}

func (d Default) Validate(validate any) error {
	return Validate(validate)
}

var _ Validator = Default{}
