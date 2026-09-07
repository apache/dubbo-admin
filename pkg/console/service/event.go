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
	"strings"

	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

const defaultPageSize = 20

// ListApplicationEvents returns a paginated event timeline for a Dubbo application.
// Resolves pod names from all instances belonging to this app, then matches
// events by InvolvedObjName against both the pod names and the registry event
// encoding (appName/...).
func ListApplicationEvents(ctx consolectx.Context, req *model.EventQueryReq) (*model.EventListResp, error) {
	matchNames := resolveAppEventNames(ctx, req)

	allConditions := []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
	}
	resources, err := manager.ListByIndexes[*meshresource.LifecycleEventResource](
		ctx.ResourceManager(),
		meshresource.LifecycleEventKind,
		allConditions,
	)
	if err != nil {
		return nil, err
	}

	filtered := filterEventsByNames(resources, matchNames)

	offset := req.PageOffset
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	paged := filtered[offset:end]

	return toEventListResp(&coremodel.PageData[*meshresource.LifecycleEventResource]{
		Pagination: coremodel.Pagination{
			Total:      len(filtered),
			PageOffset: req.PageOffset,
			PageSize:   req.PageSize,
		},
		Data: paged,
	}), nil
}

// resolveAppEventNames returns all InvolvedObjName values that could match
// events for the given application. It collects pod names from RuntimeInstances
// belonging to this app, plus the registry event prefix and the K8s pod name prefix.
func resolveAppEventNames(ctx consolectx.Context, req *model.EventQueryReq) []string {
	names := make([]string, 0)

	if req.AppName == "" {
		return names
	}

	// Registry event format: appName/... (matched by HasPrefix in filter)
	names = append(names, req.AppName+"/")

	// K8s Deployment pod names follow {appName}-{hash}-{hash}.
	// This HasPrefix fallback catches events even after RuntimeInstances are gone.
	names = append(names, req.AppName+"-")

	// Find all InstanceResources for this app, then map to pod names via RuntimeInstance IP.
	instances, err := manager.ListByIndexes[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		[]index.IndexCondition{
			{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
			{IndexName: index.ByInstanceAppNameIndex, Value: req.AppName, Operator: index.Equals},
		},
	)
	if err != nil {
		logger.Warnf("resolveAppEventNames: failed to list instances for app %s: %v", req.AppName, err)
	}
	for _, inst := range instances {
		if inst.Spec == nil || inst.Spec.Ip == "" {
			continue
		}
		rtResources, err := manager.ListByIndexes[*meshresource.RuntimeInstanceResource](
			ctx.ResourceManager(),
			meshresource.RuntimeInstanceKind,
			[]index.IndexCondition{
				{IndexName: index.ByRuntimeInstanceIPIndex, Value: inst.Spec.Ip, Operator: index.Equals},
			},
		)
		if err != nil {
			continue
		}
		for _, rt := range rtResources {
			if rt.Spec != nil && rt.Spec.Name != "" {
				names = append(names, rt.Spec.Name)
			}
		}
	}

	return names
}

// filterEventsByNames returns events whose InvolvedObjName matches any of the
// given names, supporting both exact match and prefix match.
// Prefix candidates end with "/" (registry: appName/...) or "-" (K8s pod: appName-hash-hash).
func filterEventsByNames(resources []*meshresource.LifecycleEventResource, matchNames []string) []*meshresource.LifecycleEventResource {
	filtered := make([]*meshresource.LifecycleEventResource, 0)
	for _, r := range resources {
		if r.Spec == nil {
			continue
		}
		n := r.Spec.InvolvedObjName
		for _, candidate := range matchNames {
			if strings.HasSuffix(candidate, "/") || strings.HasSuffix(candidate, "-") {
				if strings.HasPrefix(n, candidate) {
					filtered = append(filtered, r)
					break
				}
			} else {
				if n == candidate {
					filtered = append(filtered, r)
					break
				}
			}
		}
	}
	return filtered
}

// ListInstanceEvents returns a paginated event timeline for a specific instance.
// K8s events use InvolvedObjName = podName; registry events use appName/ip.
// The frontend sends instanceName = {appName}{IP}:{port}, which doesn't match
// either format. So we look up the RuntimeInstance by IP to get the pod name,
// then match events against both the pod name and the IP-based encoding.
func ListInstanceEvents(ctx consolectx.Context, req *model.EventQueryReq) (*model.EventListResp, error) {
	matchNames := resolveInstanceEventNames(ctx, req)

	allConditions := []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
	}
	resources, err := manager.ListByIndexes[*meshresource.LifecycleEventResource](
		ctx.ResourceManager(),
		meshresource.LifecycleEventKind,
		allConditions,
	)
	if err != nil {
		return nil, err
	}

	filtered := filterEventsByNames(resources, matchNames)

	offset := req.PageOffset
	if offset < 0 {
		offset = 0
	}
	if offset > len(filtered) {
		offset = len(filtered)
	}
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	paged := filtered[offset:end]

	return toEventListResp(&coremodel.PageData[*meshresource.LifecycleEventResource]{
		Pagination: coremodel.Pagination{
			Total:      len(filtered),
			PageOffset: req.PageOffset,
			PageSize:   req.PageSize,
		},
		Data: paged,
	}), nil
}

