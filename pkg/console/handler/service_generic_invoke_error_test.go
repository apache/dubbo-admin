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
package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/service"
)

func TestWriteGenericInvokeErrorStructuredType(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	err := &service.GenericInvokeArgumentError{
		Kind:     service.ArgumentErrorType,
		Field:    "args[0]",
		Expected: "int",
		Actual:   "string",
	}

	writeGenericInvokeError(ctx, err)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP status: %d", recorder.Code)
	}
	var response struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Field    string `json:"field"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
	}

	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("invalid response JSON: %v", err)
	}
	if response.Message != "invalid argument type" {
		t.Errorf(
			"expected invalid argument type, got %s",
			response.Message,
		)
	}

	if response.Code != "InvalidArgument" {
		t.Errorf("unexpected code: %s", response.Code)
	}

	if response.Field != "args[0]" {
		t.Errorf("unexpected field: %s", response.Field)
	}

	if response.Expected != "int" {
		t.Errorf("unexpected expected type: %s", response.Expected)
	}

	if response.Actual != "string" {
		t.Errorf("unexpected actual type: %s", response.Actual)
	}
}

func TestWriteGenericInvokeErrorWrappedError(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)

	fieldErr := &service.GenericInvokeArgumentError{
		Kind:     service.ArgumentErrorType,
		Field:    "args[0]",
		Expected: "int",
		Actual:   "string",
	}

	wrappedErr := fmt.Errorf(
		"%w: %w",
		bizerror.New(
			bizerror.InvalidArgument,
			"invalid generic invoke arguments",
		),
		fieldErr,
	)
	writeGenericInvokeError(ctx, wrappedErr)

	var response struct {
		Code     string `json:"code"`
		Field    string `json:"field"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
		Message  string `json:"message"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if response.Message != "invalid argument type" {
		t.Errorf(
			"expected invalid argument type, got %s",
			response.Message,
		)
	}

	if response.Code != "InvalidArgument" {
		t.Errorf("unexpected code: %s", response.Code)
	}

	if response.Field != "args[0]" {
		t.Errorf("unexpected field: %s", response.Field)
	}

	if response.Expected != "int" {
		t.Errorf("unexpected expected: %s", response.Expected)
	}

	if response.Actual != "string" {
		t.Errorf("unexpected actual: %s", response.Actual)
	}
}

func TestWriteGenericInvokeErrorCountMismatch(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	err := &service.GenericInvokeArgumentError{
		Kind:     service.ArgumentErrorCount,
		Field:    "args",
		Expected: "2",
		Actual:   "1",
	}

	writeGenericInvokeError(ctx, err)
	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"unexpected HTTP status: %d",
			recorder.Code,
		)
	}

	var response struct {
		Code     string `json:"code"`
		Message  string `json:"message"`
		Field    string `json:"field"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}

	if response.Code != "InvalidArgument" {
		t.Errorf("unexpected code: %s", response.Code)
	}

	if response.Message != "invalid argument count" {
		t.Errorf("unexpected message: %s", response.Message)
	}

	if response.Field != "args" {
		t.Errorf("unexpected field: %s", response.Field)
	}

	if response.Expected != "2" {
		t.Errorf("unexpected expected: %s", response.Expected)
	}

	if response.Actual != "1" {
		t.Errorf("unexpected actual: %s", response.Actual)
	}
}

func TestWriteGenericInvokeErrorFallback(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	err := bizerror.New(
		bizerror.NotFoundError,
		"method not found",
	)

	writeGenericInvokeError(ctx, err)

	if recorder.Code != http.StatusOK {
		t.Fatalf(
			"unexpected HTTP status: %d",
			recorder.Code,
		)
	}

	var response struct {
		Code     string          `json:"code"`
		Message  string          `json:"message"`
		Field    json.RawMessage `json:"field"`
		Expected json.RawMessage `json:"expected"`
		Actual   json.RawMessage `json:"actual"`
	}

	if err := json.Unmarshal(
		recorder.Body.Bytes(),
		&response,
	); err != nil {
		t.Fatal(err)
	}
	if response.Code != "NotFoundError" {
		t.Errorf(
			"expected NotFoundError, got %s",
			response.Code,
		)
	}
	if response.Message != "method not found" {
		t.Errorf(
			"unexpected message: %s",
			response.Message,
		)
	}
	if response.Field != nil ||
		response.Expected != nil ||
		response.Actual != nil {
		t.Fatalf(
			"unexpected structured fields: %s",
			recorder.Body.String(),
		)
	}
}
