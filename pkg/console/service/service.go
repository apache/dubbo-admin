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
	"strconv"
	"strings"

	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoveryutil "github.com/apache/dubbo-admin/pkg/common/util/discovery"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

// GetServiceTabDistribution get service distribution
func GetServiceTabDistribution(ctx consolectx.Context, req *model.ServiceTabDistributionReq) (*model.SearchPaginationResult, error) {
	indexes := map[string]string{
		index.ByServiceConsumerServiceName: req.ServiceName,
	}
	// for now, only support accurate name match
	if strutil.IsNotBlank(req.Keywords) {
		indexes[index.ByServiceConsumerAppName] = req.Keywords
	}
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		indexes,
		req.PageReq)
	if err != nil {
		logger.Errorf("get service consumer %s failed, cause: %v", req.ServiceName, err)
		return nil, bizerror.New(bizerror.InternalError, "get service consumer failed, please try again")
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return &model.SearchPaginationResult{
			List: []*meshresource.ServiceConsumerMetadataResourceList{},
			PageInfo: coremodel.Pagination{
				Total:      0,
				PageSize:   req.PageReq.PageSize,
				PageOffset: req.PageReq.PageOffset,
			},
		}, nil
	}
	appResKeys := slice.Map(pageData.Data, func(_ int, item *meshresource.ServiceConsumerMetadataResource) string {
		return coremodel.BuildResourceKey(req.Mesh, item.Spec.ConsumerAppName)
	})
	appResList, err := manager.GetByKeys[*meshresource.ApplicationResource](
		ctx.ResourceManager(), meshresource.ApplicationKind, appResKeys)
	if err != nil {
		logger.Errorf("get application list %v failed, cause: %s", appResKeys, err)
		return nil, err
	}
	respList := slice.Map(appResList, func(_ int, item *meshresource.ApplicationResource) model.ApplicationSearchResp {
		return model.ApplicationSearchResp{
			AppName:          item.Spec.Name,
			InstanceCount:    item.Spec.InstanceCount,
			DeployClusters:   []string{ctx.Config().Engine.Name},
			RegistryClusters: []string{discoveryutil.GetOrDefaultRegistryName(ctx.Config(), item.Mesh)},
		}
	})
	return &model.SearchPaginationResult{
		List:     respList,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchServices search services pageably
func SearchServices(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	if strutil.IsNotBlank(req.Keywords) {
		return SearchServicesByKeywords(ctx, req)
	}
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]string{
			index.ByMeshIndex: req.Mesh,
		},
		req.PageReq,
	)
	if err != nil {
		logger.Errorf("get service provider failed, cause: %v", err)
		return nil, err
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return nil, nil
	}
	serviceSearchResps := slice.Map(pageData.Data,
		func(_ int, item *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByProvider(item)
		})
	return &model.SearchPaginationResult{
		List:     serviceSearchResps,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchServicesByKeywords search services by keywords, for now only support accurate search
func SearchServicesByKeywords(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]string{
			index.ByMeshIndex:                  req.Mesh,
			index.ByServiceProviderServiceName: req.Keywords,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}
	searchRespList := slice.Map(pageData.Data,
		func(_ int, item *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByProvider(item)
		})
	return &model.SearchPaginationResult{
		List:     searchRespList,
		PageInfo: pageData.Pagination,
	}, nil
}

func ToServiceSearchRespByProvider(res *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName:     res.Spec.ServiceName,
		Group:           res.Spec.Group,
		Version:         res.Spec.Version,
		ProviderAppName: res.Spec.ProviderAppName,
	}
}

func ToServiceSearchRespByConsumer(res *meshresource.ServiceConsumerMetadataResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName:     res.Spec.ServiceName,
		Group:           res.Spec.Group,
		Version:         res.Spec.Version,
		ConsumerAppName: res.Spec.ConsumerAppName,
	}
}

func GetServiceTimeoutConfig(ctx consolectx.Context, req model.BaseServiceReq) (int32, error) {
	serviceConfiguratorName := req.ServiceKey() + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, serviceConfiguratorName, req.Mesh)
	if err != nil {
		logger.Errorf("get service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return 0, err
	}
	if res == nil || res.Spec == nil {
		logger.Infof("service configurator %s not found, return default timeout", serviceConfiguratorName)
		return constants.ServiceDefaultTimeout, nil
	}
	timeout := constants.ServiceDefaultTimeout
	slice.ForEachWithBreak(res.Spec.Configs, func(_ int, conf *meshproto.OverrideConfig) bool {
		t, found := getServiceTimeout(conf)
		if found {
			timeout = t
			return true
		}
		return found
	})
	return timeout, nil
}

