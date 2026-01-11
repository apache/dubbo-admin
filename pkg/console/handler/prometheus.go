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
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
)

func PromQL(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		promBaseUrl := ctx.Config().Console.PrometheusBaseURL
		if promBaseUrl == nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.ConfigError, "Please configure prometheus url to retrieve metrics")))
			return
		}

		u := *promBaseUrl
		u.RawQuery = c.Request.URL.RawQuery
		u.Path = "/api/v1/query"
		s := u.String()
		resp, err := http.Get(s)
		if err != nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.NetWorkError, err.Error())))
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusOK, model.NewBizErrorResp(
				bizerror.New(bizerror.NetWorkError, err.Error())))
			return
		}
		c.Data(http.StatusOK, resp.Header.Get("Content-Type"), body)
	}
}
