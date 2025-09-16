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

type PageReq struct {
	PageOffset int `form:"pageOffset" json:"pageOffset"`
	PageSize   int `form:"pageSize" json:"pageSize"`
}

type Pagination struct {
	Total      int `json:"total"`
	PageSize   int `json:"pageSize"`
	PageOffset int `json:"pageOffset"`
}

type PageData[T any] struct {
	Pagination
	Data []T `json:"data"`
}

func NewPageData[T any](total, pageOffset, pageSize int, data []T) *PageData[T] {
	return &PageData[T]{
		Pagination: Pagination{
			Total:      total,
			PageSize:   pageSize,
			PageOffset: pageOffset,
		},
		Data: data,
	}
}

func (pd *PageData[T]) WithTotal(total int) *PageData[T] {
	pd.Total = total
	return pd
}

func (pd *PageData[T]) WithCurPage(curPage int) *PageData[T] {
	pd.PageOffset = curPage
	return pd
}

func (pd *PageData[T]) WithPageSize(pageSize int) *PageData[T] {
	pd.PageSize = pageSize
	return pd
}

func (pd *PageData[T]) WithData(data []T) *PageData[T] {
	pd.Data = data
	return pd
}
