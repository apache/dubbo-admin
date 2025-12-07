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

package service

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/duke-git/lancet/v2/strutil"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// BannerSearchIp
// TODO: implement me
func BannerSearchIp(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	return nil, nil
}

// BannerSearchInstances
// TODO: implement me
func BannerSearchInstances(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	// TODO: implement me
	return nil, nil
}

func SearchInstances(ctx consolectx.Context, req *model.SearchInstanceReq) (*model.SearchPaginationResult, error) {
	var pageData *coremodel.PageData[*meshresource.InstanceResource]
	var err error
	if strutil.IsBlank(req.Keywords) {
		pageData, err = manager.PageListByIndexes[*meshresource.InstanceResource](
			ctx.ResourceManager(),
			meshresource.InstanceKind,
			map[string]string{
				index.ByMeshIndex: req.Mesh,
			},
			req.PageReq)
		if err != nil {
			return nil, err
		}
	} else {
		pageData, err = manager.PageSearchResourceByConditions[*meshresource.InstanceResource](
			ctx.ResourceManager(),
			meshresource.InstanceKind,
			[]string{"ip=" + req.Keywords},
			req.PageReq,
		)
		if err != nil {
			return nil, err
		}
	}

	resp := model.NewSearchPaginationResult()
	var list []*model.SearchInstanceResp
	for _, item := range pageData.Data {
		list = append(list, model.NewSearchInstanceResp().FromInstanceResource(item))
	}
	resp.List = list
	resp.PageInfo = pageData.Pagination
	return resp, nil
}

func GetInstanceDetail(ctx consolectx.Context, req *model.InstanceDetailReq) (*model.InstanceDetailResp, error) {
	res, _, err := manager.GetByKey[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		coremodel.BuildResourceKey(req.Mesh, req.InstanceName),
	)
	if err != nil {
		return nil, err
	}

	resp := model.FromInstanceResource(res)
	return resp, nil
}

func GetInstanceMetrics(ctx consolectx.Context, req *model.MetricsReq) ([]*model.MetricsResp, error) {
	res, _, err := manager.GetByKey[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		coremodel.BuildResourceKey(req.Mesh, req.InstanceName),
	)
	if err != nil {
		return nil, err
	}
	instance := res.Spec
	metricsData, err := fetchMetricsData(instance.Ip, instance.QosPort)
	if err != nil {
		return nil, err
	}

	metricsResp := &model.MetricsResp{
		InstanceName: instance.Name,
		Metrics:      metricsData,
	}

	return []*model.MetricsResp{metricsResp}, nil
}

func fetchMetricsData(ip string, port int64) ([]model.Metric, error) {
	url := fmt.Sprintf("http://%s:%d/metrics", ip, port)
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	metrics, err := parsePrometheusData(string(body))
	if err != nil {
		return nil, err
	}
	return metrics, nil
}

// parsePrometheusData parses Prometheus text format data and converts it to a slice of Metrics.
func parsePrometheusData(data string) ([]model.Metric, error) {
	var metrics []model.Metric
	lines := strings.Split(data, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, " ")
		if len(parts) != 2 {
			continue
		}

		metricPart := parts[0]
		valuePart := parts[1]

		// Extract the metric name and labels
		nameAndLabels := strings.SplitN(metricPart, "{", 2)
		if len(nameAndLabels) != 2 {
			continue
		}

		name := nameAndLabels[0]
		labelsPart := strings.TrimSuffix(nameAndLabels[1], "}")

		labels := make(map[string]string)
		for _, label := range strings.Split(labelsPart, ",") {
			if label == "" {
				continue
			}
			labelParts := strings.SplitN(label, "=", 2)
			if len(labelParts) == 2 {
				labels[labelParts[0]] = strings.Trim(labelParts[1], `"`)
			}
		}

		// Parse the value
		var value float64
		fmt.Sscanf(valuePart, "%f", &value)

		metrics = append(metrics, model.Metric{
			Name:   name,
			Labels: labels,
			Value:  value,
		})
	}

	return metrics, nil
}
