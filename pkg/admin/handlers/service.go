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
	"net/http"

	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"

	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/version"
	"github.com/gin-gonic/gin"
)

var (
	providerService services.ProviderService = &services.ProviderServiceImpl{}
	consumerService services.ConsumerService = &services.ConsumerServiceImpl{}
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
