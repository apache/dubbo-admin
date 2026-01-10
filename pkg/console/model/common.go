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

import (
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type CommonResp struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func (r *CommonResp) WithCode(code string) *CommonResp {
	r.Code = code
	return r
}

func (r *CommonResp) WithMsg(msg string) *CommonResp {
	r.Message = msg
	return r
}

func (r *CommonResp) WithData(data any) *CommonResp {
	r.Data = data
	return r
}

func NewSuccessResp(data any) *CommonResp {
	return &CommonResp{
		Code:    "Success",
		Message: "success",
		Data:    data,
	}
}

// NewErrorResp TODO replace with NewBizErrorResp
func NewErrorResp(msg string) *CommonResp {
	return &CommonResp{
		Code:    string(bizerror.UnknownError),
		Message: msg,
		Data:    nil,
	}
}

func NewBizErrorResp(err bizerror.Error) *CommonResp {
	return &CommonResp{
		Code:    string(err.Code()),
		Message: err.Message(),
		Data:    nil,
	}
}

type SearchReq struct {
	coremodel.PageReq

	SearchType string `form:"searchType"`
	Keywords   string `form:"keywords"`
	Mesh       string `form:"mesh"`
}

func NewSearchReq() *SearchReq {
	return &SearchReq{
		PageReq: coremodel.PageReq{PageSize: 15},
	}
}

type SearchRes struct {
	Find       bool     `json:"find"`
	Candidates []string `json:"candidates"`
}
