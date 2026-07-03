package result

import "errors"

type Result[T any] struct {
	Value T
	Error error
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
	return r.Error == nil
}

func (r Result[T]) IsErr() bool {
	return r.Error != nil
}

func Equal[T comparable](r, other Result[T]) bool {
	if r.IsOk() && other.IsOk() {
		return r.Value == other.Value
	}
	if r.IsErr() && other.IsErr() {
		return r.Error == other.Error
	}
	return false
}
