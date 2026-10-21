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
	"net/http"

	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

func GetApplicationEvents(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.EventQueryReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.ListApplicationEvents(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetInstanceEvents(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.EventQueryReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.ListInstanceEvents(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetServiceEvents(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.EventQueryReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.ListServiceEvents(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}
