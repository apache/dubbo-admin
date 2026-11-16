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

package zk

import (
	"testing"

	"github.com/stretchr/testify/require"

	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestZKConfigResourcesAcceptOnlyGroupedRulePaths(t *testing.T) {
	const (
		mesh = "test-mesh"
		name = "demo.affinity-router"
	)

	upsert := toUpsertZKConfigResource(mesh, "/dubbo/config/dubbo/"+name, "rule-content")
	require.NotNil(t, upsert)
	upsertConfig, ok := upsert.(*meshresource.ZKConfigResource)
	require.True(t, ok)
	require.Equal(t, name, upsertConfig.Name)
	require.Equal(t, "rule-content", upsertConfig.Spec.NodeData)

	deleted := toDeleteZKConfigResource(mesh, "/dubbo/config/dubbo/"+name)
	require.NotNil(t, deleted)
	deleteConfig, ok := deleted.(*meshresource.ZKConfigResource)
	require.True(t, ok)
	require.Equal(t, name, deleteConfig.Name)

	for _, path := range []string{
		"/dubbo/config/" + name,
		"/dubbo/config/dubbo",
		"/dubbo/config/other/" + name,
		"/dubbo/config/dubbo/",
	} {
		require.Nil(t, toUpsertZKConfigResource(mesh, path, "rule-content"), path)
		require.Nil(t, toDeleteZKConfigResource(mesh, path), path)
	}
}
