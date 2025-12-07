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

package util

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

func HandleServiceError(ctx *gin.Context, err error) {
	var e bizerror.Error
	if !errors.As(err, &e) {
		e = bizerror.New(bizerror.UnknownError, err.Error())
	}
	ctx.JSON(http.StatusOK, model.NewBizErrorResp(e))
}

func HandleArgumentError(ctx *gin.Context, err error) {
	e := bizerror.New(bizerror.InvalidArgument, err.Error())
	ctx.JSON(http.StatusOK, model.NewBizErrorResp(e))
}
