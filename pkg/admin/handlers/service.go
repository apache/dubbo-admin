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

package handlers

import (
	"dubbo.apache.org/dubbo-go/v3/config/generic"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	hessian "github.com/apache/dubbo-go-hessian2"
	"net/http"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"github.com/apache/dubbo-admin/pkg/admin/config"

	"github.com/apache/dubbo-admin/pkg/admin/util"

	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/version"
	"github.com/gin-gonic/gin"
)

var (
	providerService    services.ProviderService    = &services.ProviderServiceImpl{}
	consumerService    services.ConsumerService    = &services.ConsumerServiceImpl{}
	genericServiceImpl services.GenericServiceImpl = services.GenericServiceImpl{}
)

func AllServices(c *gin.Context) {
	services, err := providerService.FindServices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": services,
	})
}

func SearchService(c *gin.Context) {
	pattern := c.Query("pattern")
	filter := c.Query("filter")
	providers, err := providerService.FindService(pattern, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": providers,
	})
}

func AllApplications(c *gin.Context) {
	applications, err := providerService.FindApplications()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": applications,
	})
}

func AllConsumers(c *gin.Context) {
	consumers, err := consumerService.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": consumers,
	})
}

func ServiceDetail(c *gin.Context) {
	service := c.Param("service")
	group := util.GetGroup(service)
	version := util.GetVersion(service)
	interfaze := util.GetInterface(service)

	providers, err := providerService.FindByService(service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	consumers, err := consumerService.FindByService(service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	application := ""
	if len(providers) > 0 {
		application = providers[0].Application
	}
	identifier := &identifier.MetadataIdentifier{
		Application: application,
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: interfaze,
			Version:          version,
			Group:            group,
			Side:             "provider",
		},
	}
	metadata, _ := config.MetadataReportCenter.GetServiceDefinition(identifier)

	serviceDetail := &model.ServiceDetail{
		Providers:   providers,
		Consumers:   consumers,
		Service:     service,
		Application: application,
		Metadata:    metadata,
	}
	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": serviceDetail,
	})
}

func Version(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetVersion())
}

// ServiceTest

func Test(c *gin.Context) {
	env := c.Param("env")
	var serviceTestDTO model.ServiceTest
	err := c.BindJSON(&serviceTestDTO)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := genericService.Invoke(serviceTestDTO.Service, serviceTestDTO.Method, serviceTestDTO.ParameterTypes, serviceTestDTO.Params)
	c.JSON(http.StatusOK, result)

	return genericService.invoke(serviceTestDTO.getService(), serviceTestDTO.getMethod(), serviceTestDTO.getParameterTypes(), serviceTestDTO.getParams())

	refConf := genericServiceImpl.NewRefConf("dubbo-admin", serviceTestDTO)
	resp, err := refConf.
		GetRPCService().(*generic.GenericService).
		Invoke(
			context.TODO(),
			"GetUser",
			[]string{"java.lang.String"},
			[]hessian.Object{"A003"},
		)
}

func MethodDetail(c *gin.Context) {
	service := c.Param("service")
	application := c.Param("application")
	method := c.Param("method")

	info := util.ServiceName2Map(service)

	identifier := &identifier.MetadataIdentifier{
		Application: application,
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: info[constant.InterfaceKey],
			Version:          info[constant.VersionKey],
			Group:            info[constant.GroupKey],
			Side:             constant.ProviderSide,
		},
	}
	metadata, _ := config.MetadataReportCenter.GetServiceDefinition(identifier)
	var methodMetadata model.MethodMetadata
	if metadata != "" {
		var serviceTestUtil util.ServiceTestUtil
		release, err := providerService.FindVersionInApplication(application)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		if strings.HasPrefix(release, "2.") {
			serviceDefinition := &definition.FullServiceDefinition{}
			json.Unmarshal([]byte(metadata), &serviceDefinition)
			methods := serviceDefinition.Methods
			if methods != nil {
				for _, m := range methods {
					if serviceTestUtil.SameMethod(m, method) {
						methodMetadata = serviceTestUtil.GenerateMethodMeta(*serviceDefinition, m)
						break
					}
				}
			}
		}
	} else {
		var serviceTestV3Util util.ServiceTestV3Util
		serviceDefinition := &definition.FullServiceDefinition{}
		json.Unmarshal([]byte(metadata), &serviceDefinition)
		methods := serviceDefinition.Methods
		if methods != nil {
			for _, m := range methods {
				if serviceTestV3Util.SameMethod(m, method) {
					methodMetadata = serviceTestV3Util.GenerateMethodMeta(*serviceDefinition, m)
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": methodMetadata,
	})
}
