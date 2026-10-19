//go:build e2e

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

package routerrulechain_test

import (
	"os"
	"strings"
	"testing"

	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/remoting"

	"github.com/apache/dubbo-admin/pkg/core/clients"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const e2eZKAddressEnv = "DUBBO_ADMIN_E2E_ZK_ADDR"

// TestZookeeperRouterRuleChainLifecycle starts at the public Console API, then
// verifies ZooKeeper-specific propagation before the shared router assertions.
func TestZookeeperRouterRuleChainLifecycle(t *testing.T) {
	address := configCenterAddress(configCenterZookeeper, os.Getenv(e2eZKAddressEnv))
	if address == "" {
		t.Skipf("set %s to run the ZooKeeper integration test", e2eZKAddressEnv)
	}
	e2e := newZookeeperRouterRuleChainE2E(t, requireAdminRuntimeFixture(t), address)
	t.Run("Affinity", e2e.testAffinityLifecycle)
	t.Run("Script", e2e.testScriptLifecycle)
}

func newZookeeperRouterRuleChainE2E(t *testing.T, fixture *adminRuntimeFixture, address string) *routerRuleChainE2E {
	t.Helper()
	adapter := &zookeeperConfigCenterE2EAdapter{}
	e2e := newRouterRuleChainE2E(t, fixture, configCenterZookeeper, e2eZKMesh, address, adapter)
	conn, err := clients.NewZKConnection(address)
	require.NoError(t, err)
	adapter.conn = conn
	t.Cleanup(conn.Close)
	adapter.events = e2e.subscribe(t, meshresource.ZKConfigKind, "zk-config")
	return e2e
}

type zookeeperConfigCenterE2EAdapter struct {
	conn   *zk.Conn
	events *resourceEventRecorder
}

func (*zookeeperConfigCenterE2EAdapter) configureURL(*common.URL) {}

func (a *zookeeperConfigCenterE2EAdapter) configMatches(_ *routerRuleChainE2E, name, content string, exists bool) bool {
	data, _, getErr := a.conn.Get("/dubbo/config/dubbo/" + name)
	legacy, _, legacyErr := a.conn.Exists("/dubbo/config/" + name)
	if exists {
		return getErr == nil && legacyErr == nil && !legacy && strings.Contains(string(data), content)
	}
	return getErr == zk.ErrNoNode && legacyErr == nil && !legacy
}

func (a *zookeeperConfigCenterE2EAdapter) verifyUpsert(t *testing.T, e2e *routerRuleChainE2E, rule coremodel.Resource, events *resourceEventRecorder, rawType cache.DeltaType, content string) {
	t.Helper()
	a.events.waitFor(t, rule.ResourceKey(), rawType, "")
	e2e.waitStore(t, meshresource.ZKConfigKind, rule.ResourceKey(), true, func(resource coremodel.Resource) bool {
		config, ok := resource.(*meshresource.ZKConfigResource)
		return ok && config.Spec != nil && config.Spec.NodeName == rule.ResourceMeta().Name && strings.Contains(config.Spec.NodeData, content)
	})
	// The source context belongs to the typed rule event emitted by
	// ZKConfigEventSubscriber, not to the raw informer event.
	events.waitFor(t, rule.ResourceKey(), cache.Updated, "zookeeper")
}

func (a *zookeeperConfigCenterE2EAdapter) verifyDelete(t *testing.T, e2e *routerRuleChainE2E, rule coremodel.Resource, _ *resourceEventRecorder) {
	t.Helper()
	a.events.waitFor(t, rule.ResourceKey(), cache.Deleted, "")
	e2e.waitStore(t, meshresource.ZKConfigKind, rule.ResourceKey(), false, nil)
}

func (*zookeeperConfigCenterE2EAdapter) listenerEventType(expected remoting.EventType) remoting.EventType {
	if expected == remoting.EventTypeUpdate {
		// The pinned Dubbo-Go ZooKeeper cache listener reports znode data
		// changes as Add. The RouterChain assertion verifies replacement.
		return remoting.EventTypeAdd
	}
	return expected
}

// Fresh ZooKeeper instances lack the roots used by the other real ListWatchers.
func prepareZKRoots(address string) (func(), error) {
	conn, err := clients.NewZKConnection(address)
	if err != nil {
		return nil, err
	}
	created := make([]string, 0, 5)
	for _, target := range []string{"/dubbo/config", "/dubbo/mapping", "/dubbo/metadata", "/services"} {
		current := ""
		for _, part := range strings.Split(strings.Trim(target, "/"), "/") {
			current += "/" + part
			exists, _, existsErr := conn.Exists(current)
			if existsErr != nil {
				conn.Close()
				return nil, existsErr
			}
			if exists {
				continue
			}
			_, createErr := conn.Create(current, nil, 0, zk.WorldACL(zk.PermAll))
			if createErr == nil {
				created = append(created, current)
				continue
			}
			if createErr != zk.ErrNodeExists {
				conn.Close()
				return nil, createErr
			}
		}
	}
	return func() {
		for i := len(created) - 1; i >= 0; i-- {
			_ = conn.Delete(created[i], -1)
		}
		conn.Close()
	}, nil
}
