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

import meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"

type MCPServerCreateReq struct {
	Mesh  string                       `json:"mesh" binding:"required"`
	Name  string                       `json:"name" binding:"required"`
	Draft *meshproto.MCPServerSnapshot `json:"draft"`
}

type MCPDraftReq struct {
	ExpectedResourceVersion string                       `json:"expectedResourceVersion" binding:"required"`
	Draft                   *meshproto.MCPServerSnapshot `json:"draft" binding:"required"`
}

type MCPToolReq struct {
	ExpectedResourceVersion string                    `json:"expectedResourceVersion" binding:"required"`
	Tool                    *meshproto.MCPToolBinding `json:"tool" binding:"required"`
}

type MCPPublishReq struct {
	ExpectedResourceVersion string `json:"expectedResourceVersion" binding:"required"`
}

type MCPValidationIssue struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	ToolID  string `json:"toolId,omitempty"`
}

type MCPValidationResult struct {
	Valid    bool                 `json:"valid"`
	Errors   []MCPValidationIssue `json:"errors"`
	Warnings []MCPValidationIssue `json:"warnings"`
}

type MCPCredentialCreateReq struct {
	DisplayName string `json:"displayName" binding:"required"`
	ExpiresAt   string `json:"expiresAt,omitempty"`
}

type MCPCredentialView struct {
	ID              string `json:"id"`
	Mesh            string `json:"mesh"`
	ServerID        string `json:"serverId"`
	ResourceVersion string `json:"resourceVersion"`
	DisplayName     string `json:"displayName"`
	Status          string `json:"status"`
	ExpiresAt       string `json:"expiresAt,omitempty"`
	CreatedAt       string `json:"createdAt"`
}
