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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/api"
	prom_v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	"github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/apache/dubbo-admin/pkg/logger"
	"github.com/apache/dubbo-admin/pkg/monitor/prometheus"
)

var (
	providerService     ProviderService = &ProviderServiceImpl{}
	consumerService     ConsumerService = &ConsumerServiceImpl{}
	providerServiceImpl                 = &ProviderServiceImpl{}
)

type PrometheusServiceImpl struct{}

func (p *PrometheusServiceImpl) PromDiscovery(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	// Reduce the call chain and improve performance.
	proAddr, err := providerServiceImpl.findAddresses()
	if err != nil {
		logger.Sugar().Errorf("Error provider findAddresses: %v\n", err)
		return err
	}
	var targets []string
	for i := 0; i < len(proAddr); i++ {
		targets = append(targets, util.GetDiscoveryPath(proAddr[i]))
	}
	filterCon := make(map[string]string)
	filterCon[constant.CategoryKey] = constant.ConsumersCategory
	servicesMap, err := util.FilterFromCategory(filterCon)
	if err != nil {
		logger.Sugar().Errorf("Error filter category: %v\n", err)
		return err
	}
	for _, url := range servicesMap {
		targets = append(targets, util.GetDiscoveryPath(url.Location))
	}
	target := []model.Target{
		{
			Targets: targets,
			Labels:  map[string]string{},
		},
	}
	err = json.NewEncoder(w).Encode(target)
	return err
}

func (p *PrometheusServiceImpl) ClusterMetrics() ([]model.Response, error) {
	res := make([]model.Response, 5)
	applications, err := providerService.FindApplications()
	appNum := 0
	if err != nil {
		logger.Sugar().Errorf("Error find applications: %v\n", err)
		res[0].Status = http.StatusInternalServerError
		res[0].Data = ""
	} else {
		appNum = len(applications)
		res[0].Status = http.StatusOK
		res[0].Data = strconv.Itoa(appNum)
	}
	services, err := providerService.FindServices()
	svc := 0
	if err != nil {
		logger.Sugar().Errorf("Error find services: %v\n", err)
		res[1].Status = http.StatusInternalServerError
		res[1].Data = ""
	} else {
		svc = len(services)
		res[1].Status = http.StatusOK
		res[1].Data = strconv.Itoa(svc)
	}
	providers, err := providerService.FindService(constant.IP, constant.AnyValue)
	pro := 0
	if err != nil {
		logger.Sugar().Errorf("Error find providers: %v\n", err)
		res[2].Status = http.StatusInternalServerError
		res[2].Data = ""
	} else {
		pro = len(providers)
		res[2].Status = http.StatusOK
		res[2].Data = strconv.Itoa(pro)
	}
	consumers, err := consumerService.FindAll()
	con := 0
	if err != nil {
		logger.Sugar().Errorf("Error find consumers: %v\n", err)
		res[3].Status = http.StatusInternalServerError
		res[3].Data = ""
	} else {
		con = len(consumers)
		res[3].Status = http.StatusOK
		res[3].Data = strconv.Itoa(con)
	}
	allInstance := pro + con
	res[5].Status = http.StatusOK
	res[5].Data = strconv.Itoa(allInstance)
	return res, nil
}

func (p *PrometheusServiceImpl) FlowMetrics() ([]model.Response, error) {
	res := make([]model.Response, 5)
	ip := config.PrometheusIp
	port := config.PrometheusPort
	address := fmt.Sprintf("http://%s:%s", ip, port)
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		logger.Sugar().Errorf("Error creating client: %v\n", err)
		return nil, err
	}
	v1api := prom_v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	vector1 := prometheus.FetchQuery(ctx, v1api, constant.MetricsQps, nil)
	err = vector1.Err
	if err != nil {
		logger.Sugar().Errorf("Error query qps: %v\n", err)
		res[0].Status = http.StatusInternalServerError
		res[0].Data = ""
	} else {
		qps := float64(vector1.Vector[0].Value)
		res[0].Status = http.StatusOK
		res[0].Data = fmt.Sprintf("%d", int(qps))
	}
	vector2 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestSuccessCount, nil)
	data1 := float64(vector2.Vector[0].Value)
	vector3 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestTotalCount, nil)
	data2 := float64(vector3.Vector[0].Value)
	if vector2.Err != nil && vector3.Err != nil {
		res[1].Status = http.StatusInternalServerError
		res[1].Data = ""
	} else {
		res[1].Status = http.StatusOK
		successRate := data1 / data2
		res[1].Data = fmt.Sprintf("%0.2f", successRate)
	}
	vector4 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestOutOfTimeCount, nil)
	data4 := float64(vector4.Vector[0].Value)
	if vector4.Err != nil {
		res[2].Status = http.StatusInternalServerError
		res[2].Data = ""
	} else {
		res[2].Status = http.StatusOK
		outOfTimeRate := data4 / data2
		res[2].Data = fmt.Sprintf("%0.2f", outOfTimeRate)
	}
	vector5 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestAddressNotFount, nil)
	data5 := float64(vector5.Vector[0].Value)
	if vector5.Err != nil {
		res[3].Status = http.StatusInternalServerError
		res[3].Data = ""
	} else {
		res[3].Status = http.StatusOK
		notFound := data5 / data2
		res[3].Data = fmt.Sprintf("%0.2f", notFound)
	}
	vector6 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestOtherException, nil)
	data6 := float64(vector6.Vector[0].Value)
	if vector6.Err != nil {
		res[4].Status = http.StatusInternalServerError
		res[4].Data = ""
	} else {
		res[4].Status = http.StatusOK
		other := data6 / data2
		res[4].Data = fmt.Sprintf("%0.2f", other)
	}
	return res, nil
}
