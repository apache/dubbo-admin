package schema

import (
	"reflect"
)

type Schema interface {
	Validate(T reflect.Type) error
}
