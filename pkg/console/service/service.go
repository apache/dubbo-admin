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
	"sort"
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
	conditions := []index.IndexCondition{
		{IndexName: index.ByServiceConsumerServiceName, Value: req.ServiceName, Operator: index.Equals},
	}
	// for now, only support accurate name match
	if strutil.IsNotBlank(req.Keywords) {
		conditions = append(conditions, index.IndexCondition{
			IndexName: index.ByServiceConsumerAppName,
			Value:     req.Keywords,
			Operator:  index.Equals,
		})
	}
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		conditions,
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
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceResource](
		ctx.ResourceManager(),
		meshresource.ServiceKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
		},
		req.PageReq,
	)
	if err != nil {
		logger.Errorf("get service provider failed, cause: %v", err)
		return nil, err
	}
	if pageData.Data == nil || len(pageData.Data) == 0 {
		return &model.SearchPaginationResult{
			List:     []*model.ServiceSearchResp{},
			PageInfo: pageData.Pagination,
		}, nil
	}
	serviceSearchResps := slice.Map(pageData.Data,
		func(_ int, item *meshresource.ServiceResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByService(item)
		})
	return &model.SearchPaginationResult{
		List:     serviceSearchResps,
		PageInfo: pageData.Pagination,
	}, nil
}

// SearchServicesByKeywords search services by keywords with prefix matching
func SearchServicesByKeywords(ctx consolectx.Context, req *model.ServiceSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceResource](
		ctx.ResourceManager(),
		meshresource.ServiceKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
			{IndexName: index.ByServiceName, Value: req.Keywords, Operator: index.HasPrefix},
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}
	searchRespList := slice.Map(
		pageData.Data,
		func(_ int, item *meshresource.ServiceResource) *model.ServiceSearchResp {
			return ToServiceSearchRespByService(item)
		},
	)
	return &model.SearchPaginationResult{
		List:     searchRespList,
		PageInfo: pageData.Pagination,
	}, nil
}

func ToServiceSearchRespByService(res *meshresource.ServiceResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName: res.Spec.Name,
		Group:       res.Spec.Group,
		Version:     res.Spec.Version,
	}
}

func ToServiceSearchRespByProvider(res *meshresource.ServiceProviderMetadataResource) *model.ServiceSearchResp {
	return &model.ServiceSearchResp{
		ServiceName: res.Spec.ServiceName,
		Group:       res.Spec.Group,
		Version:     res.Spec.Version,
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

func GetServiceMethodNames(ctx consolectx.Context, req model.BaseServiceReq) ([]model.ServiceMethodSummaryResp, error) {
	metadataList, err := listProviderMeta(ctx, req)
	if err != nil {
		return nil, err
	}

	return buildMethodSummaries(metadataList), nil
}

func GetServiceMethodDetail(ctx consolectx.Context, req model.ServiceMethodDetailReq) (*model.ServiceMethodDetailResp, error) {
	metadataList, err := listProviderMeta(ctx, req.BaseServiceReq)
	if err != nil {
		return nil, err
	}
	method := findMethod(metadataList, req.MethodName, req.Signature)
	if method == nil {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("method %s not found for service %s", req.MethodName, req.ServiceName),
		)
	}

	detail := toMethodDetail(method)
	detail.Types = buildRelatedTypes(metadataList, method)
	return detail, nil
}

// providerIndexes defines the canonical indexes for provider metadata
func providerIndexes(req model.BaseServiceReq) []index.IndexCondition {
	return []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
		{IndexName: index.ByServiceProviderServiceKey, Value: req.ServiceKey(), Operator: index.Equals},
	}
}

// listProviderMeta loads provider metadata by the canonical mesh + serviceKey indexes.
func listProviderMeta(ctx consolectx.Context, req model.BaseServiceReq) ([]*meshresource.ServiceProviderMetadataResource, error) {
	return manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		providerIndexes(req),
	)
}

func buildMethodSummaries(metadataList []*meshresource.ServiceProviderMetadataResource) []model.ServiceMethodSummaryResp {
	methods := collectMethods(metadataList)
	summaries := make([]model.ServiceMethodSummaryResp, 0, len(methods))
	for _, method := range methods {
		detail := toMethodDetail(method)
		summaries = append(summaries, model.ServiceMethodSummaryResp{
			MethodName:     detail.MethodName,
			ParameterTypes: detail.ParameterTypes,
			Signature:      detail.Signature,
		})
	}
	return summaries
}

// collectMethods flattens provider metadata into a unique, sorted method list.
func collectMethods(metadataList []*meshresource.ServiceProviderMetadataResource) []*meshproto.Method {
	methodByKey := make(map[string]*meshproto.Method)

	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		for _, method := range metadata.Spec.Methods {
			methodName := method.GetName()
			if method == nil || methodName == "" {
				continue
			}
			methodByKey[methodKey(methodName, methodSig(method))] = method
		}
	}

	methods := make([]*meshproto.Method, 0, len(methodByKey))
	for _, method := range methodByKey {
		methods = append(methods, method)
	}
	sort.Slice(methods, func(i, j int) bool {
		leftName := methods[i].GetName()
		rightName := methods[j].GetName()
		if leftName != rightName {
			return leftName < rightName
		}
		return methodSig(methods[i]) < methodSig(methods[j])
	})
	return methods
}

