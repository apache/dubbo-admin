// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"reflect"
)

// kindOf returns the reflection Kind that represents the dynamic type of value.
// If value is a nil interface value, kindOf returns reflect.Invalid.
func kindOf(value interface{}) reflect.Kind {
	if value == nil {
		return reflect.Invalid
	}
	return reflect.TypeOf(value).Kind()
}

// IsSlice reports whether value is a slice type.
func IsSlice(value interface{}) bool {
	return kindOf(value) == reflect.Slice
}

// IsSliceInterfacePtr reports whether v is a slice ptr type.
func IsSliceInterfacePtr(v interface{}) bool {
	// Must use ValueOf because Elem().Elem() type resolves dynamically.
	vv := reflect.ValueOf(v)
	return vv.Kind() == reflect.Ptr && vv.Elem().Kind() == reflect.Interface && vv.Elem().Elem().Kind() == reflect.Slice
}

// IsMap reports whether value is a map type.
func IsMap(value interface{}) bool {
	return kindOf(value) == reflect.Map
}

// IsNilOrInvalidValue reports whether v is nil or reflect.Zero.
func IsNilOrInvalidValue(v reflect.Value) bool {
	return !v.IsValid() || (v.Kind() == reflect.Ptr && v.IsNil()) || IsValueNil(v.Interface())
}

// IsValueNil returns true if either value is nil, or has dynamic type {ptr,
// map, slice} with value nil.
func IsValueNil(value interface{}) bool {
	if value == nil {
		return true
	}
	switch kindOf(value) {
	case reflect.Slice, reflect.Ptr, reflect.Map:
		return reflect.ValueOf(value).IsNil()
	}
	return false
}

// IsValuePtr reports whether v is a ptr type.
func IsValuePtr(v reflect.Value) bool {
	return v.Kind() == reflect.Ptr
}

// IsValueStruct reports whether v is a struct type.
func IsValueStruct(v reflect.Value) bool {
	return v.Kind() == reflect.Struct
}

// IsValueMap reports whether v is a map type.
func IsValueMap(v reflect.Value) bool {
	return v.Kind() == reflect.Map
}

// IsValueSlice reports whether v is a slice type.
func IsValueSlice(v reflect.Value) bool {
	return v.Kind() == reflect.Slice
}

// IsValueScalar reports whether v is a scalar type.
func IsValueScalar(v reflect.Value) bool {
	if IsNilOrInvalidValue(v) {
		return false
	}
	if IsValuePtr(v) {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	return !IsValueStruct(v) && !IsValueMap(v) && !IsValueSlice(v)
}

// IsValueNilOrDefault returns true if either IsValueNil(value) or the default
// value for the type.
func IsValueNilOrDefault(value interface{}) bool {
	if IsValueNil(value) {
		return true
	}
	if !IsValueScalar(reflect.ValueOf(value)) {
		// Default value is nil for non-scalar types.
		return false
	}
	return value == reflect.New(reflect.TypeOf(value)).Elem().Interface()
}

// InsertIntoMap inserts value with key into parent which must be a map, map ptr, or interface to map.
func InsertIntoMap(parentMap interface{}, key interface{}, value interface{}) error {
	// scope.Debugf("InsertIntoMap key=%v, value=%v, map=\n%v", key, value, parentMap)
	v := reflect.ValueOf(parentMap)
	kv := reflect.ValueOf(key)
	vv := reflect.ValueOf(value)

	if v.Type().Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Type().Kind() == reflect.Interface {
		v = v.Elem()
	}

	if v.Type().Kind() != reflect.Map {
		// scope.Debugf("error %v", v.Type().Kind())
		return fmt.Errorf("insertIntoMap parent type is %T, must be map", parentMap)
	}

	v.SetMapIndex(kv, vv)

	return nil
}

// DeleteFromMap deletes an entry with the given key parent, which must be a map.
func DeleteFromMap(parentMap interface{}, key interface{}) error {
	// scope.Debugf("DeleteFromMap key=%s, parent:\n%v\n", key, parentMap)
	pv := reflect.ValueOf(parentMap)

	if !IsMap(parentMap) {
		return fmt.Errorf("deleteFromMap parent type is %T, must be map", parentMap)
	}
	pv.SetMapIndex(reflect.ValueOf(key), reflect.Value{})

	return nil
}

// DeleteFromSlicePtr deletes an entry at index from the parent, which must be a slice ptr.
func DeleteFromSlicePtr(parentSlice interface{}, index int) error {
	// scope.Debugf("DeleteFromSlicePtr index=%d, slice=\n%v", index, parentSlice)
	pv := reflect.ValueOf(parentSlice)

	if !IsSliceInterfacePtr(parentSlice) {
		return fmt.Errorf("deleteFromSlicePtr parent type is %T, must be *[]interface{}", parentSlice)
	}

	pvv := pv.Elem()
	if pvv.Kind() == reflect.Interface {
		pvv = pvv.Elem()
	}

	pv.Elem().Set(reflect.AppendSlice(pvv.Slice(0, index), pvv.Slice(index+1, pvv.Len())))

	return nil
}
