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

package handler

import (
	"net/http"

	gxset "github.com/dubbogo/gost/container/set"
	"github.com/gin-gonic/gin"

	"github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	coremanager "github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func GetInstances(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := ctx.ResourceManager()
		dataplaneList := &mesh.DataplaneResourceList{}
		if err := manager.List(ctx.AppContext(), dataplaneList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(dataplaneList))
	}
}

func GetMetas(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := ctx.ResourceManager()
		metadataList := &mesh.MetaDataResourceList{}
		if err := manager.List(ctx.AppContext(), metadataList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(metadataList))
	}
}

func GetMappings(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		manager := ctx.ResourceManager()
		mappingList := &mesh.MappingResourceList{}
		if err := manager.List(ctx.AppContext(), mappingList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}
		c.JSON(http.StatusOK, model.NewSuccessResp(mappingList))
	}
}

func AdminMetadata(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		res := model.NewAdminMetadata()
		// TODO
		c.JSON(http.StatusOK, model.NewSuccessResp(res))
	}
}

func ClusterOverview(ctx consolectx.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		res := model.NewOverviewResp()

		manager := ctx.ResourceManager()
		dataplaneList := &mesh.DataplaneResourceList{}
		if err := manager.List(ctx.AppContext(), dataplaneList); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": err.Error(),
			})
			return
		}

		calAppCount(res, dataplaneList)
		err := calServiceInfo(res, dataplaneList, ctx, manager)
		if err != nil {
			return
		}
		res.InsCount = len(dataplaneList.Items)

		c.JSON(http.StatusOK, model.NewSuccessResp(res))
	}
}

func calAppCount(res *model.OverviewResp, list *mesh.DataplaneResourceList) {
	set := gxset.NewSet()
	for _, ins := range list.Items {
		set.Add(ins.Spec.GetExtensions()[v1alpha1.Application])
	}

	res.AppCount = set.Size()
}

func calServiceInfo(res *model.OverviewResp, list *mesh.DataplaneResourceList, ctx consolectx.Context, manager coremanager.ResourceManager) error {
	revisions := make(map[string]*mesh.MetaDataResource, 0)
	protocols := make(map[string]*gxset.HashSet)
	releases := make(map[string]string)
	services := make(map[string]string)
	discoveries := make(map[string]*gxset.HashSet)

	for _, ins := range list.Items {
		app := ins.Spec.GetExtensions()[v1alpha1.Application]

		t := ins.Spec.GetExtensions()["registry-type"]
		if t == "" {
			t = "instance"
		}
		if set, ok := discoveries[t]; ok {
			set.Add(app)
		} else {
			newSet := gxset.NewSet()
			newSet.Add(app)
			discoveries[t] = newSet
		}

		rev, exists := ins.Spec.GetExtensions()[v1alpha1.RevisionLabel]
		if exists {
			metadata, cached := revisions[rev]
			if !cached {
				metadata = &mesh.MetaDataResource{
					Spec: &v1alpha1.MetaData{},
				}
				if err := manager.Get(ctx.AppContext(), metadata, store.GetByRevision(rev), store.GetByType(ins.Spec.GetExtensions()["registry-type"])); err != nil {
					return err
				}
				revisions[rev] = metadata

				for _, serviceInfo := range metadata.Spec.Services {
					// proKey := serviceInfo.Protocol + strconv.Itoa(int(serviceInfo.Port))
					proKey := serviceInfo.Protocol
					if extProtocols, ok := serviceInfo.GetParams()["ext.protocol"]; ok {
						proKey = proKey + extProtocols
					}
					if set, ok := protocols[proKey]; ok {
						set.Add(serviceInfo.Name)
					} else {
						newSet := gxset.NewSet()
						protocols[proKey] = newSet
						newSet.Add(serviceInfo.Name)
					}

					if _, ok := releases[app]; !ok {
						releases[app] = serviceInfo.Params["release"]
					}

					if _, ok := services[serviceInfo.Name]; !ok {
						services[serviceInfo.Name] = serviceInfo.Name
					}
				}
			}
		}
	}

	releaseCount := make(map[string]int)
	for _, ver := range releases {
		if n, ok := releaseCount[ver]; ok {
			releaseCount[ver] = n + 1
		} else {
			releaseCount[ver] = 1
		}
	}

	protocolCount := make(map[string]int)
	for p, set := range protocols {
		protocolCount[p] = set.Size()
	}

	discoveryCount := make(map[string]int)
	for d, set := range discoveries {
		discoveryCount[d] = set.Size()
	}

	res.ServiceCount = len(services)
	res.Releases = releaseCount
	res.Protocols = protocolCount
	res.Discoveries = discoveryCount

	return nil
}
