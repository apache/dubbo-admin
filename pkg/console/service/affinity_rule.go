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
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

func GetAffinityRule(ctx consolectx.Context, name string, mesh string) (*meshresource.AffinityRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.AffinityRouteResource](
		ctx.ResourceManager(),
		meshresource.AffinityRouteKind,
		coremodel.BuildResourceKey(mesh, name))
	if err != nil {
		logger.Warnf("get affinity rule %s error: %v", name, err)
		return nil, err
	}
	return res, nil
}

func UpdateAffinityRule(ctx consolectx.Context, res *meshresource.AffinityRouteResource) error {
	if err := ctx.ResourceManager().Update(res); err != nil {
		logger.Warnf("update %s affinity rule failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func CreateAffinityRule(ctx consolectx.Context, res *meshresource.AffinityRouteResource) error {
	if err := ctx.ResourceManager().Add(res); err != nil {
		logger.Warnf("create %s condition failed with error: %s", res.Name, err.Error())
		return err
	}
	return nil
}

func DeleteAffinityRule(ctx consolectx.Context, name string, mesh string) error {
	if err := ctx.ResourceManager().DeleteByKey(
		meshresource.AffinityRouteKind,
		coremodel.BuildResourceKey(mesh, name)); err != nil {
		return err
	}
	return nil
}
