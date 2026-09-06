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

package tools

import (
	"context"
	"encoding/json"
	"testing"

	meshapi "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/config/app"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/lock"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
	"github.com/apache/dubbo-admin/pkg/core/versioning"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetServiceDetailMissingServiceName(t *testing.T) {
	result, err := GetServiceDetail(newToolTestContext(nil), map[string]any{})
	if err != nil {
		t.Fatalf("GetServiceDetail returned unexpected error: %v", err)
	}
	if !result.IsError {
		t.Fatal("Expected error result")
	}
	if got := result.Content[0].Text; got != "required parameter 'serviceName' is missing" {
		t.Fatalf("Expected missing parameter error, got %q", got)
	}
}

func TestGetServiceDetailSuccess(t *testing.T) {
	const (
		mesh        = "mesh1"
		serviceName = "org.apache.demo.DemoService"
		version     = "1.0.0"
		group       = "demo"
	)
	serviceKey := coremodel.BuildResourceKey(mesh, meshresource.BuildServiceIdentityKey(serviceName, version, group))
	resource := &meshresource.ServiceResource{
		ObjectMeta: metav1.ObjectMeta{Name: meshresource.BuildServiceIdentityKey(serviceName, version, group)},
		Mesh:       mesh,
		Spec: &meshapi.Service{
			Name:     serviceName,
			Version:  version,
			Group:    group,
			Language: "java",
			Methods:  []string{"sayHello"},
		},
	}
	ctx := newToolTestContext(map[coremodel.ResourceKind]map[string]coremodel.Resource{
		meshresource.ServiceKind: {
			serviceKey: resource,
		},
	})

	result, err := GetServiceDetail(ctx, map[string]any{
		"serviceName": serviceName,
		"version":     version,
		"group":       group,
		"mesh":        mesh,
	})
	if err != nil {
		t.Fatalf("GetServiceDetail returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Expected success result, got %q", result.Content[0].Text)
	}

	var payload struct {
		Language string   `json:"language"`
		Methods  []string `json:"methods"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if payload.Language != "java" {
		t.Fatalf("Expected language java, got %q", payload.Language)
	}
	if len(payload.Methods) != 1 || payload.Methods[0] != "sayHello" {
		t.Fatalf("Expected methods [sayHello], got %v", payload.Methods)
	}
}

func TestGetServiceDistributionSuccessWithEmptyDistribution(t *testing.T) {
	result, err := GetServiceDistribution(newToolTestContext(nil), map[string]any{"serviceName": "missing"})
	if err != nil {
		t.Fatalf("get_service_distribution handler returned unexpected error: %v", err)
	}
	if result.IsError {
		t.Fatalf("Expected success result with empty distribution, got %q", result.Content[0].Text)
	}

	var payload struct {
		ServiceName  string `json:"serviceName"`
		Distribution []any  `json:"distribution"`
		TotalApps    int    `json:"totalApps"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &payload); err != nil {
		t.Fatalf("Failed to unmarshal result: %v", err)
	}
	if payload.ServiceName != "missing" {
		t.Fatalf("Expected serviceName missing, got %q", payload.ServiceName)
	}
	if len(payload.Distribution) != 0 || payload.TotalApps != 0 {
		t.Fatalf("Expected empty distribution, got distribution=%v totalApps=%d", payload.Distribution, payload.TotalApps)
	}
}

func newToolTestContext(resources map[coremodel.ResourceKind]map[string]coremodel.Resource) consolectx.Context {
	return &toolTestContext{
		config: app.AdminConfig{
			Discovery: []*discoverycfg.Config{{ID: "mesh1"}},
			Engine:    &enginecfg.Config{Name: "engine1"},
		},
		resourceManager: &toolTestResourceManager{resources: resources},
	}
}

type toolTestContext struct {
	config          app.AdminConfig
	resourceManager manager.ResourceManager
}

func (c *toolTestContext) ResourceManager() manager.ResourceManager {
	return c.resourceManager
}

func (c *toolTestContext) CounterManager() counter.CounterManager {
	return nil
}

func (c *toolTestContext) Config() app.AdminConfig {
	return c.config
}

func (c *toolTestContext) AppContext() context.Context {
	return context.Background()
}

func (c *toolTestContext) LockManager() lock.Lock {
	return nil
}

func (c *toolTestContext) RuleVersioning() *versioning.Service {
	return nil
}

type toolTestResourceManager struct {
	resources map[coremodel.ResourceKind]map[string]coremodel.Resource
}

func (m *toolTestResourceManager) GetByKey(rk coremodel.ResourceKind, key string) (coremodel.Resource, bool, error) {
	byKind := m.resources[rk]
	if byKind == nil {
		return nil, false, nil
	}
	resource, ok := byKind[key]
	return resource, ok, nil
}

func (m *toolTestResourceManager) GetByKeys(rk coremodel.ResourceKind, keys []string) ([]coremodel.Resource, error) {
	byKind := m.resources[rk]
	result := make([]coremodel.Resource, 0, len(keys))
	for _, key := range keys {
		if resource, ok := byKind[key]; ok {
			result = append(result, resource)
		}
	}
	return result, nil
}

func (m *toolTestResourceManager) ListByIndexes(coremodel.ResourceKind, []index.IndexCondition) ([]coremodel.Resource, error) {
	return nil, nil
}

func (m *toolTestResourceManager) PageListByIndexes(coremodel.ResourceKind, []index.IndexCondition, coremodel.PageReq) (*coremodel.PageData[coremodel.Resource], error) {
	return coremodel.NewPageData[coremodel.Resource](0, 0, 0, nil), nil
}

func (m *toolTestResourceManager) Add(coremodel.Resource) error {
	return nil
}

func (m *toolTestResourceManager) Update(coremodel.Resource) error {
	return nil
}

func (m *toolTestResourceManager) Upsert(coremodel.Resource) error {
	return nil
}

func (m *toolTestResourceManager) DeleteByKey(coremodel.ResourceKind, string, string) error {
	return nil
}
