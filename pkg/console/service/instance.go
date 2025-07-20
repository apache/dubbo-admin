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
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	corers "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func BannerSearchIp(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &corers.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByFilterFunc(searchByIp(req.Keywords)), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
		return nil, err
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		list[i] = model.NewSearchInstanceResp()
		list[i] = list[i].FromDataplaneResource(item)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func searchByIp(ip string) store.ListFilterFunc {
	return func(rs coremodel.Resource) bool {
		// make sure that the resource is of type mesh.DataplaneResource
		if dp, ok := rs.(*mesh.DataplaneResource); ok {
			return dp.GetIP() == ip
		}
		return false
	}
}

func BannerSearchInstances(ctx consolectx.Context, req *model.SearchReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
		return nil, err
	}

	res := model.NewSearchPaginationResult()
	list := make([]*model.SearchInstanceResp, len(dataplaneList.Items))
	for i, item := range dataplaneList.Items {
		list[i] = model.NewSearchInstanceResp()
		list[i] = list[i].FromDataplaneResource(item)
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func SearchInstances(ctx consolectx.Context, req *model.SearchInstanceReq) (*model.SearchPaginationResult, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if req.Keywords == "" {
		if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	} else {
		if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByNameContains(req.Keywords), store.ListByPage(req.PageSize, strconv.Itoa(req.PageOffset))); err != nil {
			return nil, err
		}
	}

	res := model.NewSearchPaginationResult()
	var list []*model.SearchInstanceResp
	for _, item := range dataplaneList.Items {
		if strings.Split(item.Meta.GetName(), constant.KeySeparator)[1] == "0" {
			continue
		}
		list = append(list, model.NewSearchInstanceResp().FromDataplaneResource(item))
	}

	res.List = list
	res.PageInfo = &dataplaneList.Pagination

	return res, nil
}

func GetInstanceDetail(ctx consolectx.Context, req *model.InstanceDetailReq) ([]*model.InstanceDetailResp, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}

	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByNameContains(req.InstanceName)); err != nil {
		return nil, err
	}

	instMap := make(map[string]*model.InstanceDetail)
	for _, dataplane := range dataplaneList.Items {

		// instName := dataplane.Meta.GetLabels()[mesh_proto.InstanceTag]//This tag is "" in universal mode
		instName := dataplane.Meta.GetName()
		var instanceDetail *model.InstanceDetail
		if _, ok := instMap[instName]; ok {
			// found previously recorded instance detail in instMap
			// the detail should be merged with the new instance detail
			instanceDetail = instMap[instName]
		} else {
			// the instance information appears for the 1st time
			instanceDetail = model.NewInstanceDetail()
		}
		instanceDetail.Merge(dataplane) // convert dataplane info to instance detail
		instMap[instName] = instanceDetail
	}

	resp := make([]*model.InstanceDetailResp, 0, len(instMap))
	for _, instDetail := range instMap {
		respItem := &model.InstanceDetailResp{}
		resp = append(resp, respItem.FromInstanceDetail(instDetail))
	}
	return resp, nil
}

func GetInstanceMetrics(ctx consolectx.Context, req *model.MetricsReq) ([]*model.MetricsResp, error) {
	manager := ctx.ResourceManager()
	dataplaneList := &mesh.DataplaneResourceList{}
	if err := manager.List(ctx.AppContext(), dataplaneList, store.ListByNameContains(req.InstanceName)); err != nil {
		return nil, err
	}
	instMap := make(map[string]*model.InstanceDetail)
	resp := make([]*model.MetricsResp, 0)
	for _, dataplane := range dataplaneList.Items {
		instName := dataplane.Meta.GetName()
		var instanceDetail *model.InstanceDetail
		if detail, ok := instMap[instName]; ok {
			instanceDetail = detail
		} else {
			instanceDetail = model.NewInstanceDetail()
		}
		instanceDetail.Merge(dataplane)
		metrics, err := fetchMetricsData(dataplane.GetIP(), 22222)
		if err != nil {
			continue
		}
		metricsResp := &model.MetricsResp{
			InstanceName: instName,
			Metrics:      metrics,
		}
		resp = append(resp, metricsResp)
	}
	return resp, nil
}

func fetchMetricsData(ip string, port int) ([]model.Metric, error) {
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
