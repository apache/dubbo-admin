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
	"strconv"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/duke-git/lancet/v2/strutil"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

func GetApplicationDetail(ctx consolectx.Context, req *model.ApplicationDetailReq) (*model.ApplicationDetailResp, error) {
	instanceResources, err := manager.ListByIndexes[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		map[string]string{
			index.ByMeshIndex:            req.Mesh,
			index.ByInstanceAppNameIndex: req.AppName,
		},
	)
	if err != nil {
		return nil, err
	}

	applicationDetail := model.NewApplicationDetail()
	for _, instanceRes := range instanceResources {
		applicationDetail.MergeInstance(instanceRes)
	}

	respItem := &model.ApplicationDetailResp{
		AppName: req.AppName,
	}
	respItem = respItem.FromApplicationDetail(applicationDetail)

	return respItem, nil
}

func GetAppInstanceInfo(ctx consolectx.Context, req *model.ApplicationTabInstanceInfoReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		map[string]string{
			index.ByMeshIndex:            req.Mesh,
			index.ByInstanceAppNameIndex: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	list := slice.Map[*meshresource.InstanceResource, *model.AppInstanceInfoResp](pageData.Data,
		func(_ int, item *meshresource.InstanceResource) *model.AppInstanceInfoResp {
			return buildAppInstanceInfoResp(item)
		})
	searchResult := &model.SearchPaginationResult{
		List:     list,
		PageInfo: pageData.Pagination,
	}
	return searchResult, nil
}

func buildAppInstanceInfoResp(instanceRes *meshresource.InstanceResource) *model.AppInstanceInfoResp {
	instance := instanceRes.Spec
	resp := &model.AppInstanceInfoResp{}
	resp.Name = instance.Name
	resp.AppName = instance.AppName
	resp.CreateTime = instance.CreateTime
	resp.DeployState = instance.DeployState
	resp.DeployClusters = ""
	resp.IP = instance.Ip
	resp.Labels = instance.Tags
	resp.RegisterCluster = instanceRes.Mesh
	if instance.RegisterTime == "" {
		resp.RegisterState = "Registered"
	} else {
		resp.RegisterState = "UnRegistered"
	}
	resp.RegisterTime = instance.RegisterTime
	resp.WorkloadName = instance.WorkloadName
	return resp
}

func GetAppServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	if req.Side == constants.ConsumerSide {
		return getAppProvideServiceInfo(ctx, req)
	} else {
		return getAppConsumeServiceInfo(ctx, req)
	}

}

func getAppProvideServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {

	pageData, err := manager.PageListByIndexes[*meshresource.ServiceProviderMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceProviderMetadataKind,
		map[string]string{
			index.ByMeshIndex:              req.Mesh,
			index.ByServiceProviderAppName: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*model.ApplicationServiceFormResp)
	for _, res := range pageData.Data {
		providerMetaData := res.Spec

		if resp, exists := serviceMap[providerMetaData.ServiceName]; exists {
			resp.VersionGroups = append(resp.VersionGroups, model.VersionGroup{
				Version: providerMetaData.Version,
				Group:   providerMetaData.Group,
			})
		} else {
			serviceMap[providerMetaData.ServiceName] = &model.ApplicationServiceFormResp{
				ServiceName: providerMetaData.ServiceName,
				VersionGroups: []model.VersionGroup{
					{
						Version: providerMetaData.Version,
						Group:   providerMetaData.Group,
					},
				},
			}
		}
	}

	pageResult := &model.SearchPaginationResult{
		List:     maputil.Values(serviceMap),
		PageInfo: pageData.Pagination,
	}
	return pageResult, nil
}

func getAppConsumeServiceInfo(ctx consolectx.Context, req *model.ApplicationServiceFormReq) (*model.SearchPaginationResult, error) {
	pageData, err := manager.PageListByIndexes[*meshresource.ServiceConsumerMetadataResource](
		ctx.ResourceManager(),
		meshresource.ServiceConsumerMetadataKind,
		map[string]string{
			index.ByMeshIndex:              req.Mesh,
			index.ByServiceConsumerAppName: req.AppName,
		},
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	serviceMap := make(map[string]*model.ApplicationServiceFormResp)
	for _, res := range pageData.Data {
		consumerMetadata := res.Spec

		if resp, exists := serviceMap[consumerMetadata.ServiceName]; exists {
			resp.VersionGroups = append(resp.VersionGroups, model.VersionGroup{
				Version: consumerMetadata.Version,
				Group:   consumerMetadata.Group,
			})
		} else {
			serviceMap[consumerMetadata.ServiceName] = &model.ApplicationServiceFormResp{
				ServiceName: consumerMetadata.ServiceName,
				VersionGroups: []model.VersionGroup{
					{
						Version: consumerMetadata.Version,
						Group:   consumerMetadata.Group,
					},
				},
			}
		}
	}

	pageResult := &model.SearchPaginationResult{
		List:     maputil.Values(serviceMap),
		PageInfo: pageData.Pagination,
	}
	return pageResult, nil
}

func SearchApplications(ctx consolectx.Context, req *model.ApplicationSearchReq) (*model.SearchPaginationResult, error) {
	pageData, err := searchApplications(ctx, req.AppName, req.Mesh, req.PageReq)
	if err != nil {
		return nil, err
	}
	respList := slice.Map[*meshresource.ApplicationResource, *model.ApplicationSearchResp](pageData.Data,
		func(_ int, item *meshresource.ApplicationResource) *model.ApplicationSearchResp {
			return buildApplicationSearchResp(item, req.Mesh)
		})
	searchResult := &model.SearchPaginationResult{
		List:     respList,
		PageInfo: pageData.Pagination,
	}
	return searchResult, nil
}

func BannerSearchApplications(ctx consolectx.Context, req *model.SearchReq) ([]*model.ApplicationSearchResp, error) {
	pageData, err := searchApplications(ctx, req.Keywords, req.Mesh, req.PageReq)
	if err != nil {
		return nil, err
	}
	respList := slice.Map[*meshresource.ApplicationResource, *model.ApplicationSearchResp](pageData.Data,
		func(_ int, item *meshresource.ApplicationResource) *model.ApplicationSearchResp {
			return buildApplicationSearchResp(item, req.Mesh)
		})
	return respList, nil
}

func searchApplications(
	ctx consolectx.Context,
	keywords string,
	mesh string,
	pageReq coremodel.PageReq) (*coremodel.PageData[*meshresource.ApplicationResource], error) {

	var pageData *coremodel.PageData[*meshresource.ApplicationResource]
	var err error
	if strutil.IsBlank(keywords) {
		pageData, err = manager.PageListByIndexes[*meshresource.ApplicationResource](
			ctx.ResourceManager(),
			meshresource.ApplicationKind,
			map[string]string{
				index.ByMeshIndex: mesh,
			},
			pageReq,
		)
	} else {
		pageData, err = manager.PageSearchResourceByConditions[*meshresource.ApplicationResource](
			ctx.ResourceManager(),
			meshresource.ApplicationKind,
			[]string{"name=" + keywords},
			pageReq,
		)
	}
	if err != nil {
		return nil, err
	}
	return pageData, nil
}

func buildApplicationSearchResp(appResource *meshresource.ApplicationResource, mesh string) *model.ApplicationSearchResp {
	application := appResource.Spec
	return &model.ApplicationSearchResp{
		AppName:          application.Name,
		DeployClusters:   []string{""},
		InstanceCount:    application.InstanceCount,
		RegistryClusters: []string{mesh},
	}
}

func isAppAccessLogConfig(conf *meshproto.OverrideConfig, appName string) bool {
	if conf.Side != constants.SideProvider ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Application == nil ||
		conf.Match.Application.Oneof == nil ||
		len(conf.Match.Application.Oneof) != 1 ||
		conf.Match.Application.Oneof[0].Exact != appName {
		return false
	}
	if _, ok := conf.Parameters[`accesslog`]; !ok {
		return false
	}
	return true
}

func UpInsertAppAccessLog(ctx consolectx.Context, appName string, openAccessLog bool, mesh string) error {
	// check app exists
	data, err := GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName})
	if err != nil {
		return err
	}
	if data == nil {
		return bizerror.New(bizerror.AppNotFound, fmt.Sprintf("%s does not exist", appName))
	}
	// check app configurator exists
	appConfiguratorName := appName + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, appConfiguratorName, mesh)
	if err != nil {
		return err
	}
	// if not exists, create one configurator with access log enable
	if res == nil {
		return insertConfiguratorWithAccessLog(ctx, res, openAccessLog, appConfiguratorName, appName, mesh)
	}
	// else we update the configurator
	return updateConfiguratorWithAccessLog(ctx, res, openAccessLog, appConfiguratorName, appName, mesh)
}

