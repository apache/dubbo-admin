package schema

import (
	"reflect"
)

type Status string

const (
	Continued Status = "CONTINUED"
	Finished  Status = "FINISHED"
	// Pending   Status = "PENDING"
)

type Schema interface {
	Validate(T reflect.Type) error
}
