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

package manager

import (
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/apache/dubbo-admin/pkg/core/governor"
	"github.com/apache/dubbo-admin/pkg/core/resource/model"
	corestore "github.com/apache/dubbo-admin/pkg/core/store"
	memorystore "github.com/apache/dubbo-admin/pkg/store/memory"
)

func TestResourceManagerListUsesStoreFullList(t *testing.T) {
	const kind model.ResourceKind = "TestManagerResource"
	st := memorystore.NewMemoryResourceStore(kind)
	require.NoError(t, st.Init(nil))
	res1 := &managerTestResource{kind: kind, key: "mesh/rule-b", mesh: "mesh", meta: metav1.ObjectMeta{Name: "rule-b"}}
	res2 := &managerTestResource{kind: kind, key: "mesh/rule-a", mesh: "mesh", meta: metav1.ObjectMeta{Name: "rule-a"}}
	require.NoError(t, st.Add(res1))
	require.NoError(t, st.Add(res2))

	rm := NewResourceManager(singleStoreRouter{store: st}, noopGovernorRouter{})
	resources, err := rm.List(kind)
	require.NoError(t, err)
	require.Len(t, resources, 2)
	require.Equal(t, "mesh/rule-a", resources[0].ResourceKey())
	require.Equal(t, "mesh/rule-b", resources[1].ResourceKey())
}

type managerTestResource struct {
	kind model.ResourceKind
	key  string
	mesh string
	meta metav1.ObjectMeta
}

func (r *managerTestResource) ResourceMesh() string {
	return r.mesh
}

func (r *managerTestResource) GetObjectKind() schema.ObjectKind {
	return schema.EmptyObjectKind
}

func (r *managerTestResource) DeepCopyObject() runtime.Object {
	return r
}

func (r *managerTestResource) ResourceKind() model.ResourceKind {
	return r.kind
}

func (r *managerTestResource) ResourceKey() string {
	return r.key
}

func (r *managerTestResource) ResourceMeta() metav1.ObjectMeta {
	return r.meta
}

func (r *managerTestResource) ResourceSpec() model.ResourceSpec {
	return nil
}

func (r *managerTestResource) String() string {
	return r.key
}

type singleStoreRouter struct {
	store corestore.ResourceStore
}

func (r singleStoreRouter) ResourceRoute(model.Resource) (corestore.ResourceStore, error) {
	return r.store, nil
}

func (r singleStoreRouter) ResourceKindRoute(model.ResourceKind) (corestore.ResourceStore, error) {
	return r.store, nil
}

type noopGovernorRouter struct{}

func (noopGovernorRouter) ResourceRoute(model.Resource) (governor.RuleGovernor, error) {
	return noopGovernor{}, nil
}

func (noopGovernorRouter) ResourceMeshRoute(string) (governor.RuleGovernor, error) {
	return noopGovernor{}, nil
}

type noopGovernor struct{}

func (noopGovernor) CreateRule(model.Resource) error {
	return nil
}

func (noopGovernor) UpdateRule(model.Resource) error {
	return nil
}

func (noopGovernor) DeleteRule(model.Resource) error {
	return nil
}
