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

package model

const (
	successCode      = 200
	unauthorizedCode = 401
	errorCode        = 500
)

type CommonResp struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data"`
}

func (r *CommonResp) WithCode(code int) *CommonResp {
	r.Code = code
	return r
}

func (r *CommonResp) WithMsg(msg string) *CommonResp {
	r.Msg = msg
	return r
}

func (r *CommonResp) WithData(data any) *CommonResp {
	r.Data = data
	return r
}

func NewSuccessResp(data any) *CommonResp {
	return &CommonResp{
		Code: successCode,
		Msg:  "success",
		Data: data,
	}
}

func NewUnauthorizedResp() *CommonResp {
	return &CommonResp{
		Code: unauthorizedCode,
		Msg:  "UnAuthorized, please login",
		Data: nil,
	}
}

func NewErrorResp(msg string) *CommonResp {
	return &CommonResp{
		Code: errorCode,
		Msg:  msg,
		Data: nil,
	}
}

type PageData struct {
	Total    int `json:"total"`
	CurPage  int `json:"curPage"`
	PageSize int `json:"pageSize"`
	Data     any `json:"data"`
}

func NewPageData() *PageData {
	return &PageData{}
}

func (pd *PageData) WithTotal(total int) *PageData {
	pd.Total = total
	return pd
}

func (pd *PageData) WithCurPage(curPage int) *PageData {
	pd.CurPage = curPage
	return pd
}

func (pd *PageData) WithPageSize(pageSize int) *PageData {
	pd.PageSize = pageSize
	return pd
}

func (pd *PageData) WithData(data any) *PageData {
	pd.Data = data
	return pd
}

type PageReq struct {
	PageOffset int `form:"pageOffset" json:"pageOffset"`
	PageSize   int `form:"pageSize" json:"pageSize"`
}

type SearchReq struct {
	SearchType string `form:"searchType"`
	Keywords   string `form:"keywords"`
	PageReq
}

func NewSearchReq() *SearchReq {
	return &SearchReq{
		PageReq: PageReq{PageSize: 15},
	}
}

type SearchRes struct {
	Find       bool     `json:"find"`
	Candidates []string `json:"candidates"`
}
