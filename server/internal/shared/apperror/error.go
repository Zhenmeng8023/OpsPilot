package apperror

import "fmt"

type Error struct {
	HTTPStatus int
	Code       int
	Message    string
	Cause      error
}

func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func New(httpStatus, code int, message string) *Error {
	return &Error{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

func Wrap(httpStatus, code int, message string, cause error) *Error {
	return &Error{
		HTTPStatus: httpStatus,
		Code:       code,
		Message:    message,
		Cause:      cause,
	}
}
