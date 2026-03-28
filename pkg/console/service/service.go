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

type serviceMethodCandidate struct {
	detail    *model.ServiceMethodDetailResp
	signature string
	method    *meshproto.Method
}

type serviceProviderMetadataLookupReq struct {
	ServiceName     string
	Group           string
	Version         string
	Mesh            string
	ProviderAppName string
}

type serviceMethodLookupReq struct {
	Metadata   serviceProviderMetadataLookupReq
	MethodName string
	Signature  string
}

type serviceMethodResolveReq struct {
	ServiceName string
	MethodName  string
	Signature   string
}

type resolvedServiceMethod struct {
	metadataList []*meshresource.ServiceProviderMetadataResource
	candidate    *serviceMethodCandidate
}

func newServiceProviderMetadataLookupReqFromBaseServiceReq(req model.BaseServiceReq) serviceProviderMetadataLookupReq {
	return serviceProviderMetadataLookupReq{
		ServiceName: req.ServiceName,
		Group:       req.Group,
		Version:     req.Version,
		Mesh:        req.Mesh,
	}
}

func newServiceMethodLookupReqFromServiceMethodDetailReq(req model.ServiceMethodDetailReq) serviceMethodLookupReq {
	return serviceMethodLookupReq{
		Metadata:   newServiceProviderMetadataLookupReqFromBaseServiceReq(req.BaseServiceReq),
		MethodName: req.MethodName,
		Signature:  req.Signature,
	}
}

func newServiceMethodLookupReqFromGenericInvokeReq(req model.ServiceGenericInvokeReq, providerAppName string) serviceMethodLookupReq {
	return serviceMethodLookupReq{
		Metadata: serviceProviderMetadataLookupReq{
			ServiceName:     req.ServiceName,
			Group:           req.Group,
			Version:         req.Version,
			Mesh:            req.Mesh,
			ProviderAppName: providerAppName,
		},
		MethodName: req.MethodName,
		Signature:  req.Signature,
	}
}

func GetServiceMethodNames(ctx consolectx.Context, req model.BaseServiceReq) ([]model.ServiceMethodSummaryResp, error) {
	metadataList, err := listServiceProviderMetadata(ctx, newServiceProviderMetadataLookupReqFromBaseServiceReq(req))
	if err != nil {
		return nil, err
	}

	return buildServiceMethodSummaries(metadataList), nil
}

func GetServiceMethodDetail(ctx consolectx.Context, req model.ServiceMethodDetailReq) (*model.ServiceMethodDetailResp, error) {
	resolvedMethod, err := resolveServiceMethod(ctx, newServiceMethodLookupReqFromServiceMethodDetailReq(req))
	if err != nil {
		return nil, err
	}

	detail := cloneServiceMethodDetailResp(resolvedMethod.candidate.detail)
	detail.Types = buildServiceMethodRelatedTypes(resolvedMethod.metadataList, resolvedMethod.candidate.method)
	return detail, nil
}

func resolveServiceMethod(ctx consolectx.Context, req serviceMethodLookupReq) (*resolvedServiceMethod, error) {
	metadataList, err := listServiceProviderMetadata(ctx, req.Metadata)
	if err != nil {
		return nil, err
	}

	candidate, err := resolveStructuredServiceMethodCandidate(buildServiceMethodCandidates(metadataList), serviceMethodResolveReq{
		ServiceName: req.Metadata.ServiceName,
		MethodName:  req.MethodName,
		Signature:   req.Signature,
	})
	if err != nil {
		return nil, err
	}

	return &resolvedServiceMethod{
		metadataList: metadataList,
		candidate:    candidate,
	}, nil
}

func buildServiceProviderLookupKey(req serviceProviderMetadataLookupReq) string {
	return req.ServiceName + constants.ColonSeparator + req.Version + constants.ColonSeparator + req.Group
}

func buildServiceProviderLookupIndexes(req serviceProviderMetadataLookupReq) map[string]string {
	return map[string]string{
		index.ByMeshIndex:                 req.Mesh,
		index.ByServiceProviderServiceKey: buildServiceProviderLookupKey(req),
	}
}

func listServiceProviderMetadata(ctx consolectx.Context, req serviceProviderMetadataLookupReq) ([]*meshresource.ServiceProviderMetadataResource, error) {
	return listServiceProviderMetadataByIndexes(ctx, req, buildServiceProviderLookupIndexes(req))
}