func UpInsertServiceConfigTimeoutConfig(ctx consolectx.Context, req model.BaseServiceReq, timeout int32) error {
	serviceConfiguratorName := req.ServiceKey() + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, serviceConfiguratorName, req.Mesh)
	if err != nil {
		logger.Errorf("get service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return err
	}
	// if configurator doesn't exist
	if res == nil || res.Spec == nil {
		// if timeout config is default value, skip updating
		if timeout == constants.ServiceDefaultTimeout {
			logger.Infof("service configurator %s not found, timeout config is default value, "+
				"skip updating configurator", serviceConfiguratorName)
			return nil
		}
		// otherwise create a new configurator with timeout config
		res = meshresource.NewDynamicConfigResourceWithAttributes(serviceConfiguratorName, req.Mesh)
		res.Spec = &meshproto.DynamicConfig{
			Key:           req.ServiceName,
			Scope:         constants.ScopeService,
			ConfigVersion: constants.ConfiguratorVersionV3,
			Enabled:       true,
			Configs: []*meshproto.OverrideConfig{
				{
					Side:          constants.SideProvider,
					Parameters:    map[string]string{`timeout`: strconv.Itoa(int(timeout))},
					XGenerateByCp: true,
				},
			},
		}
		err = CreateConfigurator(ctx, res)
		if err != nil {
			logger.Errorf("create service configurator %s failed, cause: %v", serviceConfiguratorName, err)
			return err
		}
		return nil
	}
	// if configurator exists, match config one by one
	for _, conf := range res.Spec.Configs {
		oldTimeout, found := getServiceTimeout(conf)
		if !found {
			continue
		}
		// if timeout config is same as input, skip updating
		if oldTimeout == timeout {
			logger.Infof("service configurator %s already exists, timeout config is same as input, "+
				"skip updating configurator", serviceConfiguratorName)
			return nil
		}
		// if timeout config is different from input, update
		conf.Parameters[`timeout`] = strconv.Itoa(int(timeout))
		err := UpdateConfigurator(ctx, res)
		if err != nil {
			logger.Errorf("update service configurator %s failed, cause: %v", serviceConfiguratorName, err)
			return err
		}
		return nil
	}
	// if timeout config is not found, create a new one
	res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
		Side:          constants.SideProvider,
		Parameters:    map[string]string{`timeout`: strconv.Itoa(int(timeout))},
		XGenerateByCp: true,
	})
	err = UpdateConfigurator(ctx, res)
	if err != nil {
		logger.Errorf("update service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return err
	}
	return nil
}

func getServiceTimeout(conf *meshproto.OverrideConfig) (int32, bool) {
	if conf.Side == constants.SideProvider && conf.Parameters != nil && conf.Parameters[`timeout`] != "" {
		timeout, err := strconv.Atoi(conf.Parameters[`timeout`])
		if err == nil {
			return int32(timeout), true
		}
	}
	return 0, false
}

func GetServiceRetryConfig(ctx consolectx.Context, req model.BaseServiceReq) (int32, error) {
	serviceConfiguratorName := req.ServiceKey() + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, serviceConfiguratorName, req.Mesh)
	if err != nil {
		logger.Errorf("get service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return 0, err
	}
	if res == nil || res.Spec == nil {
		logger.Infof("service configurator %s not found, return default retries", serviceConfiguratorName)
		return constants.ServiceDefaultRetries, nil
	}
	retries := constants.ServiceDefaultRetries
	slice.ForEachWithBreak(res.Spec.Configs, func(_ int, conf *meshproto.OverrideConfig) bool {
		t, found := getServiceRetryTimes(conf)
		if found {
			retries = t
			return true
		}
		return found
	})
	return retries, nil
}

