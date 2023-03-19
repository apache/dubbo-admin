// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.package services

package services

import (
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/monitor/prometheus"
)

type PrometheusServiceImpl struct{}

func (p *PrometheusServiceImpl) ClusterMetrics() ([]model.Response, error) {
	return nil, nil
}

func (p *PrometheusServiceImpl) FlowMetrics() ([]model.Response, error) {
	res := make([]model.Response, 5)
	// QPS
	qpsLabels := []string{""}
	resQps, err := prometheus.FetchRange(constant.MetricsQps, qpsLabels)
	if err != nil {
		return nil, err
	}
	res[0].Status = resQps.Status
	res[0].Data = resQps.Data.Result[0].Value[1].(string)
	// success rate
	successLabels1 := []string{""}
	successLabels2 := []string{""}
	resSuccess, err := prometheus.FetchRadio(constant.MetricsHttpRequestSuccessCount,
		constant.MetricsHttpRequestTotalCount, successLabels1, successLabels2)
	if err != nil {
		return nil, err
	}
	res[1].Status = resSuccess.Status
	res[1].Data = resSuccess.Data.Result[0].Value[1].(string)
	// out of time exception rate
	outofTimeLabels1 := []string{""}
	outofTimeLabels2 := []string{""}
	resOutOfTime, err := prometheus.FetchRadio(constant.MetricsHttpRequestOutOfTimeCount,
		constant.MetricsHttpRequestTotalCount, outofTimeLabels1, outofTimeLabels2)
	if err != nil {
		return nil, err
	}
	res[2].Status = resOutOfTime.Status
	res[2].Data = resOutOfTime.Data.Result[0].Value[1].(string)
	// 404 not found rate
	notFoundLabels1 := []string{""}
	notFoundLabels2 := []string{""}
	resnotFount, err := prometheus.FetchRadio(constant.MetricsHttpRequestAddressNotFount,
		constant.MetricsHttpRequestTotalCount, notFoundLabels1, notFoundLabels2)
	if err != nil {
		return nil, err
	}
	res[3].Status = resnotFount.Status
	res[3].Data = resnotFount.Data.Result[0].Value[1].(string)
	// other exception rate
	otherExceptionLabels1 := []string{""}
	otherExceptionLabels2 := []string{""}
	resOther, err := prometheus.FetchRadio(constant.MetricsHttpRequestOtherException,
		constant.MetricsHttpRequestTotalCount, otherExceptionLabels1, otherExceptionLabels2)
	if err != nil {
		return nil, err
	}
	res[4].Status = resOther.Status
	res[4].Data = resOther.Data.Result[0].Value[1].(string)
	return res, nil
}