// findMethod scans the current metadata snapshot for one exact method signature.
func findMethod(metadataList []*meshresource.ServiceProviderMetadataResource, methodName string, signature string) *meshproto.Method {
	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		for _, method := range metadata.Spec.Methods {
			if method == nil {
				continue
			}
			if method.GetName() == methodName && methodSig(method) == signature {
				return method
			}
		}
	}
	return nil
}

func methodKey(methodName, signature string) string {
	return methodName + "\x00" + signature
}

// toMethodDetail projects proto metadata into the API response shape.
func toMethodDetail(method *meshproto.Method) *model.ServiceMethodDetailResp {
	resp := &model.ServiceMethodDetailResp{
		MethodName:     method.GetName(),
		Signature:      methodSig(method),
		ParameterTypes: method.GetParameterTypes(),
		Parameters:     make([]model.ServiceMethodParameter, 0, len(method.GetParameters())),
		ReturnType:     method.GetReturnType(),
		Types:          []model.ServiceMethodTypeResp{},
	}
	for _, parameter := range method.GetParameters() {
		if parameter == nil {
			continue
		}
		resp.Parameters = append(resp.Parameters, model.ServiceMethodParameter{
			Name: parameter.GetName(),
			Type: parameter.GetType(),
		})
	}
	return resp
}

// buildRelatedTypes walks parameter and return types against the current metadata snapshot.
func buildRelatedTypes(metadataList []*meshresource.ServiceProviderMetadataResource, method *meshproto.Method) []model.ServiceMethodTypeResp {
	if method == nil {
		return []model.ServiceMethodTypeResp{}
	}

	// Index all declared types once, then resolve only the subset reachable from this method.
	typesByName := buildTypeMap(metadataList)
	visited := make(map[string]struct{})
	for _, parameterType := range method.GetParameterTypes() {
		collectRelatedTypes(typesByName, parameterType, visited)
	}
	collectRelatedTypes(typesByName, method.GetReturnType(), visited)

	// Sort for stable API output and deterministic tests.
	typeNames := make([]string, 0, len(visited))
	for typeName := range visited {
		typeNames = append(typeNames, typeName)
	}
	sort.Strings(typeNames)

	resp := make([]model.ServiceMethodTypeResp, 0, len(typeNames))
	for _, typeName := range typeNames {
		typeSpec, ok := typesByName[typeName]
		if !ok {
			continue
		}
		resp = append(resp, toServiceMethodTypeResp(typeSpec))
	}
	return resp
}

// buildTypeMap keeps the first declaration for each type name in the current metadata snapshot.
func buildTypeMap(metadataList []*meshresource.ServiceProviderMetadataResource) map[string]*meshproto.Type {
	typesByName := make(map[string]*meshproto.Type)
	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		for _, typeSpec := range metadata.Spec.Types {
			if typeSpec == nil {
				continue
			}
			typeName := typeSpec.GetType()
			if typeName == "" {
				continue
			}
			if _, exists := typesByName[typeName]; !exists {
				typesByName[typeName] = typeSpec
			}
		}
	}
	return typesByName
}

// collectRelatedTypes follows nested item/property references and uses visited to stop cycles.
func collectRelatedTypes(typesByName map[string]*meshproto.Type, typeName string, visited map[string]struct{}) {
	if strutil.IsBlank(typeName) {
		return
	}
	typeSpec, ok := typesByName[typeName]
	if !ok {
		return
	}
	if _, exists := visited[typeName]; exists {
		return
	}
	visited[typeName] = struct{}{}

	for _, itemType := range typeSpec.GetItems() {
		collectRelatedTypes(typesByName, itemType, visited)
	}
	for _, propertyType := range typeSpec.GetProperties() {
		collectRelatedTypes(typesByName, propertyType, visited)
	}
}

func toServiceMethodTypeResp(typeSpec *meshproto.Type) model.ServiceMethodTypeResp {
	return model.ServiceMethodTypeResp{
		Type:       typeSpec.GetType(),
		Properties: typeSpec.GetProperties(),
		Items:      typeSpec.GetItems(),
		Enums:      typeSpec.GetEnums(),
	}
}