func UpInsertServiceRetryConfig(ctx consolectx.Context, req model.BaseServiceReq, retries int32) error {
	serviceConfiguratorName := req.ServiceKey() + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, serviceConfiguratorName, req.Mesh)
	if err != nil {
		logger.Errorf("get service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return err
	}
	// if configurator doesn't exist
	if res == nil || res.Spec == nil {
		// if retries config is default value, skip updating
		if retries == constants.ServiceDefaultRetries {
			logger.Infof("service configurator %s not found, retries config is default value, "+
				"skip updating configurator", serviceConfiguratorName)
			return nil
		}
		// otherwise create a new configurator with retries config
		res = meshresource.NewDynamicConfigResourceWithAttributes(serviceConfiguratorName, req.Mesh)
		res.Spec = &meshproto.DynamicConfig{
			Key:           req.ServiceName,
			Scope:         constants.ScopeService,
			ConfigVersion: constants.ConfiguratorVersionV3,
			Enabled:       true,
			Configs: []*meshproto.OverrideConfig{
				{
					Side:          constants.SideConsumer,
					Parameters:    map[string]string{`retries`: strconv.Itoa(int(retries))},
					XGenerateByCp: true,
				},
			},
		}
		if err := CreateConfigurator(ctx, res); err != nil {
			logger.Errorf("create service configurator %s failed, cause: %v", serviceConfiguratorName, err)
			return err
		}
		return nil
	}
	// if configurator exists, match config one by one
	for _, conf := range res.Spec.Configs {
		retryTimes, found := getServiceRetryTimes(conf)
		if !found {
			continue
		}
		// if retries config is same as input, skip updating
		if retryTimes == retries {
			logger.Infof("service configurator %s already exists, retries config is same as input, "+
				"skip updating configurator", serviceConfiguratorName)
			return nil
		}
		// if retries config is different from input, update
		conf.Parameters[`retries`] = strconv.Itoa(int(retries))
		if err := UpdateConfigurator(ctx, res); err != nil {
			logger.Errorf("update service configurator %s failed, cause: %v", serviceConfiguratorName, err)
			return err
		}
	}
	// no retry config found and retries is default value, skip updating
	if retries == constants.ServiceDefaultRetries {
		logger.Infof("service configurator %s already exists, retries config is default value, "+
			"skip updating configurator", serviceConfiguratorName)
		return nil
	}
	// otherwise create a new one
	res.Spec.Configs = append(res.Spec.Configs, &meshproto.OverrideConfig{
		Side:          constants.SideConsumer,
		Parameters:    map[string]string{`retries`: strconv.Itoa(int(retries))},
		XGenerateByCp: true,
	})
	if err = UpdateConfigurator(ctx, res); err != nil {
		logger.Errorf("update service configurator %s failed, cause: %v", serviceConfiguratorName, err)
		return err
	}
	return nil
}

func getServiceRetryTimes(conf *meshproto.OverrideConfig) (int32, bool) {
	if conf.Side == constants.SideConsumer && conf.Parameters != nil && conf.Parameters[`retries`] != "" {
		retries, err := strconv.Atoi(conf.Parameters[`retries`])
		if err == nil {
			return int32(retries), true
		}
	}
	return 0, false
}

