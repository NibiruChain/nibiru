package collections

import "errors"

var ErrNotFound = errors.New("not found")

func notFoundError(name string, key string) error {
	// TODO
	return ErrNotFound
}