func listServiceProviderMetadataByIndexes(
	ctx consolectx.Context,
	req serviceProviderMetadataLookupReq,
	indexes map[string]string,
) ([]*meshresource.ServiceProviderMetadataResource, error) {
	if req.ProviderAppName != "" {
		indexes[index.ByServiceProviderAppName] = req.ProviderAppName
	}

	return manager.ListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		indexes,
	)
}

func buildServiceMethodSummaries(metadataList []*meshresource.ServiceProviderMetadataResource) []model.ServiceMethodSummaryResp {
	candidates := buildServiceMethodCandidates(metadataList)
	summaries := make([]model.ServiceMethodSummaryResp, 0, len(candidates))
	for _, candidate := range candidates {
		summaries = append(summaries, model.ServiceMethodSummaryResp{
			MethodName:     candidate.detail.MethodName,
			ParameterTypes: append([]string{}, candidate.detail.ParameterTypes...),
			Signature:      candidate.signature,
		})
	}
	return summaries
}

func buildServiceMethodCandidates(metadataList []*meshresource.ServiceProviderMetadataResource) []*serviceMethodCandidate {
	candidateByKey := make(map[string]*serviceMethodCandidate)

	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		for _, method := range metadata.Spec.Methods {
			candidate, ok := newStructuredServiceMethodCandidate(method)
			if !ok {
				continue
			}
			candidateByKey[serviceMethodKey(candidate.detail.MethodName, candidate.signature)] = candidate
		}
	}

	candidates := make([]*serviceMethodCandidate, 0, len(candidateByKey))
	for _, candidate := range candidateByKey {
		candidates = append(candidates, candidate)
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].detail.MethodName != candidates[j].detail.MethodName {
			return candidates[i].detail.MethodName < candidates[j].detail.MethodName
		}
		return candidates[i].signature < candidates[j].signature
	})
	return candidates
}

func findServiceMethodCandidate(candidates []*serviceMethodCandidate, req serviceMethodResolveReq) (*serviceMethodCandidate, bool) {
	if req.Signature != "" {
		for _, candidate := range candidates {
			if candidate.detail.MethodName == req.MethodName && candidate.signature == req.Signature {
				return candidate, false
			}
		}
		return nil, false
	}

	var candidateMatch *serviceMethodCandidate
	matchCount := 0
	for _, candidate := range candidates {
		if candidate.detail.MethodName != req.MethodName {
			continue
		}
		matchCount++
		if candidateMatch == nil {
			candidateMatch = candidate
		}
	}
	if matchCount > 1 {
		return nil, true
	}
	return candidateMatch, false
}

func resolveStructuredServiceMethodCandidate(candidates []*serviceMethodCandidate, req serviceMethodResolveReq) (*serviceMethodCandidate, error) {
	candidate, ambiguous := findServiceMethodCandidate(candidates, req)
	if ambiguous {
		return nil, bizerror.New(
			bizerror.InvalidArgument,
			fmt.Sprintf("multiple overloaded definitions found for method %s, please specify signature", req.MethodName),
		)
	}
	if candidate == nil {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("method %s not found for service %s", req.MethodName, req.ServiceName),
		)
	}
	if candidate.method == nil {
		return nil, bizerror.New(
			bizerror.InvalidArgument,
			fmt.Sprintf("structured metadata not found for method %s of service %s", req.MethodName, req.ServiceName),
		)
	}
	return candidate, nil
}

func newStructuredServiceMethodCandidate(method *meshproto.Method) (*serviceMethodCandidate, bool) {
	if method == nil {
		return nil, false
	}
	detail := toServiceMethodDetailResp(method)
	if detail.MethodName == "" {
		return nil, false
	}
	return &serviceMethodCandidate{
		detail:    detail,
		signature: buildServiceMethodSignature(method),
		method:    method,
	}, true
}

func serviceMethodKey(methodName, signature string) string {
	return methodName + "\x00" + signature
}

func toServiceMethodDetailResp(method *meshproto.Method) *model.ServiceMethodDetailResp {
	resp := &model.ServiceMethodDetailResp{
		MethodName:     strings.TrimSpace(method.GetName()),
		Signature:      buildServiceMethodSignature(method),
		ParameterTypes: normalizeServiceMethodParameterTypes(method.GetParameterTypes()),
		Parameters:     make([]model.ServiceMethodParameter, 0, len(method.GetParameters())),
		ReturnType:     strings.TrimSpace(method.GetReturnType()),
		Types:          []model.ServiceMethodTypeResp{},
	}
	for _, parameter := range method.GetParameters() {
		if parameter == nil {
			continue
		}
		resp.Parameters = append(resp.Parameters, model.ServiceMethodParameter{
			Name: strings.TrimSpace(parameter.GetName()),
			Type: strings.TrimSpace(parameter.GetType()),
		})
	}
	return resp
}