func GetServiceRegionPriorityConfig(ctx consolectx.Context, req model.BaseServiceReq) (bool, error) {
	serviceConditionRuleName := req.ServiceKey() + constants.ConditionRuleDotSuffix
	res, err := GetConditionRule(ctx, serviceConditionRuleName, req.Mesh)
	if err != nil {
		logger.Errorf("get service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return true, err
	}
	if res == nil {
		return false, nil
	}
	openSameRegionPrior := false
	slice.ForEachWithBreak(res.Spec.Conditions, func(_ int, condition string) bool {
		openSameRegionPrior = isServiceSameRegion(condition)
		return openSameRegionPrior
	})
	return openSameRegionPrior, nil
}

func UpInsertServiceRegionPriorityConfig(ctx consolectx.Context, req model.BaseServiceReq, enabled bool) error {
	serviceConditionRuleName := req.ServiceKey() + constants.ConditionRuleSuffix
	res, err := GetConditionRule(ctx, serviceConditionRuleName, req.Mesh)
	if err != nil {
		logger.Errorf("get service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return err
	}
	// if condition rule doesn't exist
	if res == nil || res.Spec == nil {
		// if same region priority is needed to disable, skip updating
		if !enabled {
			logger.Infof("service condition rule %s not found, and same region priority is needed to disable, "+
				"skip updating condition rule", serviceConditionRuleName)
			return nil
		}
		// otherwise create a new condition rule
		res := meshresource.NewConditionRouteResourceWithAttributes(serviceConditionRuleName, req.Mesh)
		res.Spec = &meshproto.ConditionRoute{
			ConfigVersion: "v3.0",
			Priority:      0,
			Enabled:       true,
			Force:         false,
			Runtime:       true,
			Key:           req.ServiceName,
			Scope:         constants.ScopeService,
			Conditions:    []string{"=>region=$region"},
		}
		if err := CreateConditionRule(ctx, res); err != nil {
			logger.Errorf("create service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
			return err
		}
		return nil
	}
	// if condition rule exists, match condition one by one
	for i, condition := range res.Spec.Conditions {
		isSameRegion := isServiceSameRegion(condition)
		if !isSameRegion {
			continue
		}
		// if same region priority is needed to enable, and condition is already enabled, skip updating
		if enabled {
			logger.Infof("same region prior is already opened, skip updating service condition rule %s", serviceConditionRuleName)
			return nil
		}
		// otherwise we need to remove the condition and update condition rule
		res.Spec.Conditions = slice.Concat(res.Spec.Conditions[:i], res.Spec.Conditions[i+1:])
		if err := UpdateConditionRule(ctx, res); err != nil {
			logger.Errorf("update service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
			return err
		}
		return nil
	}
	// no same region priority found and region priority is needed to disable, skip updating
	if !enabled {
		logger.Infof("enabled is false and same region prior config is not exists, "+
			"skip updating service condition rule %s", serviceConditionRuleName)
		return nil
	}
	// otherwise create a new condition
	res.Spec.Conditions = append(res.Spec.Conditions, "=>region=$region")
	if err := UpdateConditionRule(ctx, res); err != nil {
		logger.Errorf("update service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return err
	}
	return nil
}

func isServiceSameRegion(condition string) bool {
	c := strings.TrimSpace(condition)
	return strings.Contains(c, "=>region=$region")
}

func GetServiceArgumentRouteConfig(ctx consolectx.Context, req model.BaseServiceReq) (*model.ServiceArgumentRoute, error) {
	serviceConditionRuleName := req.ServiceKey() + constants.ConditionRuleDotSuffix
	rawRes, err := GetConditionRule(ctx, serviceConditionRuleName, req.Mesh)
	if err != nil {
		logger.Errorf("get service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return nil, err
	}
	if rawRes == nil || rawRes.Spec == nil {
		return nil, nil
	}
	argumentRoutes := slice.Map(rawRes.Spec.Conditions, func(index int, condition string) model.ServiceArgument {
		return model.ParseConditionExpression(condition)
	})
	return &model.ServiceArgumentRoute{
		Routes: argumentRoutes,
	}, nil
}

func UpInsertServiceArgumentRouteConfig(ctx consolectx.Context, req model.BaseServiceReq, route model.ServiceArgumentRoute) error {
	serviceConditionRuleName := req.ServiceKey() + constants.ConditionRuleDotSuffix
	conditionRouteRes, err := GetConditionRule(ctx, serviceConditionRuleName, req.Mesh)
	if err != nil {
		logger.Errorf("get service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return err
	}
	if conditionRouteRes == nil {
		conditionRouteRes = meshresource.NewConditionRouteResourceWithAttributes(serviceConditionRuleName, req.Mesh)
		conditionRouteRes.Spec.Conditions = make([]string, 0)
	}
	conditions := slice.Filter(conditionRouteRes.Spec.Conditions, func(index int, condition string) bool {
		return !isArgumentRoute(condition)
	})
	conditions = slice.Concat(conditions,
		slice.Map(route.Routes, func(index int, item model.ServiceArgument) string {
			return item.ToExpression()
		}))
	conditionRouteRes.Spec = &meshproto.ConditionRoute{
		ConfigVersion: "v3.0",
		Priority:      0,
		Enabled:       true,
		Force:         false,
		Runtime:       true,
		Key:           req.ServiceName,
		Scope:         constants.ScopeService,
		Conditions:    conditions,
	}
	if err = UpdateConditionRule(ctx, conditionRouteRes); err != nil {
		logger.Errorf("create service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
		return err
	}
	return nil
}

// isArgumentRoute judge whether the condition is argument route
func isArgumentRoute(condition string) bool {
	if strings.Contains(condition, "method") {
		return true
	}
	return false
}
