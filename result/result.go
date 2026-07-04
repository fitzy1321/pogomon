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
	return Result[T]{Error: err}
}

func ErrFromStr[T any](errMsg string) Result[T] {
	var t T
	return Result[T]{t, errors.New(errMsg)}
}

func (r *Result[T]) IsOk() bool {
	return r.Error == nil
}

func (r *Result[T]) IsErr() bool {
	return r.Error != nil
}

func Wrap[T any](v T, err error) Result[T] {
	return Result[T]{v, err}
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
