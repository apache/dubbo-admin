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
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"encoding/json"
	"github.com/apache/dubbo-admin/pkg/logger"
	"net/http"
	"strconv"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/vcraescu/go-paginator"
	"github.com/vcraescu/go-paginator/adapter"

	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"

	"github.com/apache/dubbo-admin/pkg/admin/services"
	"github.com/apache/dubbo-admin/pkg/version"
	"github.com/gin-gonic/gin"
)

var (
	providerService   services.ProviderService = &services.ProviderServiceImpl{}
	consumerService   services.ConsumerService = &services.ConsumerServiceImpl{}
	prometheusService services.MonitorService  = &services.PrometheusServiceImpl{}
)

// AllServices get all services
// @Summary      Get all services
// @Description  Get all services
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Success      200  {object}  []string
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/services [get]
func AllServices(c *gin.Context) {
	allServices, err := providerService.FindServices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, allServices)
}

// SearchService search services by different patterns and keywords
// @Summary      Search services by different patterns and keywords
// @Description  Search services by different patterns and keywords
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        env       	path     string     false   "environment"       default(dev)
// @Param        pattern    query    string     true    "supported values: application, service or ip"
// @Param        filter     query    string     true    "keyword to search"
// @Param        page       query    string     false   "page number"
// @Param        size       query    string     false   "page size"
// @Success      200  {object}  model.ListServiceByPage
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/service [get]
func SearchService(c *gin.Context) {
	pattern := c.Query("pattern")
	filter := c.Query("filter")
	page := c.Query("page")
	pageInt, err := strconv.Atoi(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	size := c.Query("size")
	sizeInt, err := strconv.Atoi(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	// get services
	serviceDTOS, err := providerService.FindService(pattern, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	// paging
	p := paginator.New(adapter.NewSliceAdapter(serviceDTOS), sizeInt)
	p.SetPage(pageInt)
	var serviceResults []*model.ServiceDTO
	p.Results(&serviceResults)
	// return results
	c.JSON(http.StatusOK, model.ListServiceByPage{
		Content:       serviceResults,
		TotalPages:    p.PageNums(),
		TotalElements: p.Nums(),
		Size:          size,
		First:         pageInt == 0,
		Last:          pageInt == p.PageNums()-1,
		PageNumber:    page,
		Offset:        (pageInt - 1) * sizeInt,
	})
}

// AllApplications get all applications
// @Summary      Get all applications
// @Description  Get all applications
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Success      200  {object}  []string
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/applications [get]
func AllApplications(c *gin.Context) {
	applications, err := providerService.FindApplications()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, applications)
}

// AllConsumers get all consumers
// @Summary      Get all consumers
// @Description  Get all consumers
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        env       			path     string     false  "environment"       default(dev)
// @Success      200  {object}  []string
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/consumers [get]
func AllConsumers(c *gin.Context) {
	consumers, err := consumerService.FindAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, consumers)
}

// ServiceDetail show detail of the specified service
// @Summary      Show detail of the specified service
// @Description  Show detail of the specified service
// @Tags         Services
// @Accept       json
// @Produce      json
// @Param        env       	path     string     false   "environment"       default(dev)
// @Param        service    path     string     true    "service format: 'group/service:version'"
// @Success      200  {object}  model.ServiceDetailDTO
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/service/{service} [get]
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
			Side:             constant.ProviderSide,
		},
	}
	metadata, _ := config.MetadataReportCenter.GetServiceDefinition(identifier)

	typed_meta := definition.ServiceDefinition{}
	err = json.Unmarshal([]byte(metadata), &typed_meta)
	if err != nil {
		logger.Errorf("Error parsing metadata, err msg is %s", err.Error())
	}

	serviceDetail := &model.ServiceDetailDTO{
		Providers:   providers,
		Consumers:   consumers,
		Service:     service,
		Application: application,
		Metadata:    typed_meta,
	}
	c.JSON(http.StatusOK, serviceDetail)
}

func Version(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetVersion())
}

func FlowMetrics(c *gin.Context) {
	res, err := prometheusService.FlowMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, res)
}

func ClusterMetrics(c *gin.Context) {
	res, err := prometheusService.ClusterMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, res)
}
