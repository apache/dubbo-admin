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
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/consts"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coreresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func GetConfigurator(ctx consolectx.Context, name string) (*meshproto.DynamicConfig, error) {
	res := &coreresource.DynamicConfig{Spec: &meshproto.DynamicConfig{}}
	if err := ctx.ResourceManager().Get(ctx.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.GetByApplication(name), store.GetByKey(name+consts.ConfiguratorRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("get %s configurator failed with error: %s", name, err.Error())
		return nil, err
	}
	return res, nil
}

func UpdateConfigurator(ctx consolectx.Context, name string, res *mesh.DynamicConfigResource) error {
	if err := ctx.ResourceManager().Update(ctx.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.UpdateByApplication(name), store.UpdateByKey(name+consts.ConfiguratorRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("update %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func CreateConfigurator(ctx consolectx.Context, name string, res *mesh.DynamicConfigResource) error {
	if err := ctx.ResourceManager().Create(ctx.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.CreateByApplication(name), store.CreateByKey(name+consts.ConfiguratorRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("create %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}

func DeleteConfigurator(ctx consolectx.Context, name string, res *mesh.DynamicConfigResource) error {
	if err := ctx.ResourceManager().Delete(ctx.AppContext(), res,
		// here `name` may be service-name or app-name, set *ByApplication(`name`) is ok.
		store.DeleteByApplication(name), store.DeleteByKey(name+consts.ConfiguratorRuleSuffix, coremodel.DefaultMesh)); err != nil {
		logger.Warnf("delete %s configurator failed with error: %s", name, err.Error())
		return err
	}
	return nil
}
