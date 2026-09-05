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
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/affinity"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	scriptrouter "dubbo.apache.org/dubbo-go/v3/cluster/router/script"
	"dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconfig "dubbo.apache.org/dubbo-go/v3/common/config"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	dubboconfigcenter "dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	adminzk "github.com/apache/dubbo-admin/pkg/governor/zk"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
)

const e2eZKAddressEnv = "DUBBO_ADMIN_E2E_ZK_ADDR"

// TestAdminRulesReachDubboGoRouterChain checks the production direction of
// travel: the Admin governor writes a ZK rule, then a real Dubbo-Go dynamic
// configuration listener changes the routers already installed in a chain.
//
// It is intentionally opt-in because it needs an external ZooKeeper server:
//
//	DUBBO_ADMIN_E2E_ZK_ADDR=zookeeper://127.0.0.1:2181 go test -tags=e2e ./e2e/router-rule-chain
func TestAdminRulesReachDubboGoRouterChain(t *testing.T) {
	zkAddress := os.Getenv(e2eZKAddressEnv)
	if zkAddress == "" {
		t.Skipf("set %s to run the ZooKeeper integration test", e2eZKAddressEnv)
	}
	// The Admin module currently pins a Dubbo-Go version in which the Script
	// factory is intentionally opt-in. Register it explicitly so this test
	// reflects a consumer that enables Script Router; current Dubbo-Go builds
	// register the same factory during normal imports.
	extension.SetRouterFactory(dubbogoconstant.ScriptRouterFactoryKey, scriptrouter.NewScriptRouterFactory)

	storeRouter := newE2EStoreRouter(t)
	governor, err := adminzk.NewZKRuleGovernor(&discoverycfg.Config{
		ID:   "router-rule-chain-e2e",
		Name: "router-rule-chain-e2e",
		Type: discoverycfg.Zookeeper,
		Address: discoverycfg.AddressConfig{
			Registry:     zkAddress,
			ConfigCenter: zkAddress,
		},
	}, storeRouter, discardEmitter{})
	require.NoError(t, err)

	configURL, err := common.NewURL(zkAddress)
	require.NoError(t, err)
	dynamicConfigFactory, err := extension.GetConfigCenterFactory("zookeeper")
	require.NoError(t, err)
	dynamicConfig, err := dynamicConfigFactory.GetDynamicConfiguration(configURL)
	require.NoError(t, err)
	oldDynamicConfig := dubbogoconfig.GetEnvInstance().GetDynamicConfiguration()
	dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(dynamicConfig)
	t.Cleanup(func() {
		dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(oldDynamicConfig)
	})

	providerApplication := fmt.Sprintf("router-rule-e2e-%d", time.Now().UnixNano())
	consumerURL := mustURL(t, fmt.Sprintf(
		"consumer://127.0.0.1:20000/com.example.RouterRuleE2E?application=consumer-%s&region=beijing",
		providerApplication))
	chain, err := chain.NewRouterChain(consumerURL)
	require.NoError(t, err)
	chain.SetInvokers([]base.Invoker{
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf(
			"dubbo://127.0.0.1:20880/com.example.RouterRuleE2E?application=%s&region=beijing", providerApplication))),
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf(
			"dubbo://127.0.0.1:20881/com.example.RouterRuleE2E?application=%s&region=shanghai", providerApplication))),
	})
	inv := invocation.NewRPCInvocation("sayHello", nil, nil)

	affinityName := providerApplication + ".affinity-router"
	affinityEvents := newConfigEventRecorder()
	dynamicConfig.AddListener(affinityName, affinityEvents)
	affinityRule := meshresource.NewAffinityRouteResourceWithAttributes(affinityName, "router-rule-chain-e2e")
	affinityRule.Spec.ConfigVersion = "v3.1"
	affinityRule.Spec.Scope = "application"
	affinityRule.Spec.Key = providerApplication
	affinityRule.Spec.Runtime = true
	affinityRule.Spec.Enabled = true
	affinityRule.Spec.Affinity = &meshproto.AffinityAware{Key: "region", Ratio: 50}
	require.NoError(t, governor.CreateRule(affinityRule))
	t.Cleanup(func() { _ = governor.DeleteRule(affinityRule) })

	assertZKRule(t, zkAddress, affinityName, "affinityAware:", "affinity:\n")
	waitForConfigEvent(t, affinityEvents.events, remoting.EventTypeAdd)
	waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool {
		return len(result) == 1 && result[0].GetURL().Port == "20880"
	})

	// A ratio above the matching proportion makes Affinity fall back to the
	// unfiltered invoker list. This proves update events replace router state.
	affinityRule.Spec.Affinity.Ratio = 60
	require.NoError(t, governor.UpdateRule(affinityRule))
	// Older Dubbo-Go ZK listeners report a data change as Add. Both router
	// implementations intentionally reload on either event while the runtime
	// listener compatibility fix is rolling out.
	waitForConfigEvent(t, affinityEvents.events, remoting.EventTypeUpdate, remoting.EventTypeAdd)
	waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool { return len(result) == 2 })
	require.NoError(t, governor.DeleteRule(affinityRule))
	waitForConfigEvent(t, affinityEvents.events, remoting.EventTypeDel)
	waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool { return len(result) == 2 })

	scriptName := providerApplication + ".script-router"
	scriptEvents := newConfigEventRecorder()
	dynamicConfig.AddListener(scriptName, scriptEvents)
	scriptRule := meshresource.NewScriptRouteResourceWithAttributes(scriptName, "router-rule-chain-e2e")
	scriptRule.Spec.ConfigVersion = "v3.0"
	scriptRule.Spec.Scope = "application"
	scriptRule.Spec.Key = providerApplication
	scriptRule.Spec.Enabled = true
	scriptRule.Spec.Type = "javascript"
	// Keep the script deliberately small and use the Router's documented
	// invoker-array contract. The selected invoker makes the assertion below
	// independent of JavaScript reflection details on common.URL.
	scriptRule.Spec.Script = `
(function (invokers, invocation, context) {
  return [invokers[1]];
})(invokers, invocation, context);`
	require.NoError(t, governor.CreateRule(scriptRule))
	t.Cleanup(func() { _ = governor.DeleteRule(scriptRule) })

	assertZKRule(t, zkAddress, scriptName, "type: javascript", "")
	waitForConfigEvent(t, scriptEvents.events, remoting.EventTypeAdd)
	initialScriptRoute := waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool {
		return len(result) == 1
	})
	initialScriptPort := initialScriptRoute[0].GetURL().Port

	// Updating the script moves traffic to the other provider. Deletion then
	// resets ScriptRouter and restores the complete invoker set.
	scriptRule.Spec.Script = `
(function (invokers, invocation, context) {
  return [invokers[0]];
})(invokers, invocation, context);`
	require.NoError(t, governor.UpdateRule(scriptRule))
	waitForConfigEvent(t, scriptEvents.events, remoting.EventTypeUpdate, remoting.EventTypeAdd)
	waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool {
		return len(result) == 1 && result[0].GetURL().Port != initialScriptPort
	})
	require.NoError(t, governor.DeleteRule(scriptRule))
	waitForConfigEvent(t, scriptEvents.events, remoting.EventTypeDel)
	waitForRoute(t, chain, consumerURL, inv, func(result []base.Invoker) bool { return len(result) == 2 })
}

