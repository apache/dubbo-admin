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
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

func writeGenericInvokeError(c *gin.Context, err error) {
	var validationErr *model.ServiceGenericInvokeValidationError
	if errors.As(err, &validationErr) {
		writeGenericInvokeFieldResponse(
			c,
			model.NewBizErrorResp(bizerror.New(bizerror.InvalidArgument, validationErr.Error())),
			validationErr.Field,
			validationErr.Expected,
			validationErr.Actual,
		)
		return
	}

	var fieldErr *service.GenericInvokeArgumentError

	if !errors.As(err, &fieldErr) {
		util.HandleServiceError(c, err)
		return
	}
	message := "invalid argument"

	switch fieldErr.Kind {
	case service.ArgumentErrorType:
		message = "invalid argument type"

	case service.ArgumentErrorCount:
		message = "invalid argument count"

	case service.ArgumentErrorSignature:
		message = "invalid method signature"
	}

	baseResp := model.NewBizErrorResp(
		bizerror.New(
			bizerror.InvalidArgument,
			message,
		),
	)
	if fieldErr.Kind == service.ArgumentErrorSignature {
		var bizErr bizerror.Error

		if errors.As(err, &bizErr) {
			baseResp = model.NewBizErrorResp(bizErr)
		}
	}

	writeGenericInvokeFieldResponse(
		c,
		baseResp,
		fieldErr.Field,
		fieldErr.Expected,
		fieldErr.Actual,
	)
}

func writeGenericInvokeFieldResponse(
	c *gin.Context,
	baseResp *model.CommonResp,
	field string,
	expected string,
	actual string,
) {
	c.JSON(http.StatusOK, struct {
		*model.CommonResp

		Field    string `json:"field"`
		Expected string `json:"expected"`
		Actual   string `json:"actual"`
	}{
		CommonResp: baseResp,
		Field:      field,
		Expected:   expected,
		Actual:     actual,
	})
}
