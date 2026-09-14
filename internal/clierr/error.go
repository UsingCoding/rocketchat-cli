package clierr

import "errors"

type Code int

const (
	CodeGeneric    Code = 1
	CodeUsage      Code = 2
	CodeAuth       Code = 3
	CodeNotFound   Code = 4
	CodePermission Code = 5
	CodeConflict   Code = 6
	CodeNetwork    Code = 7
)

type Error struct {
	Code Code
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func New(code Code, err error) error {
	if err == nil {
		return nil
	}
	return &Error{Code: code, Err: err}
}

func ExitCode(err error) int {
	var e *Error
	if errors.As(err, &e) {
		return int(e.Code)
	}
	return int(CodeGeneric)
}
