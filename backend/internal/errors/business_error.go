package errors

import "fmt"

// BusinessError 业务错误。
type BusinessError struct {
	Code       int
	HTTPStatus int
	Message    string
}

func (e *BusinessError) Error() string {
	return fmt.Sprintf("code=%d message=%s", e.Code, e.Message)
}

// New 构造业务错误（默认 HTTP 400）。
func New(code int, message string) *BusinessError {
	return &BusinessError{Code: code, HTTPStatus: 400, Message: message}
}

// NewWithStatus 构造带 HTTP 状态码的业务错误。
func NewWithStatus(code, status int, message string) *BusinessError {
	return &BusinessError{Code: code, HTTPStatus: status, Message: message}
}

// ValidationError 参数校验错误。
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation: " + e.Message
}

// MachineOfflineError 农机离线错误。
type MachineOfflineError struct {
	MachineCode string
}

func (e *MachineOfflineError) Error() string {
	return fmt.Sprintf("machine %s is offline", e.MachineCode)
}