func mustURL(t *testing.T, raw string) *common.URL {
	t.Helper()
	url, err := common.NewURL(raw)
	require.NoError(t, err)
	return url
}

func waitForRoute(
	t *testing.T,
	routerChain *chain.RouterChain,
	consumerURL *common.URL,
	invocation base.Invocation,
	match func([]base.Invoker) bool,
) []base.Invoker {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var last []base.Invoker
	for time.Now().Before(deadline) {
		if routed := routerChain.Route(consumerURL, invocation); match(routed) {
			return routed
		} else {
			last = routed
		}
		time.Sleep(50 * time.Millisecond)
	}
	ports := make([]string, 0, len(last))
	for _, item := range last {
		ports = append(ports, item.GetURL().Port)
	}
	t.Fatalf("RouterChain did not reach the expected routing result before %s; last ports: %s",
		deadline.Format(time.RFC3339), strings.Join(ports, ","))
	return nil
}

func assertZKRule(t *testing.T, zkAddress, name, expected, unexpected string) {
	t.Helper()
	conn, err := clients.NewZKConnection(zkAddress)
	require.NoError(t, err)
	t.Cleanup(conn.Close)
	path := "/dubbo/config/dubbo/" + name
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		content, _, getErr := conn.Get(path)
		if getErr == nil {
			actual := string(content)
			require.Contains(t, actual, expected)
			if unexpected != "" {
				require.NotContains(t, actual, unexpected)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("ZooKeeper rule %s was not published at %s", name, path)
}

func waitForConfigEvent(
	t *testing.T,
	events <-chan *dubboconfigcenter.ConfigChangeEvent,
	expected ...remoting.EventType,
) {
	t.Helper()
	select {
	case event := <-events:
		require.Contains(t, expected, event.ConfigType)
	case <-time.After(10 * time.Second):
		t.Fatalf("did not receive one of %v from the Dubbo-Go configuration listener", expected)
	}
}

type configEventRecorder struct {
	events chan *dubboconfigcenter.ConfigChangeEvent
}

func newConfigEventRecorder() *configEventRecorder {
	return &configEventRecorder{events: make(chan *dubboconfigcenter.ConfigChangeEvent, 8)}
}

func (r *configEventRecorder) Process(event *dubboconfigcenter.ConfigChangeEvent) {
	select {
	case r.events <- event:
	default:
	}
}

type e2eStoreRouter struct {
	stores map[coremodel.ResourceKind]store.ResourceStore
}

func newE2EStoreRouter(t *testing.T) *e2eStoreRouter {
	t.Helper()
	stores := make(map[coremodel.ResourceKind]store.ResourceStore)
	for _, kind := range []coremodel.ResourceKind{meshresource.AffinityRouteKind, meshresource.ScriptRouteKind} {
		resourceStore := memoryst.NewMemoryResourceStore(kind)
		require.NoError(t, resourceStore.Init(nil))
		stores[kind] = resourceStore
	}
	return &e2eStoreRouter{stores: stores}
}

func (r *e2eStoreRouter) ResourceRoute(resource coremodel.Resource) (store.ResourceStore, error) {
	return r.ResourceKindRoute(resource.ResourceKind())
}

func (r *e2eStoreRouter) ResourceKindRoute(kind coremodel.ResourceKind) (store.ResourceStore, error) {
	resourceStore, ok := r.stores[kind]
	if !ok {
		return nil, bizerror.New(bizerror.InvalidArgument, "no E2E store for resource kind "+string(kind))
	}
	return resourceStore, nil
}

type discardEmitter struct{}

func (discardEmitter) Send(events.Event) {}
