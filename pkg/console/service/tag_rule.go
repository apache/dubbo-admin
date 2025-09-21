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

func GetTagRule(ctx consolectx.Context, name string, mesh string) (*meshresource.TagRouteResource, error) {
	res, _, err := manager.GetByKey[*meshresource.TagRouteResource](
		ctx.ResourceManager(),
		meshresource.TagRouteKind,
		coremodel.BuildResourceKey(mesh, name),
	)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func UpdateTagRule(ctx consolectx.Context, res *meshresource.TagRouteResource) error {
	err := ctx.ResourceManager().Update(res)
	if err != nil {
		logger.Warnf("update tag rule %s error: %v", res.Name, err)
		return err
	}
	return nil
}

func CreateTagRule(ctx consolectx.Context, res *meshresource.TagRouteResource) error {
	err := ctx.ResourceManager().Add(res)
	if err != nil {
		logger.Warnf("create tag rule %s error: %v", res.Name, err)
		return err
	}
	return nil
}

func DeleteTagRule(ctx consolectx.Context, name string, mesh string) error {
	err := ctx.ResourceManager().DeleteByKey(meshresource.TagRouteKind, coremodel.BuildResourceKey(mesh, name))
	if err != nil {
		logger.Warnf("delete tag rule %s error: %v", name, err)
		return err
	}
	return nil
}
