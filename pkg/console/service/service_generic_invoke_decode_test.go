/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package service

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestDecodeGenericInvokeArgsCountMismatch(t *testing.T) {
	parameterTypes := []string{"int", "int"}
	args := []json.RawMessage{
		json.RawMessage(`10`),
	}

	result, err := decodeGenericInvokeArgs(parameterTypes, args)
	if err == nil {
		t.Fatal("expected parameter count mismatch error")
	}
	if !strings.Contains(err.Error(), "count 2 does not match args count 1") {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestDecodeGenericInvokeArgsInvalidType(t *testing.T) {
	parameterTypes := []string{"int", "int"}
	args := []json.RawMessage{
		json.RawMessage(`"hello"`),
		json.RawMessage(`20`),
	}
	result, err := decodeGenericInvokeArgs(parameterTypes, args)

	if err == nil {
		t.Fatal("expected invalid argument type error")
	}
	if !strings.Contains(
		err.Error(),
		"invalid argument at index 0 for type int",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result, got %v", result)
	}
}

func TestDecodeGenericInvokeArgsStructuredError(t *testing.T) {
	parameterTypes := []string{"int", "int"}

	args := []json.RawMessage{
		json.RawMessage(`"hello"`),
		json.RawMessage(`20`),
	}

	_, err := decodeGenericInvokeArgs(parameterTypes, args)

	if err == nil {
		t.Fatal("expected decoding error")
	}

	var fieldErr *GenericInvokeArgumentError

	if !errors.As(err, &fieldErr) {
		t.Fatalf("expected GenericInvokeArgumentError, got %T: %v", err, err)
	}

	if fieldErr.Field != "args[0]" {
		t.Errorf("expected field args[0], got %s", fieldErr.Field)
	}

	if fieldErr.Expected != "int" {
		t.Errorf("expected type int, got %s", fieldErr.Expected)
	}

	if fieldErr.Actual != "string" {
		t.Errorf("expected actual string, got %s", fieldErr.Actual)
	}
}

func TestDecodeGenericInvokeArgsRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name     string
		javaType string
		raw      string
	}{
		{"string receives number", "java.lang.String", `42`},
		{"boolean receives object", "boolean", `{"a":1}`},
		{"primitive int receives null", "int", `null`},
		{"long overflow", "long", `9.223372036854776e18`},
		{"float overflow", "float", `1e40`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := decodeGenericInvokeArgs(
				[]string{tt.javaType},
				[]json.RawMessage{json.RawMessage(tt.raw)},
			)

			if err == nil {
				t.Fatalf(
					"expected error for Java type %s with JSON %s",
					tt.javaType,
					tt.raw,
				)
			}
		})
	}
}

func TestDecodeGenericInvokeArgsAcceptsValidBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		javaType string
		raw      string
		want     any
	}{
		{"string", "java.lang.String", `"hello"`, "hello"},
		{"boolean", "boolean", `true`, true},
		{"nullable wrapper", "java.lang.Integer", `null`, nil},
		{"max int64", "long", `9223372036854775807`, int64(9223372036854775807)},
		{"min int64", "long", `-9223372036854775808`, int64(-9223372036854775808)},
		{"scientific integer", "long", `1e3`, int64(1000)},
		{"float", "float", `1.5`, float32(1.5)},
		{"integer array", "int[]", `[1,2]`, []int32{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeGenericInvokeArgs(
				[]string{tt.javaType},
				[]json.RawMessage{json.RawMessage(tt.raw)},
			)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != 1 || !reflect.DeepEqual(got[0], tt.want) {
				t.Fatalf("want %#v, got %#v", tt.want, got)
			}
		})
	}
}