func methodSig(method *meshproto.Method) string {
	return strings.Join(method.GetParameterTypes(), ",") +
		"->" + method.GetReturnType()
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
	serviceConditionRuleName := req.ServiceKey() + constants.ConditionRuleDotSuffix
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
	shouldCreate := conditionRouteRes == nil
	if conditionRouteRes == nil {
		conditionRouteRes = meshresource.NewConditionRouteResourceWithAttributes(serviceConditionRuleName, req.Mesh)
		conditionRouteRes.Spec.Conditions = make([]string, 0)
	} else if conditionRouteRes.Spec == nil {
		conditionRouteRes.Spec = &meshproto.ConditionRoute{Conditions: make([]string, 0)}
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
		Key:           req.ServiceKey(),
		Scope:         constants.ScopeService,
		Conditions:    conditions,
	}
	if shouldCreate {
		err = CreateConditionRule(ctx, conditionRouteRes)
	} else {
		err = UpdateConditionRule(ctx, conditionRouteRes)
	}
	if err != nil {
		logger.Errorf("upsert service condition rule %s failed, cause: %v", serviceConditionRuleName, err)
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

func GetServiceDetail(ctx consolectx.Context, req *model.ServiceDetailReq) (*model.ServiceDetailResp, error) {
	serviceKey := coremodel.BuildResourceKey(req.Mesh, meshresource.BuildServiceIdentityKey(req.ServiceName, req.Version, req.Group))
	serviceRes, exists, err := manager.GetByKey[*meshresource.ServiceResource](
		ctx.ResourceManager(),
		meshresource.ServiceKind,
		serviceKey,
	)
	if err != nil {
		logger.Errorf("get service detail failed, serviceKey: %s, cause: %v", serviceKey, err)
		return nil, err
	}
	if !exists || serviceRes.Spec == nil {
		return nil, bizerror.New(bizerror.NotFoundError, "service not found")
	}

	return &model.ServiceDetailResp{
		Language: serviceRes.Spec.Language,
		Methods:  serviceRes.Spec.Methods,
	}, nil
}

// GraphServices builds a service dependency graph for the given service key.
//
// It gathers both provider and consumer metadata for serviceKey and creates
// graph nodes/edges where:
//   - application nodes are marked as provider/consumer
//   - service node is the target/subject service
//   - edges describe provide/consume relationships
//
// This API is used by topology view to visualize service-level dependencies.
func GraphServices(ctx consolectx.Context, req *model.ServiceGraphReq) (*model.GraphData, error) {
	serviceKey := req.ServiceKey()

	providers, err := manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
			{IndexName: index.ByServiceProviderServiceKey, Value: serviceKey, Operator: index.Equals},
		},
	)
	if err != nil {
		logger.Errorf("get service providers for mesh %s, serviceKey %s failed, cause: %v", req.Mesh, serviceKey, err)
		return nil, bizerror.New(bizerror.InternalError, "get service providers failed, please try again")
	}

	if len(providers) == 0 {
		logger.Errorf("no providers found for service %s in mesh %s", serviceKey, req.Mesh)
		return nil, bizerror.New(bizerror.NotFoundError, "no providers found for this service")
	}

	// Nodes for this graph: provider apps, service itself, consumer apps.
	// Edges represent provider->service and consumer->service relationships.

	consumers, err := manager.ListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
			{IndexName: index.ByServiceConsumerServiceKey, Value: serviceKey, Operator: index.Equals},
		})
	if err != nil {
		logger.Errorf("get service consumers for mesh %s, serviceKey %s failed, cause: %v", req.Mesh, serviceKey, err)
		return nil, bizerror.New(bizerror.InternalError, "get service consumers failed, please try again")
	}

	nodes := make([]model.GraphNode, 0)
	edges := make([]model.GraphEdge, 0)

	// use struct{} as a zero-size value for a lightweight deduplication set
	// this prevents duplicate application nodes when multiple providers are recorded.
	providerAppSet := make(map[string]struct{})
	for _, provider := range providers {
		if provider.Spec == nil {
			continue
		}
		if _, ok := providerAppSet[provider.Spec.ProviderAppName]; !ok {
			providerAppSet[provider.Spec.ProviderAppName] = struct{}{}
			nodes = append(nodes, model.GraphNode{
				ID:    provider.Spec.ProviderAppName,
				Label: provider.Spec.ProviderAppName,
				Type:  "application",
				Rule:  "provider",
				Data:  nil,
			})
		}
	}

	nodes = append(nodes, model.GraphNode{
		ID:    serviceKey,
		Label: serviceKey,
		Type:  "service",
		Rule:  "",
		Data:  nil,
	})

	consumerAppSet := make(map[string]struct{})
	for _, consumer := range consumers {
		if consumer.Spec == nil {
			continue
		}
		if _, ok := consumerAppSet[consumer.Spec.ConsumerAppName]; !ok {
			consumerAppSet[consumer.Spec.ConsumerAppName] = struct{}{}
			nodes = append(nodes, model.GraphNode{
				ID:    consumer.Spec.ConsumerAppName,
				Label: consumer.Spec.ConsumerAppName,
				Type:  "application",
				Rule:  "consumer",
				Data:  nil,
			})
		}
	}

	// Connect provider applications to service node.
	for providerApp := range providerAppSet {
		edges = append(edges, model.GraphEdge{
			Source: serviceKey,
			Target: providerApp,
			Data: map[string]interface{}{
				"type": "provides",
			},
		})
	}

	// Connect consumer applications to service node.
	for consumerApp := range consumerAppSet {
		edges = append(edges, model.GraphEdge{
			Source: consumerApp,
			Target: serviceKey,
			Data: map[string]interface{}{
				"type": "consumes",
			},
		})
	}

	return &model.GraphData{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func sortedKeys(items map[string]struct{}) []string {
	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
