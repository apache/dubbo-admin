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
	"fmt"
	"net/http"
	"time"

	logger2 "github.com/apache/dubbo-admin/pkg/core/logger"

	set "github.com/dubbogo/gost/container/set"

	"github.com/prometheus/client_golang/api"
	prom_v1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"github.com/apache/dubbo-admin/pkg/admin/config"
	"github.com/apache/dubbo-admin/pkg/admin/constant"
	"github.com/apache/dubbo-admin/pkg/admin/model"
	util2 "github.com/apache/dubbo-admin/pkg/admin/util"
	"github.com/apache/dubbo-admin/pkg/core/monitor/prometheus"
)

var (
	providerService     ProviderService = &ProviderServiceImpl{}
	consumerService     ConsumerService = &ConsumerServiceImpl{}
	providerServiceImpl                 = &ProviderServiceImpl{}
)

type PrometheusServiceImpl struct{}

func (p *PrometheusServiceImpl) PromDiscovery(w http.ResponseWriter) ([]model.Target, error) {
	w.Header().Set("Content-Type", "application/json")
	// Reduce the call chain and improve performance.

	// Find all provider addresses
	proAddr, err := providerServiceImpl.findAddresses()
	if err != nil {
		logger2.Sugar().Errorf("Error provider findAddresses: %v\n", err)
		return nil, err
	}
	addresses := set.NewSet()
	items := proAddr.Values()
	for i := 0; i < len(items); i++ {
		addresses.Add(util2.GetDiscoveryPath(items[i].(string)))
	}

	targets := make([]string, 0, addresses.Size())
	items = addresses.Values()
	for _, v := range items {
		targets = append(targets, v.(string))
	}

	target := []model.Target{
		{
			Targets: targets,
			Labels:  map[string]string{},
		},
	}
	return target, err
}

func (p *PrometheusServiceImpl) ClusterMetrics() (model.ClusterMetricsRes, error) {
	res := model.ClusterMetricsRes{
		Data: make(map[string]int),
	}
	// total application number
	applications, err := providerService.FindApplications()
	appNum := 0
	if err != nil {
		logger2.Sugar().Errorf("Error find applications: %v\n", err)
	} else {
		appNum = applications.Size()
	}
	res.Data["application"] = appNum

	// total service number
	services, err := providerService.FindServices()
	svc := 0
	if err != nil {
		logger2.Sugar().Errorf("Error find services: %v\n", err)
	} else {
		svc = services.Size()
	}
	res.Data["services"] = svc

	providers, err := providerService.FindService(constant.IP, constant.AnyValue)
	pro := 0
	if err != nil {
		logger2.Sugar().Errorf("Error find providers: %v\n", err)
	} else {
		pro = len(providers)
	}
	res.Data["providers"] = pro

	consumers, err := consumerService.FindAll()
	con := 0
	if err != nil {
		logger2.Sugar().Errorf("Error find consumers: %v\n", err)
	} else {
		con = len(consumers)
	}
	res.Data["consumers"] = con

	res.Data["all"] = con
	return res, nil
}

func (p *PrometheusServiceImpl) FlowMetrics() (model.FlowMetricsRes, error) {
	res := model.FlowMetricsRes{
		Data: make(map[string]float64),
	}

	ip := config.PrometheusIp
	port := config.PrometheusPort
	address := fmt.Sprintf("http://%s:%s", ip, port)
	client, err := api.NewClient(api.Config{
		Address: address,
	})
	if err != nil {
		logger2.Sugar().Errorf("Error creating clientgen: %v\n", err)
		return res, err
	}
	v1api := prom_v1.NewAPI(client)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// qps
	vector1 := prometheus.FetchQuery(ctx, v1api, constant.MetricsQps, nil)
	err = vector1.Err
	qps := float64(0)
	if err != nil {
		logger2.Sugar().Errorf("Error query qps: %v\n", err)
	} else {
		if vector1.Vector.Len() != 0 {
			qps = float64(vector1.Vector[0].Value)
		}
		res.Data["qps"] = qps
	}

	// total count
	vector3 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestTotalCount, nil)
	total := float64(0)
	if vector3.Err != nil {
		logger2.Sugar().Errorf("Error query total count: %v\n", err)
	} else {
		if vector3.Vector.Len() != 0 {
			total = float64(vector3.Vector[0].Value)
		}
		res.Data["total"] = total
	}

	// success count
	vector2 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestSuccessCount, nil)
	success := float64(0)
	if vector2.Err != nil {
		logger2.Sugar().Errorf("Error query success count: %v\n", err)
	} else {
		if vector2.Vector.Len() != 0 {
			success = float64(vector2.Vector[0].Value)
		}
		res.Data["total"] = success
	}

	// timeout count
	vector4 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestOutOfTimeCount, nil)
	timeout := float64(0)
	if vector4.Err != nil {
		logger2.Sugar().Errorf("Error query timeout count: %v\n", err)
	} else {
		if vector4.Vector.Len() != 0 {
			timeout = float64(vector4.Vector[0].Value)
		}
		res.Data["timeout"] = timeout
	}

	// address not found count
	vector5 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestAddressNotFount, nil)
	addrNotFound := float64(0)
	if vector5.Err != nil {
		logger2.Sugar().Errorf("Error query address not found count: %v\n", err)
	} else {
		if vector5.Vector.Len() != 0 {
			addrNotFound = float64(vector5.Vector[0].Value)
		}
		res.Data["addressNotFound"] = addrNotFound
	}

	// other exceptions count
	vector6 := prometheus.FetchQuery(ctx, v1api, constant.MetricsHttpRequestOtherException, nil)
	others := float64(0)
	if vector6.Err != nil {
		logger2.Sugar().Errorf("Error query othere exceptions count: %v\n", err)
	} else {
		if vector6.Vector.Len() != 0 {
			others = float64(vector6.Vector[0].Value)
		}
		res.Data["others"] = others
	}
	return res, nil
}

func (p *PrometheusServiceImpl) Metadata() (model.Metadata, error) {
	metadata := model.Metadata{}

	// versions
	versions, err := providerService.FindVersions()
	if err != nil {
		logger2.Error("Failed to parse versions!")
	}
	metadata.Versions = versions.Values()

	// protocols
	protocols, err := providerService.FindProtocols()
	if err != nil {
		logger2.Error("Failed to parse protocols!")
	}
	metadata.Protocols = protocols.Values()

	// centers
	metadata.Registry = config.RegistryCenter.GetURL().Location
	metadata.MetadataCenter = config.RegistryCenter.GetURL().Location
	metadata.ConfigCenter = config.RegistryCenter.GetURL().Location

	// rules
	rules, err := GetRules("", "*")
	if err != nil {
		return model.Metadata{}, err
	}
	keys := make([]string, 0, len(rules))
	for k := range rules {
		keys = append(keys, k)
	}
	metadata.Rules = keys

	return metadata, nil
}
