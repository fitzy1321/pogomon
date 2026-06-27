package result

import "errors"

type Result[T any] struct {
	Value T
	err   error
}

func Ok[T any](value T) Result[T] {
	return Result[T]{value, nil}
}

func Err[T any](err error) Result[T] {
	var t T
	return Result[T]{t, err}
}

func ErrFromStr[T any](errMsg string) Result[T] {
	var t T
	return Result[T]{t, errors.New(errMsg)}
}

func (r Result[T]) IsOk() bool {
	return r.err == nil
}

func (r Result[T]) IsErr() bool {
	return r.err != nil
}

func (r Result[T]) GetError() error {
	return r.err
}

// Error Interface
func (r *Result[T]) Error() string {
	return r.err.Error()
}

func Equal[T comparable](r, other Result[T]) bool {
	if r.IsOk() && other.IsOk() {
		return r.Value == other.Value
	}
	if r.IsErr() && other.IsErr() {
		return r.Error() == other.Error()
	}
	return false
}
