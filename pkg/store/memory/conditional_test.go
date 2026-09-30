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

package memory

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func TestConditionalMutations(t *testing.T) {
	managed := NewMemoryResourceStore(meshresource.MCPServerKind)
	require.NoError(t, managed.Init(nil))
	conditional := managed.(store.ConditionalResourceStore)

	resource := meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")
	require.NoError(t, managed.Add(resource))
	require.Equal(t, "1", resource.ResourceVersion)
	resource.Spec.Draft = &meshproto.MCPServerSnapshot{DisplayName: "Unstored change"}
	stored, exists, err := managed.GetByKey(resource.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists)
	require.Nil(t, stored.(*meshresource.MCPServerResource).Spec.Draft)
	stored.(*meshresource.MCPServerResource).Spec.Draft = &meshproto.MCPServerSnapshot{DisplayName: "Read mutation"}

	updated := resource.DeepCopyObject().(*meshresource.MCPServerResource)
	updated.Spec.Draft = &meshproto.MCPServerSnapshot{DisplayName: "Orders"}
	require.NoError(t, conditional.CompareAndSwap(updated, "1"))
	require.Equal(t, "2", updated.ResourceVersion)

	stale := resource.DeepCopyObject().(*meshresource.MCPServerResource)
	err = conditional.CompareAndSwap(stale, "1")
	require.True(t, errors.Is(err, store.ErrorResourceConflict("", "", "")))

	item, exists, err := managed.GetByKey(updated.ResourceKey())
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, "Orders", item.(*meshresource.MCPServerResource).Spec.Draft.DisplayName)
	require.Equal(t, "2", item.(*meshresource.MCPServerResource).ResourceVersion)
	require.ErrorIs(t, managed.Add(meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")), store.ErrorResourceAlreadyExists("", "", ""))

	require.NoError(t, conditional.CompareAndDelete(updated, "2"))
	_, exists, err = managed.GetByKey(updated.ResourceKey())
	require.NoError(t, err)
	require.False(t, exists)
}

func TestVersionedResourcesRejectUnconditionalMutation(t *testing.T) {
	managed := NewMemoryResourceStore(meshresource.MCPServerKind)
	require.NoError(t, managed.Init(nil))
	resource := meshresource.NewMCPServerResourceWithAttributes("server-1", "mesh-1")
	require.NoError(t, managed.Add(resource))

	require.Error(t, managed.Update(resource))
	require.Error(t, managed.Delete(resource))
	require.Error(t, managed.Replace(nil, ""))
}
