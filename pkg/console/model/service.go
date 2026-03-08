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
	"fmt"
	"strings"

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
	s.ServiceName = c.Query("serviceName")
	if s.ServiceName == "" {
		return fmt.Errorf("service name is empty")
	}
	s.Group = c.Query("group")
	s.Version = c.Query("version")
	s.Mesh = c.Query("mesh")
	return nil
}

func (s *BaseServiceReq) ServiceKey() string {
	return s.ServiceName + constants.ColonSeparator + s.Version + constants.ColonSeparator + s.Group
}

type ServiceMethodsReq struct {
	ServiceName     string `form:"serviceName" json:"serviceName"`
	Group           string `form:"group" json:"group"`
	Version         string `form:"version" json:"version"`
	Mesh            string `form:"mesh" json:"mesh"`
	ProviderAppName string `form:"providerAppName" json:"providerAppName"`
}

func (s *ServiceMethodsReq) Query(c *gin.Context) error {
	s.ServiceName = strings.TrimSpace(c.Query("serviceName"))
	if s.ServiceName == "" {
		return fmt.Errorf("service name is empty")
	}
	s.Mesh = strings.TrimSpace(c.Query("mesh"))
	if s.Mesh == "" {
		return fmt.Errorf("mesh is empty")
	}
	s.Group = strings.TrimSpace(c.Query("group"))
	s.Version = strings.TrimSpace(c.Query("version"))
	s.ProviderAppName = strings.TrimSpace(c.Query("providerAppName"))
	return nil
}

type ServiceMethodDetailReq struct {
	ServiceMethodsReq

	MethodName string `form:"methodName" json:"methodName"`
	Signature  string `form:"signature" json:"signature"`
}

func (s *ServiceMethodDetailReq) Query(c *gin.Context) error {
	if err := s.ServiceMethodsReq.Query(c); err != nil {
		return err
	}
	s.MethodName = strings.TrimSpace(c.Query("methodName"))
	if s.MethodName == "" {
		return fmt.Errorf("method name is empty")
	}
	s.Signature = strings.TrimSpace(c.Query("signature"))
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
	ParameterTypes []string                 `json:"parameterTypes"`
	Parameters     []ServiceMethodParameter `json:"parameters"`
	ReturnType     string                   `json:"returnType"`
}
