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
	"net/url"
	"strings"

	"github.com/duke-git/lancet/v2/maputil"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func GetAppDashboard(baseURL *url.URL, req *model.AppDashboardReq) (string, error) {
	if baseURL == nil {
		return "", bizerror.New(bizerror.NotFoundError, "grafana url is not configured")
	}
	variables := map[string]string{
		"var-application": req.AppName,
	}
	fullURL := concatURLWithQueryVars(baseURL, variables)
	return fullURL, nil
}

func GetServiceDashboard(baseURL *url.URL, req *model.ServiceDashboardReq) (string, error) {
	if baseURL == nil {
		return "", bizerror.New(bizerror.NotFoundError, "grafana url is not configured")
	}
	variables := map[string]string{
		"var-service": req.ServiceName,
	}
	return concatURLWithQueryVars(baseURL, variables), nil
}

// GetServiceTraceDashboard requires an unambiguous provider application.
func GetServiceTraceDashboard(ctx consolectx.Context, baseURL *url.URL, req *model.ServiceDashboardReq) (string, error) {
	if baseURL == nil {
		return "", bizerror.New(bizerror.NotFoundError, "grafana url is not configured")
	}
	metadata, err := listProviderMeta(ctx, model.BaseServiceReq{
		Mesh: req.Mesh, ServiceName: req.ServiceName, Group: req.Group, Version: req.Version,
	})
	if err != nil {
		return "", err
	}
	applications := make(map[string]struct{})
	for _, provider := range metadata {
		if provider == nil || provider.Spec == nil || strings.TrimSpace(provider.Spec.ProviderAppName) == "" {
			continue
		}
		applications[provider.Spec.ProviderAppName] = struct{}{}
	}
	if len(applications) == 0 {
		return "", bizerror.New(bizerror.NotFoundError, fmt.Sprintf("no provider application found for service %s", req.ServiceName))
	}
	if len(applications) != 1 {
		return "", bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("multiple provider applications found for service %s", req.ServiceName))
	}
	var application string
	for name := range applications {
		application = name
	}
	traceURL := *baseURL
	return concatURLWithQueryVars(&traceURL, map[string]string{
		"var-application": application,
		"var-service":     req.ServiceName,
	}), nil
}

func GetInstanceDashboard(ctx consolectx.Context, baseURL *url.URL, req *model.InstanceDashboardReq) (string, error) {
	if baseURL == nil {
		return "", bizerror.New(bizerror.NotFoundError, "grafana url is not configured")
	}
	variables, err := GetInstanceDashboardVariables(ctx, req)
	if err != nil {
		return "", err
	}
	return concatURLWithQueryVars(baseURL, variables), nil
}

func GetInstanceDashboardVariables(ctx consolectx.Context, req *model.InstanceDashboardReq) (map[string]string, error) {
	res, err := getInstanceDashboardResource(ctx, req)
	if err != nil {
		return nil, err
	}
	var instanceValue string
	if res.Spec.QosPort != 0 {
		instanceValue = fmt.Sprintf("%s:%d", res.Spec.Ip, res.Spec.QosPort)
	} else {
		instanceValue = fmt.Sprintf("%s:%d", res.Spec.Ip, 22222)
	}

	return map[string]string{
		"var-application": res.Spec.AppName,
		"var-instance":    instanceValue,
	}, nil
}

func GetInstanceTraceDashboard(ctx consolectx.Context, baseURL *url.URL, req *model.InstanceDashboardReq) (string, error) {
	if baseURL == nil {
		return "", bizerror.New(bizerror.NotFoundError, "grafana url is not configured")
	}
	variables, err := GetInstanceTraceDashboardVariables(ctx, req)
	if err != nil {
		return "", err
	}
	traceURL := *baseURL
	return concatURLWithQueryVars(&traceURL, variables), nil
}

func GetInstanceTraceDashboardVariables(ctx consolectx.Context, req *model.InstanceDashboardReq) (map[string]string, error) {
	res, err := getInstanceDashboardResource(ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"var-application": res.Spec.AppName,
		"var-instance":    res.Spec.Ip,
	}, nil
}

func getInstanceDashboardResource(ctx consolectx.Context, req *model.InstanceDashboardReq) (*meshresource.InstanceResource, error) {
	resKey := coremodel.BuildResourceKey(req.Mesh, req.InstanceName)
	res, exists, err := manager.GetByKey[*meshresource.InstanceResource](ctx.ResourceManager(), meshresource.InstanceKind, resKey)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, bizerror.New(bizerror.NotFoundError,
			fmt.Sprintf("instance %s not found", req.InstanceName))
	}
	return res, nil
}

func concatURLWithQueryVars(baseURL *url.URL, vars map[string]string) string {
	q := baseURL.Query()
	maputil.ForEach(vars, func(key string, value string) {
		q.Set(key, value)
	})
	baseURL.RawQuery = q.Encode()
	return baseURL.String()
}
