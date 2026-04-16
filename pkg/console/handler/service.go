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

package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/console/service"
	"github.com/apache/dubbo-admin/pkg/console/util"
)

const (
	DefaultTimeout = 1000
	DefaultRetries = 2
)

func SearchServices(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.NewServiceSearchReq()
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.SearchServices(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetServiceTabDistribution(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceTabDistributionReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetServiceTabDistribution(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetServiceProviderInstances(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.BaseServiceReq{}
		if err := req.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.GetServiceProviderInstances(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func GetServiceMethodNames(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.BaseServiceReq{}
		if err := req.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		methodNames, err := service.GetServiceMethodNames(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(methodNames))
	}
}

func GetServiceMethodDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.ServiceMethodDetailReq{}
		if err := req.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.GetServiceMethodDetail(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceGenericInvoke(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.ServiceGenericInvokeReq{}
		if err := c.ShouldBindJSON(&req); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		if err := req.Validate(); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		resp, err := service.InvokeServiceGeneric(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceConfigTimeoutGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.BaseServiceReq{}
		resp := struct {
			Timeout int32 `json:"timeout"`
		}{DefaultTimeout}
		if err := req.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		timeout, err := service.GetServiceTimeoutConfig(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		resp.Timeout = timeout
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceConfigTimeoutPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			Timeout int32 `json:"timeout"`
		}{}
		Mesh := c.Query("mesh")
		param.Mesh = Mesh
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		err := service.UpInsertServiceConfigTimeoutConfig(ctx, param.BaseServiceReq, param.Timeout)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRetryGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := model.BaseServiceReq{}
		resp := struct {
			RetryTimes int32 `json:"retryTimes"`
		}{DefaultRetries}
		if err := param.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		retries, err := service.GetServiceRetryConfig(ctx, param)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		resp.RetryTimes = retries
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceConfigRetryPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			RetryTimes int32 `json:"retryTimes"`
		}{}
		Mesh := c.Query("mesh")
		param.Mesh = Mesh
		if err := c.Bind(&param); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		if err := service.UpInsertServiceRetryConfig(ctx, param.BaseServiceReq, param.RetryTimes); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigRegionPriorityGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := model.BaseServiceReq{}
		resp := struct {
			Enabled bool `json:"enabled"`
		}{false}
		if err := req.Query(c); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		openSameRegionPrior, err := service.GetServiceRegionPriorityConfig(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		resp.Enabled = openSameRegionPrior
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceConfigRegionPriorityPUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			Enabled bool `json:"enabled"`
		}{}
		Mesh := c.Query("mesh")
		param.Mesh = Mesh
		if err := c.Bind(&param); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}
		if err := service.UpInsertServiceRegionPriorityConfig(ctx, param.BaseServiceReq, param.Enabled); err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

func ServiceConfigArgumentRouteGET(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := struct {
			model.BaseServiceReq
		}{}
		if err := req.Query(c); err != nil {
			util.HandleArgumentError(c, err)
			return
		}
		resp, err := service.GetServiceArgumentRouteConfig(ctx, req.BaseServiceReq)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

func ServiceConfigArgumentRoutePUT(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		param := struct {
			model.BaseServiceReq
			model.ServiceArgumentRoute
		}{}
		Mesh := c.Query("mesh")
		param.Mesh = Mesh
		if err := c.Bind(&param); err != nil {
			util.HandleArgumentError(c, err)
			return
		}

		err := service.UpInsertServiceArgumentRouteConfig(ctx, param.BaseServiceReq, param.ServiceArgumentRoute)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(nil))
	}
}

// GetServiceGraph returns the service graph as graph data (nodes and edges) for visualization
func GetServiceGraph(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceGraphReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GraphServices(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

// GetServiceDetail returns service detail information
func GetServiceDetail(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		req := &model.ServiceDetailReq{}
		if err := c.ShouldBindQuery(req); err != nil {
			c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
			return
		}

		resp, err := service.GetServiceDetail(ctx, req)
		if err != nil {
			util.HandleServiceError(c, err)
			return
		}

		c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	}
}

// GetServiceInterfaces returns service interfaces information
func GetServiceInterfaces(ctx consolectx.Context) gin.HandlerFunc {
	// return func(c *gin.Context) {
	// 	req := &model.ServiceInterfacesReq{}
	// 	if err := c.ShouldBindQuery(req); err != nil {
	// 		c.JSON(http.StatusBadRequest, model.NewErrorResp(err.Error()))
	// 		return
	// 	}

	// 	resp, err := service.GetServiceInterfaces(ctx, req)
	// 	if err != nil {
	// 		util.HandleServiceError(c, err)
	// 		return
	// 	}

	// 	c.JSON(http.StatusOK, model.NewSuccessResp(resp))
	// }
	return nil
}
