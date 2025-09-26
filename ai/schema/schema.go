package schema

import (
	"reflect"

	"github.com/firebase/genkit/go/ai"
)

type Status string

const (
	Continued Status = "CONTINUED"
	Finished  Status = "FINISHED"
	// Pending   Status = "PENDING"
)

type Schema interface {
	Validate(T reflect.Type) error
	Usage() *ai.GenerationUsage
}
