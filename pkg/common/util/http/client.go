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

package http

import (
	nethttp "net/http"
	"net/url"
	"path"
)

type Client interface {
	Do(req *nethttp.Request) (*nethttp.Response, error)
}

type ClientFunc func(req *nethttp.Request) (*nethttp.Response, error)

func (f ClientFunc) Do(req *nethttp.Request) (*nethttp.Response, error) {
	return f(req)
}

func ClientWithBaseURL(delegate Client, baseURL *url.URL, headers map[string]string) Client {
	return ClientFunc(func(req *nethttp.Request) (*nethttp.Response, error) {
		if req.URL != nil {
			req.URL.Scheme = baseURL.Scheme
			req.URL.Host = baseURL.Host
			req.URL.Path = path.Join(baseURL.Path, req.URL.Path)
			for k, v := range headers {
				req.Header.Add(k, v)
			}
		}
		return delegate.Do(req)
	})
}
