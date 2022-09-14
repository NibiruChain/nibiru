package collections

import (
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("collections: not found")

func notFoundError(name string, key string) error {
	return fmt.Errorf("%w object '%s' with key %s", ErrNotFound, name, key)
}