func insertConfiguratorWithAccessLog(ctx consolectx.Context, res *meshresource.DynamicConfigResource, openAccessLog bool,
	appConfiguratorName, appName, mesh string) error {
	// configurator is nil, accessLog is already closed
	if !openAccessLog {
		return nil
	}
	res = meshresource.NewDynamicConfigResourceWithAttributes(appConfiguratorName, mesh)
	res.Spec = &meshproto.DynamicConfig{
		Key:           appName,
		Scope:         constants.ScopeApplication,
		ConfigVersion: constants.ConfiguratorVersionV3,
		Enabled:       true,
		Configs:       make([]*meshproto.OverrideConfig, 0),
	}
	res.Spec.Configs = append(res.Spec.Configs, newAccessLogEnabledConfig(appName))
	err := CreateConfigurator(ctx, appConfiguratorName, res)
	if err != nil {
		logger.Errorf("create configurator failed when open accesslog, resourceKey: %s, openAccessLog: %t, err: %s",
			coremodel.BuildResourceKey(mesh, appName), openAccessLog, err)
		return err
	}
	return nil
}

func updateConfiguratorWithAccessLog(ctx consolectx.Context, res *meshresource.DynamicConfigResource, openAccessLog bool,
	appConfiguratorName, appName, mesh string) error {
	var accessLogConfig *meshproto.OverrideConfig
	res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
		if isAppAccessLogConfig(conf, appName) {
			accessLogConfig = conf
			return true
		}
		return false
	})
	// access log config not found
	if accessLogConfig == nil {
		// access log needs to be closed and already closed
		if !openAccessLog {
			return nil
		}
		// insert a access log enabled config
		res.Spec.Configs = append(res.Spec.Configs, newAccessLogEnabledConfig(appName))
	} else {
		// access log config found and status is the same as needed
		if accessLogConfig.Enabled == openAccessLog {
			return nil
		}
		// update the access log enabled status as needed
		accessLogConfig.Enabled = openAccessLog
	}
	err := UpdateConfigurator(ctx, appConfiguratorName, res)
	if err != nil {
		logger.Errorf("update configurator failed when opening accesslog, resourceKey: %s, openAccessLog: %t, err: %s",
			coremodel.BuildResourceKey(mesh, appName), openAccessLog, err)
		return err
	}
	return nil
}

func newAccessLogEnabledConfig(appName string) *meshproto.OverrideConfig {
	return &meshproto.OverrideConfig{
		Side:       constants.SideProvider,
		Parameters: map[string]string{`accesslog`: `true`},
		Enabled:    true,
		Match: &meshproto.ConditionMatch{
			Application: &meshproto.ListStringMatch{
				Oneof: []*meshproto.StringMatch{
					{
						Exact: appName,
					},
				}}},
		XGenerateByCp: true,
	}
}

func GetAppAccessLog(ctx consolectx.Context, appName string, mesh string) (*model.AppAccessLogConfigResp, error) {
	appConfiguratorName := appName + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, appConfiguratorName, mesh)
	resp := &model.AppAccessLogConfigResp{
		AccessLog: false,
	}
	if err != nil {
		logger.Errorf("get configurator failed when get app accesslog, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return nil, err
	}
	if res == nil {
		return resp, nil
	}
	var appAccessLogConfig *meshproto.OverrideConfig
	res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
		if isAppAccessLogConfig(conf, appName) {
			appAccessLogConfig = conf
			return true
		}
		return false
	})
	if appAccessLogConfig == nil {
		return resp, nil
	}
	resp.AccessLog = appAccessLogConfig.Enabled
	return resp, nil
}

