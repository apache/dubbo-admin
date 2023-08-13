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
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"dubbo.apache.org/dubbo-go/v3/config/generic"

	"github.com/apache/dubbo-admin/pkg/core/cmd/version"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	hessian "github.com/apache/dubbo-go-hessian2"

	"dubbo.apache.org/dubbo-go/v3/metadata/definition"

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/vcraescu/go-paginator"
	"github.com/vcraescu/go-paginator/adapter"

	"dubbo.apache.org/dubbo-go/v3/metadata/identifier"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"

	"github.com/apache/dubbo-admin/pkg/admin/services"

	"github.com/gin-gonic/gin"
)

var (
	providerService    services.ProviderService     = &services.ProviderServiceImpl{}
	consumerService    services.ConsumerService     = &services.ConsumerServiceImpl{}
	monitorService     services.MonitorService      = &services.PrometheusServiceImpl{}
	genericServiceImpl *services.GenericServiceImpl = &services.GenericServiceImpl{}
	serviceTesting     *services.ServiceTestingV3   = &services.ServiceTestingV3{}
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

// Version show basic information of the Admin process
// @Summary      show basic information of the Admin process
// @Description  show basic information of the Admin process
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Success      200  {object}  version.Version
// @Router       /api/{env}/version [get]
func Version(c *gin.Context) {
	c.JSON(http.StatusOK, version.GetVersion())
}

// FlowMetrics show Prometheus collected metrics
// @Summary      show Prometheus collected metrics
// @Description  show Prometheus collected metrics
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.FlowMetricsRes
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/metrics/flow [get]
func FlowMetrics(c *gin.Context) {
	res, err := monitorService.FlowMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.HTTPError{Error: err.Error()})
	}
	c.JSON(http.StatusOK, res)
}

// ClusterMetrics show cluster overview
// @Summary      show cluster overview
// @Description  show cluster overview
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.ClusterMetricsRes
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/metrics/cluster [get]
func ClusterMetrics(c *gin.Context) {
	res, err := monitorService.ClusterMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, res)
}

// Metadata show metadata of the cluster, like dubbo versions, protocols, etc.
// @Summary      show metadata of the cluster, like dubbo versions, protocols, etc.
// @Description  show metadata of the cluster, like dubbo versions, protocols, etc.
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.Metadata
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/metrics/metadata [get]
func Metadata(c *gin.Context) {
	res, err := monitorService.ClusterMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, res)
}

// PromDiscovery expose the interface of Prometheus http_sd service discovery.
// @Summary      expose the interface of Prometheus http_sd service discovery.
// @Description  expose the interface of Prometheus http_sd service discovery.
// @Tags         metrics
// @Accept       json
// @Produce      json
// @Success      200  {object}  []model.Target
// @Failure      500  {object}  model.HTTPError
// @Router       /api/{env}/metrics/prometheus [get]
func PromDiscovery(c *gin.Context) {
	targets, err := monitorService.PromDiscovery(c.Writer)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
	}
	c.JSON(http.StatusOK, targets)
}

// Test works for dubbo2 tcp protocol
func Test(c *gin.Context) {
	var serviceTestDTO model.ServiceTest

	err := c.BindJSON(&serviceTestDTO)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	refConf := genericServiceImpl.NewRefConf("dubbo-admin", serviceTestDTO.Service, "dubbo")
	time.Sleep(2 * time.Second)
	resp, err := refConf.
		GetRPCService().(*generic.GenericService).
		Invoke(
			c,
			serviceTestDTO.Method,
			serviceTestDTO.ParameterTypes,
			[]hessian.Object{"A003"}, // fixme
		)
	refConf.GetInvoker().Destroy()
	if err != nil {
		logger.Error("Error do generic invoke for service test", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, resp)
}

// HttpTest works for triple protocol
func HttpTest(c *gin.Context) {
	// pattern := c.Query("service")
	// filter := c.Query("method")
	// address := c.Query("address")

	// send standard http request to backend http://address/service/method content-type:json

	c.JSON(http.StatusOK, gin.H{
		"code": 1,
		"data": "implement me",
	})
}

func MethodDetail(c *gin.Context) {
	service := c.Query("service")
	group := util.GetGroup(service)
	version := util.GetVersion(service)
	interfaze := util.GetInterface(service)
	application := c.Query("application")
	method := c.Query("method")

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
	var methodMetadata model.MethodMetadata
	if metadata != "" {
		serviceDefinition := &definition.FullServiceDefinition{}
		err := json.Unmarshal([]byte(metadata), &serviceDefinition)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		methods := serviceDefinition.Methods
		if methods != nil {
			for _, m := range methods {
				if serviceTesting.SameMethod(m, method) {
					methodMetadata = serviceTesting.GenerateMethodMeta(*serviceDefinition, m)
					break
				}
			}
		}
	}

	c.JSON(http.StatusOK, methodMetadata)
}