// resolveInstanceEventNames returns the set of InvolvedObjName values that
// could match this instance's events. For K8s events it's the pod name; for
// registry events it's appName/ip.
func resolveInstanceEventNames(ctx consolectx.Context, req *model.EventQueryReq) []string {
	names := make([]string, 0, 3)

	// Always include the raw instanceName and IP as candidates (covers registry
	// events and the case where the instance name IS the pod name).
	if req.InstanceName != "" {
		names = append(names, req.InstanceName)
	}
	if req.InstanceIP != "" {
		names = append(names, req.InstanceIP)
		// Registry event format: appName/ip
		if req.AppName != "" {
			names = append(names, req.AppName+"/"+req.InstanceIP)
		}
	}

	// Resolve pod name from RuntimeInstance by IP.
	if req.InstanceIP != "" {
		rtResources, err := manager.ListByIndexes[*meshresource.RuntimeInstanceResource](
			ctx.ResourceManager(),
			meshresource.RuntimeInstanceKind,
			[]index.IndexCondition{
				{IndexName: index.ByRuntimeInstanceIPIndex, Value: req.InstanceIP, Operator: index.Equals},
			},
		)
		if err != nil {
			logger.Warnf("resolveInstanceEventNames: failed to list RuntimeInstance by IP %s: %v", req.InstanceIP, err)
		} else {
			for _, rt := range rtResources {
				if rt.Spec != nil && rt.Spec.Name != "" {
					names = append(names, rt.Spec.Name)
				}
			}
		}
	}

	return names
}

// ListServiceEvents returns a paginated event timeline for a Dubbo service.
// Covers registry-side config and metadata events. K8s has no native
// "service" concept, so K8s-sourced events naturally will not appear here.
func ListServiceEvents(ctx consolectx.Context, req *model.EventQueryReq) (*model.EventListResp, error) {
	conditions := []index.IndexCondition{
		{IndexName: index.ByMeshIndex, Value: req.Mesh, Operator: index.Equals},
	}

	if req.ServiceName != "" && req.AppName != "" {
		conditions = append(conditions, index.IndexCondition{
			IndexName: index.ByLifecycleEventInvolvedObjName,
			Value:     req.AppName + "/" + req.ServiceName,
			Operator:  index.HasPrefix,
		})
	}

	pageData, err := manager.PageListByIndexes[*meshresource.LifecycleEventResource](
		ctx.ResourceManager(),
		meshresource.LifecycleEventKind,
		conditions,
		req.PageReq,
	)
	if err != nil {
		return nil, err
	}

	return toEventListResp(pageData), nil
}

func toEventListResp(pageData *coremodel.PageData[*meshresource.LifecycleEventResource]) *model.EventListResp {
	items := pageData.Data
	list := make([]*model.EventItem, 0, len(items))
	for _, eventRes := range items {
		if eventRes.Spec == nil {
			continue
		}

		eventType := "normal"
		if strings.EqualFold(eventRes.Spec.Type, "Warning") {
			eventType = "warning"
		}

		source := eventRes.Spec.SourceComponent
		if source == "" {
			source = eventRes.Spec.EventSource
		}

		list = append(list, &model.EventItem{
			Time:    eventRes.Spec.LastTimestamp,
			Type:    eventType,
			Message: eventRes.Spec.Message,
			Source:  source,
		})
	}

	return &model.EventListResp{
		List:  list,
		Total: pageData.Total,
	}
}
