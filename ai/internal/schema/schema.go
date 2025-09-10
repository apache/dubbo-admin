package schema

import (
	"encoding/json"
	"reflect"
)

type Schema interface {
	Validate(T reflect.Type) error
}

type StringSchema struct {
	str string
}

type ArraySchema[T any] struct {
	arr []T
}

type ObjSchema[T any] struct {
	obj T
}

func NewArraySchema[T any](arr []T) *ArraySchema[T] {
	return &ArraySchema[T]{arr: arr}
}

func NewStringSchema(str string) *StringSchema {
	return &StringSchema{str: str}
}

func NewObjSchema[T any](obj T) *ObjSchema[T] {
	// 运行时检查，确保类型不是指针
	// if reflect.TypeOf(obj).Kind() == reflect.Pointer {
	// 	panic(fmt.Sprintf("NewObjSchema does not accept pointer types, got %T", obj))
	// }
	return &ObjSchema[T]{obj: obj}
}

func (ss *StringSchema) Spec() string {
	return ss.str
}

func (as *ArraySchema[T]) Spec() []T {
	return as.arr
}

func (os *ObjSchema[T]) Spec() T {
	return os.obj
}

func (ss *StringSchema) Desc() string {
	return ss.str
}

func (as *ArraySchema[T]) Desc() string {
	data, err := json.Marshal(as.arr)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func (os *ObjSchema[T]) Desc() string {
	data, err := json.Marshal(os.obj)
	if err != nil {
		panic(err)
	}
	return string(data)
}