func GetAppFlowWeight(ctx consolectx.Context, appName string, mesh string) (*model.AppFlowWeightConfigResp, error) {
	resp := &model.AppFlowWeightConfigResp{
		FlowWeightSets: []model.FlowWeightSet{},
	}
	appConfiguratorName := appName + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, appConfiguratorName, mesh)
	if err != nil {
		logger.Errorf("get configurator failed when get app flow weight, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return nil, err
	}
	if res == nil {
		return resp, nil
	}

	weight := 0
	res.Spec.RangeConfig(func(conf *meshproto.OverrideConfig) (isStop bool) {
		if isFlowWeightConfig(conf) {
			weight, err = strconv.Atoi(conf.Parameters[`weight`])
			if err != nil {
				logger.Error("parse weight failed", err)
				return true
			}
			scope := make([]model.ParamMatch, 0, len(conf.Match.Param))
			for _, param := range conf.Match.Param {
				scope = append(scope, model.ParamMatch{
					Key:   &param.Key,
					Value: model.StringMatchToModelStringMatch(param.Value),
				})
			}

			resp.FlowWeightSets = append(resp.FlowWeightSets, model.FlowWeightSet{
				Weight: int32(weight),
				Scope:  scope,
			})
		}
		return false
	})
	return resp, nil
}

func isFlowWeightConfig(conf *meshproto.OverrideConfig) bool {
	if conf.Side != constants.SideProvider ||
		conf.Parameters == nil ||
		conf.Match == nil ||
		conf.Match.Param == nil {
		return false
	}
	if _, ok := conf.Parameters[`weight`]; !ok {
		return false
	}
	return true
}

