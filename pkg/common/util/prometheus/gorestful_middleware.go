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

package prometheus

import (
	"context"

	gorestful "github.com/emicklei/go-restful/v3"
	"github.com/slok/go-http-metrics/middleware"
)

// MetricsHandler is based on go-restful middleware.
//
// In the original version, URLPath() uses r.req.Request.URL.Path which results in following stats when querying for individual DPs
// api_server_http_response_size_bytes_bucket{code="201",handler="/meshes/default/dataplanes/backend-01",method="PUT",service="",le="100"} 1
// api_server_http_response_size_bytes_bucket{code="201",handler="/meshes/default/dataplanes/ingress-01",method="PUT",service="",le="100"} 1
// this is not scalable solution, we would be producing too many metrics. With r.req.SelectedRoutePath() the metrics look like this
// api_server_http_request_duration_seconds_bucket{code="201",handler="/meshes/{mesh}/dataplanes/{name}",method="PUT",service="",le="0.005"} 3
func MetricsHandler(handlerID string, m middleware.Middleware) gorestful.FilterFunction {
	return func(req *gorestful.Request, resp *gorestful.Response, chain *gorestful.FilterChain) {
		r := &reporter{req: req, resp: resp}
		m.Measure(handlerID, r, func() {
			chain.ProcessFilter(req, resp)
		})
	}
}

type reporter struct {
	req  *gorestful.Request
	resp *gorestful.Response
}

func (r *reporter) Method() string { return r.req.Request.Method }

func (r *reporter) Context() context.Context { return r.req.Request.Context() }

func (r *reporter) URLPath() string {
	return r.req.SelectedRoutePath()
}

func (r *reporter) StatusCode() int { return r.resp.StatusCode() }

func (r *reporter) BytesWritten() int64 { return int64(r.resp.ContentLength()) }