func cloneServiceMethodDetailResp(detail *model.ServiceMethodDetailResp) *model.ServiceMethodDetailResp {
	if detail == nil {
		return nil
	}
	cloned := &model.ServiceMethodDetailResp{
		MethodName:     detail.MethodName,
		Signature:      detail.Signature,
		ParameterTypes: append([]string{}, detail.ParameterTypes...),
		Parameters:     make([]model.ServiceMethodParameter, len(detail.Parameters)),
		ReturnType:     detail.ReturnType,
		Types:          cloneServiceMethodTypeResps(detail.Types),
	}
	copy(cloned.Parameters, detail.Parameters)
	return cloned
}

func cloneServiceMethodTypeResps(types []model.ServiceMethodTypeResp) []model.ServiceMethodTypeResp {
	cloned := make([]model.ServiceMethodTypeResp, 0, len(types))
	for _, typeResp := range types {
		cloned = append(cloned, model.ServiceMethodTypeResp{
			Type:       typeResp.Type,
			Properties: cloneStringMap(typeResp.Properties),
			Items:      append([]string{}, typeResp.Items...),
			Enums:      append([]string{}, typeResp.Enums...),
		})
	}
	return cloned
}

func cloneStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}

func buildServiceMethodRelatedTypes(metadataList []*meshresource.ServiceProviderMetadataResource, method *meshproto.Method) []model.ServiceMethodTypeResp {
	if method == nil {
		return []model.ServiceMethodTypeResp{}
	}

	typesByName := buildServiceMethodTypesByName(metadataList)
	visited := make(map[string]struct{})
	for _, parameterType := range method.GetParameterTypes() {
		collectServiceMethodRelatedTypeNames(typesByName, normalizeServiceMethodRelatedTypeName(parameterType), visited)
	}
	collectServiceMethodRelatedTypeNames(typesByName, normalizeServiceMethodRelatedTypeName(method.GetReturnType()), visited)

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

func buildServiceMethodTypesByName(metadataList []*meshresource.ServiceProviderMetadataResource) map[string]*meshproto.Type {
	typesByName := make(map[string]*meshproto.Type)
	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		for _, typeSpec := range metadata.Spec.Types {
			if typeSpec == nil {
				continue
			}
			typeName := strings.TrimSpace(typeSpec.GetType())
			if strutil.IsBlank(typeName) {
				continue
			}
			if _, exists := typesByName[typeName]; !exists {
				typesByName[typeName] = typeSpec
			}
		}
	}
	return typesByName
}

func collectServiceMethodRelatedTypeNames(typesByName map[string]*meshproto.Type, typeName string, visited map[string]struct{}) {
	typeName = normalizeServiceMethodRelatedTypeName(typeName)
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
		collectServiceMethodRelatedTypeNames(typesByName, itemType, visited)
	}
	for _, propertyType := range typeSpec.GetProperties() {
		collectServiceMethodRelatedTypeNames(typesByName, propertyType, visited)
	}
}

func normalizeServiceMethodRelatedTypeName(typeName string) string {
	typeName = strings.TrimSpace(typeName)
	for {
		elementType, isArrayType := splitGenericArrayType(typeName)
		if !isArrayType {
			return typeName
		}
		typeName = strings.TrimSpace(elementType)
	}
}

func toServiceMethodTypeResp(typeSpec *meshproto.Type) model.ServiceMethodTypeResp {
	return model.ServiceMethodTypeResp{
		Type:       strings.TrimSpace(typeSpec.GetType()),
		Properties: cloneStringMap(typeSpec.GetProperties()),
		Items:      append([]string{}, typeSpec.GetItems()...),
		Enums:      append([]string{}, typeSpec.GetEnums()...),
	}
}

func normalizeServiceMethodParameterTypes(parameterTypes []string) []string {
	normalized := make([]string, 0, len(parameterTypes))
	for _, parameterType := range parameterTypes {
		normalized = append(normalized, strings.TrimSpace(parameterType))
	}
	return normalized
}

func buildServiceMethodSignature(method *meshproto.Method) string {
	return strings.Join(normalizeServiceMethodParameterTypes(method.GetParameterTypes()), ",") +
		"->" + strings.TrimSpace(method.GetReturnType())
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
