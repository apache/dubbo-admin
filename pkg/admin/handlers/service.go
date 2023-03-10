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

	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/prometheus"
	"github.com/apache/dubbo-admin/pkg/admin/services"

	"github.com/gin-gonic/gin"
)

var providerService services.ProviderService = &services.ProviderServiceImpl{}

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

func FlowMetrics(c *gin.Context) {
	res := make([]prometheus.Response, 5)
	// QPS
	qpsLabels := []string{""}
	resQps, err := prometheus.FetchRange(constant.MetricsQps, qpsLabels)
	if err != nil {
		panic(err)
	}
	res[0].Status = resQps.Status
	res[0].Data = resQps.Data.Result[0].Value[1].(string)
	// Success rate
	successLabels1 := []string{""}
	successLabels2 := []string{""}
	resSuccess, err := prometheus.FetchRadio(constant.MetricsHttpRequestSuccessCount,
		constant.MetricsHttpRequestTotalCount, successLabels1, successLabels2)
	if err != nil {
		panic(err)
	}
	res[1].Status = resSuccess.Status
	res[1].Data = resSuccess.Data.Result[0].Value[1].(string)
	// Timeout exception rate
	outOfTimeLabels1 := []string{""}
	outOfTimeLabels2 := []string{""}
	resOutOfTime, err := prometheus.FetchRadio(constant.MetricsHttpRequestOutOfTimeCount,
		constant.MetricsHttpRequestTotalCount, outOfTimeLabels1, outOfTimeLabels2)
	if err != nil {
		panic(err)
	}
	res[2].Status = resOutOfTime.Status
	res[2].Data = resOutOfTime.Data.Result[0].Value[1].(string)
	// Address not found rate
	notFoundLabels1 := []string{""}
	notFoundLabels2 := []string{""}
	resnotFount, err := prometheus.FetchRadio(constant.MetricsHttpRequestAddressNotFount,
		constant.MetricsHttpRequestTotalCount, notFoundLabels1, notFoundLabels2)
	if err != nil {
		panic(err)
	}
	res[3].Status = resnotFount.Status
	res[3].Data = resnotFount.Data.Result[0].Value[1].(string)
	// other abnormal rate
	otherExceptionLabels1 := []string{""}
	otherExceptionLabels2 := []string{""}
	resOther, err := prometheus.FetchRadio(constant.MetricsHttpRequestOtherException,
		constant.MetricsHttpRequestTotalCount, otherExceptionLabels1, otherExceptionLabels2)
	if err != nil {
		panic(err)
	}
	res[4].Status = resOther.Status
	res[4].Data = resOther.Data.Result[0].Value[1].(string)
	c.JSON(http.StatusOK, res)
}

func ClusterMetrics(c *gin.Context) {
}
