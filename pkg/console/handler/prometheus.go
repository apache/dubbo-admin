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

// proxy for prometheus

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
)

func PromQL(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		query := c.Request.URL.Query().Get("query")
		values := url.Values{}
		values.Add("query", query)
		promUrl := ctx.Config().Console.Prometheus + "/api/v1/query?" + values.Encode()
		proxyUrl, _ := url.Parse(promUrl)
		director := func(req *http.Request) {
			req.URL.Scheme = proxyUrl.Scheme
			req.URL.Host = proxyUrl.Host
			req.Host = proxyUrl.Host
			req.URL.Path = proxyUrl.Path
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
