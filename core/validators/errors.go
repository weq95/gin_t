package validators

import "errors"

type ValidationError struct {
	error
	FieldName string `json:"field_name"`
	Rule      string `json:"rule"`
}

// Unwrap 获取原始错误
func (e *ValidationError) Unwrap() error {
	return e.error
}

func NewValidationError(err error, fieldName string, rule string) *ValidationError {
	return &ValidationError{
		error:     err,
		FieldName: fieldName,
		Rule:      rule,
	}
}

var (
	// 备注必填
	messageParamRequiredError = errors.New("param error, param `message` is required")
	// 值必填
	valueParamRequiredError = errors.New("param error, param `value` is required")
	// 其他错误
	unsupportedError = errors.New("unsupported error")
)

func IsValidationError(err error) bool {
	var _, ok = err.(ValidationError)

	return ok
}
