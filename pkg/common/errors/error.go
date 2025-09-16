package errors

import (
	"errors"
	"fmt"
)

type AssertionError struct {
	msg string
}

func NewAssertionError(expected, actual interface{}) error {
	return &AssertionError{
		msg: fmt.Sprintf("type assertion error, expected:%v, actual:%v", expected, actual),
	}
}

func (e *AssertionError) Error() string {
	return e.msg
}

type MeshNotFoundError struct {
	Mesh string
}

func (m *MeshNotFoundError) Error() string {
	return fmt.Sprintf("mesh of name %s is not found", m.Mesh)
}

func MeshNotFound(meshName string) error {
	return &MeshNotFoundError{meshName}
}

func IsMeshNotFound(err error) bool {
	var meshNotFoundError *MeshNotFoundError
	ok := errors.As(err, &meshNotFoundError)
	return ok
}