func UpInsertAppFlowWeightConfig(ctx consolectx.Context, appName string, mesh string, flowWeightSets []model.FlowWeightSet) error {
	// check app exists
	data, err := GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName})
	if err != nil {
		return err
	}
	if data == nil {
		return bizerror.New(bizerror.AppNotFound, fmt.Sprintf("%s does not exist", appName))
	}
	appConfiguratorName := appName + constants.ConfiguratorRuleDotSuffix
	res, err := GetConfigurator(ctx, appConfiguratorName, mesh)
	if err != nil {
		logger.Errorf("get configurator failed when update app flow weight, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	// configurator not exists, insert a new one
	if res == nil {
		return insertConfiguratorWithFlowWeight(ctx, flowWeightSets, appName, appConfiguratorName, mesh)
	}
	// configurator exists, update it

	// remove old flow weight config
	res.Spec.RangeConfigsToRemove(func(conf *meshproto.OverrideConfig) (IsRemove bool) {
		return isFlowWeightConfig(conf)
	})

	// add new flow weight config
	flowWeightConfigs := slice.Map(flowWeightSets, func(index int, set model.FlowWeightSet) *meshproto.OverrideConfig {
		return fromFlowWeightSet(set)
	})
	res.Spec.Configs = slice.Union(res.Spec.Configs, flowWeightConfigs)

	err = UpdateConfigurator(ctx, appConfiguratorName, res)
	if err != nil {
		logger.Errorf("update configurator failed with app flow weight, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	return nil
}

func insertConfiguratorWithFlowWeight(
	ctx consolectx.Context,
	flowWeightSets []model.FlowWeightSet,
	appName, appConfiguratorName, mesh string) error {
	res := meshresource.NewDynamicConfigResourceWithAttributes(appConfiguratorName, mesh)
	res.Spec = &meshproto.DynamicConfig{
		Key:           appName,
		Scope:         constants.ScopeApplication,
		ConfigVersion: constants.ConfiguratorVersionV3,
		Enabled:       true,
	}
	flowWeightConfigs := slice.Map(flowWeightSets, func(index int, set model.FlowWeightSet) *meshproto.OverrideConfig {
		return fromFlowWeightSet(set)
	})
	res.Spec.Configs = flowWeightConfigs
	err := CreateConfigurator(ctx, appConfiguratorName, res)
	if err != nil {
		logger.Errorf("insert configurator failed with app flow weight, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	return nil
}

func fromFlowWeightSet(set model.FlowWeightSet) *meshproto.OverrideConfig {
	paramMatch := make([]*meshproto.ParamMatch, 0, len(set.Scope))
	for _, match := range set.Scope {
		paramMatch = append(paramMatch, &meshproto.ParamMatch{
			Key:   *match.Key,
			Value: model.ModelStringMatchToStringMatch(match.Value),
		})
	}
	return &meshproto.OverrideConfig{
		Side:       constants.SideProvider,
		Parameters: map[string]string{`weight`: strconv.Itoa(int(set.Weight))},
		Match: &meshproto.ConditionMatch{
			Param: paramMatch,
		},
		XGenerateByCp: true,
	}
}
func GetGrayConfig(ctx consolectx.Context, appName string, mesh string) (*model.AppGrayConfigResp, error) {
	resp := &model.AppGrayConfigResp{}
	serviceTagRuleName := appName + constants.TagRuleDotSuffix
	res, err := GetTagRule(ctx, serviceTagRuleName, mesh)
	if err != nil {
		logger.Errorf("get tag rule failed when get gray config, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return nil, err
	}
	if res == nil {
		return resp, err
	}
	resp.GraySets = make([]model.GraySet, 0, len(res.Spec.Tags))

	res.Spec.RangeTags(func(tag *meshproto.Tag) (isStop bool) {
		if isGrayTag(tag) {
			scope := make([]model.ParamMatch, 0, len(tag.Match))
			for _, paramMatch := range tag.Match {
				scope = append(scope, model.ParamMatch{
					Key:   &paramMatch.Key,
					Value: model.StringMatchToModelStringMatch(paramMatch.Value),
				})
			}
			resp.GraySets = append(resp.GraySets, model.GraySet{
				EnvName: tag.Name,
				Scope:   scope,
			})
		}
		return false
	})
	return resp, nil
}

func isGrayTag(tag *meshproto.Tag) bool {
	if tag.Name == "" || len(tag.Addresses) != 0 {
		return false
	}
	return true
}

func UpInsertAppGrayConfig(ctx consolectx.Context, appName string, mesh string, graySets []model.GraySet) error {
	serviceTagRuleName := appName + constants.TagRuleDotSuffix
	res, err := GetTagRule(ctx, serviceTagRuleName, mesh)
	if err != nil {
		logger.Errorf("get tag rule failed when update app gray config, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	_, err = GetApplicationDetail(ctx, &model.ApplicationDetailReq{AppName: appName, Mesh: mesh})
	if err != nil {
		return err
	}
	// tag rule not exists, insert a new one
	if res == nil {
		return insertTagRuleWithGrayConfig(ctx, graySets, appName, serviceTagRuleName, mesh)
	}
	// tag rule exists, update it
	return updateTagRuleWithGrayConfig(ctx, graySets, res, appName, mesh)
}

func insertTagRuleWithGrayConfig(
	ctx consolectx.Context,
	graySets []model.GraySet,
	appName, serviceTagRuleName, mesh string) error {
	res := meshresource.NewTagRouteResourceWithAttributes(serviceTagRuleName, mesh)
	res.Spec = &meshproto.TagRoute{
		Enabled:       true,
		Key:           appName,
		ConfigVersion: constants.ConfiguratorVersionV3,
		Force:         false,
	}
	tags := slice.Map(graySets, func(index int, set model.GraySet) *meshproto.Tag {
		return fromGraySet(set)
	})
	res.Spec.Tags = tags
	err := CreateTagRule(ctx, res)
	if err != nil {
		logger.Errorf("insert tag rule failed with app gray config, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	return nil
}

func updateTagRuleWithGrayConfig(
	ctx consolectx.Context,
	graySets []model.GraySet,
	res *meshresource.TagRouteResource,
	appName string, mesh string) error {
	// remove old config, generate config from admin, append
	res.Spec.RangeTagsToRemove(func(tag *meshproto.Tag) (IsRemove bool) {
		return isGrayTag(tag)
	})
	tags := slice.Map(graySets, func(index int, set model.GraySet) *meshproto.Tag {
		return fromGraySet(set)
	})
	res.Spec.Tags = append(res.Spec.Tags, tags...)
	err := UpdateTagRule(ctx, res)
	if err != nil {
		logger.Errorf("update tag rule failed with app gray config, resourceKey: %s, err: %s",
			coremodel.BuildResourceKey(mesh, appName), err)
		return err
	}
	return nil
}

func fromGraySet(set model.GraySet) *meshproto.Tag {
	paramMatches := make([]*meshproto.ParamMatch, 0, len(set.Scope))
	for _, match := range set.Scope {
		paramMatches = append(paramMatches, &meshproto.ParamMatch{
			Key:   *match.Key,
			Value: model.ModelStringMatchToStringMatch(match.Value),
		})
	}

	return &meshproto.Tag{
		Name:          set.EnvName,
		Match:         paramMatches,
		XGenerateByCp: true,
	}
}
