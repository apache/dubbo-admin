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
	"encoding/json"
	"fmt"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type ServiceSearchReq struct {
	coremodel.PageReq

	ServiceName string `form:"serviceName" json:"serviceName"`
	Keywords    string `form:"keywords" json:"keywords"`
	Mesh        string `form:"mesh" json:"mesh"`
}

func NewServiceSearchReq() *ServiceSearchReq {
	return &ServiceSearchReq{
		PageReq: coremodel.PageReq{
			PageOffset: 0,
			PageSize:   15,
		},
	}
}

type ServiceSearchResp struct {
	ServiceName     string `json:"serviceName"`
	Version         string `json:"version"`
	Group           string `json:"group"`
	ProviderAppName string `json:"providerAppName,omitempty"`
	ConsumerAppName string `json:"consumerAppName,omitempty"`
}

type ByServiceName []*ServiceSearchResp

func (a ByServiceName) Len() int { return len(a) }

func (a ByServiceName) Less(i, j int) bool {
	return a[i].ServiceName < a[j].ServiceName
}

func (a ByServiceName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ServiceTabDistributionReq struct {
	ServiceName     string `json:"serviceName"  form:"serviceName" binding:"required"`
	Version         string `json:"version"  form:"version"`
	Group           string `json:"group"  form:"group"`
	Side            string `json:"side" form:"side"  binding:"required"`
	Mesh            string `json:"mesh" form:"mesh" binding:"required"`
	ProviderAppName string `json:"providerAppName"  form:"providerAppName"`
	Keywords        string `json:"keywords"  form:"keywords"`
	coremodel.PageReq
}

type ServiceTabDistributionResp struct {
	AppName      string            `json:"appName"`
	InstanceName string            `json:"instanceName"`
	Endpoint     string            `json:"endpoint"`
	TimeOut      string            `json:"timeOut"`
	Retries      string            `json:"retries"`
	Params       map[string]string `json:"params"`
}

type ByServiceInstanceName []*ServiceTabDistributionResp

func (a ByServiceInstanceName) Len() int { return len(a) }

func (a ByServiceInstanceName) Less(i, j int) bool {
	return a[i].InstanceName < a[j].InstanceName
}

func (a ByServiceInstanceName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

type ServiceTabDistribution struct {
	AppName      string
	InstanceName string
	Endpoint     string
	TimeOut      string
	Retries      string
}

type BaseServiceReq struct {
	ServiceName string `json:"serviceName"`
	Group       string `json:"group"`
	Version     string `json:"version"`
	Mesh        string `json:"mesh"`
}

func (s *BaseServiceReq) Query(c *gin.Context) error {
	s.ServiceName = strings.TrimSpace(c.Query("serviceName"))
	if strutil.IsBlank(s.ServiceName) {
		return fmt.Errorf("service name is empty")
	}
	s.Group = strings.TrimSpace(c.Query("group"))
	s.Version = strings.TrimSpace(c.Query("version"))
	s.Mesh = strings.TrimSpace(c.Query("mesh"))
	return nil
}

func (s *BaseServiceReq) ServiceKey() string {
	return s.ServiceName + constants.ColonSeparator + s.Version + constants.ColonSeparator + s.Group
}

type ServiceMethodDetailReq struct {
	BaseServiceReq

	MethodName string `form:"methodName" json:"methodName"`
	Signature  string `form:"signature" json:"signature"`
}

func (s *ServiceMethodDetailReq) Query(c *gin.Context) error {
	if err := s.BaseServiceReq.Query(c); err != nil {
		return err
	}
	s.MethodName = strings.TrimSpace(c.Query("methodName"))
	if strutil.IsBlank(s.MethodName) {
		return fmt.Errorf("method name is empty")
	}
	s.Signature = strings.TrimSpace(c.Query("signature"))
	if strutil.IsBlank(s.Signature) {
		return fmt.Errorf("signature is empty")
	}
	return nil
}

type ServiceMethodSummaryResp struct {
	MethodName     string   `json:"methodName"`
	ParameterTypes []string `json:"parameterTypes"`
	Signature      string   `json:"signature,omitempty"`
}

type ServiceMethodParameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ServiceMethodDetailResp struct {
	MethodName     string                   `json:"methodName"`
	Signature      string                   `json:"signature,omitempty"`
	ParameterTypes []string                 `json:"parameterTypes"`
	Parameters     []ServiceMethodParameter `json:"parameters"`
	ReturnType     string                   `json:"returnType"`
	Types          []ServiceMethodTypeResp  `json:"types"`
}

type ServiceMethodTypeResp struct {
	Type       string            `json:"type"`
	Properties map[string]string `json:"properties"`
	Items      []string          `json:"items"`
	Enums      []string          `json:"enums"`
}

const DefaultServiceGenericInvokeTimeoutMs int64 = 3000

type ServiceGenericInvokeReq struct {
	BaseServiceReq

	InstanceName string            `json:"instanceName"`
	MethodName   string            `json:"methodName"`
	Signature    string            `json:"signature"`
	Args         []json.RawMessage `json:"args"`
	TimeoutMs    int64             `json:"timeoutMs"`
	Attachments  map[string]string `json:"attachments"`
}

func (s *ServiceGenericInvokeReq) Validate() error {
	s.Mesh = strings.TrimSpace(s.Mesh)
	if strutil.IsBlank(s.Mesh) {
		return fmt.Errorf("mesh is empty")
	}

	s.ServiceName = strings.TrimSpace(s.ServiceName)
	if strutil.IsBlank(s.ServiceName) {
		return fmt.Errorf("service name is empty")
	}

	s.MethodName = strings.TrimSpace(s.MethodName)
	if strutil.IsBlank(s.MethodName) {
		return fmt.Errorf("method name is empty")
	}

	s.Signature = strings.TrimSpace(s.Signature)
	if strutil.IsBlank(s.Signature) {
		return fmt.Errorf("signature is empty")
	}

	s.InstanceName = strings.TrimSpace(s.InstanceName)
	if strutil.IsBlank(s.InstanceName) {
		return fmt.Errorf("instance name is empty")
	}

	s.Group = strings.TrimSpace(s.Group)
	s.Version = strings.TrimSpace(s.Version)

	if s.TimeoutMs <= 0 {
		s.TimeoutMs = DefaultServiceGenericInvokeTimeoutMs
	}

	return nil
}

type ServiceGenericInvokeResp struct {
	ElapsedMs int64 `json:"elapsedMs"`
	RawResult any   `json:"rawResult"`
}
