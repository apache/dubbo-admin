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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/cache"

	_ "dubbo.apache.org/dubbo-go/v3/cluster/router/affinity"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	scriptrouter "dubbo.apache.org/dubbo-go/v3/cluster/router/script"
	"dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconfig "dubbo.apache.org/dubbo-go/v3/common/config"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	dubbogoconfigcenter "dubbo.apache.org/dubbo-go/v3/config_center"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/zookeeper"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/config/app"
	consolecfg "github.com/apache/dubbo-admin/pkg/config/console"
	configauth "github.com/apache/dubbo-admin/pkg/config/console/auth"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	storecfg "github.com/apache/dubbo-admin/pkg/config/store"
	_ "github.com/apache/dubbo-admin/pkg/console"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	"github.com/apache/dubbo-admin/pkg/core/clients"
	_ "github.com/apache/dubbo-admin/pkg/core/discovery"
	_ "github.com/apache/dubbo-admin/pkg/core/engine"
	"github.com/apache/dubbo-admin/pkg/core/events"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
	_ "github.com/apache/dubbo-admin/pkg/discovery/zk"
	_ "github.com/apache/dubbo-admin/pkg/engine/mock"
	_ "github.com/apache/dubbo-admin/pkg/governor/zk"
	_ "github.com/apache/dubbo-admin/pkg/store/memory"
)

const (
	e2eZKAddressEnv  = "DUBBO_ADMIN_E2E_ZK_ADDR"
	e2eZKMesh        = "router-rule-chain-zookeeper-e2e"
	e2eNacosMesh     = "router-rule-chain-nacos-e2e"
	e2eWait          = 15 * time.Second
	e2ePoll          = 50 * time.Millisecond
	e2eSessionSecret = "router-rule-chain-e2e-session-secret-0123456789"
	e2eConsoleUser   = "router-rule-e2e"
	e2eConsolePass   = "router-rule-e2e-password"
)

const (
	scriptSelectingFirstInvoker = `(function (invokers, invocation, context) {
  return [invokers[0]];
})(invokers, invocation, context);`
	scriptSelectingSecondInvoker = `(function (invokers, invocation, context) {
  return [invokers[1]];
})(invokers, invocation, context);`
)

type configCenter string

const (
	configCenterZookeeper configCenter = "zookeeper"
	configCenterNacos     configCenter = "nacos"
)

// TestZookeeperRouterRuleChainLifecycle starts at the public Console API, then
// verifies that the rule reaches the actual config center and Dubbo-Go router.
func TestZookeeperRouterRuleChainLifecycle(t *testing.T) {
	address := configCenterAddress(configCenterZookeeper, os.Getenv(e2eZKAddressEnv))
	if address == "" {
		t.Skipf("set %s to run the ZooKeeper integration test", e2eZKAddressEnv)
	}
	e2e := newRouterRuleChainE2E(t, requireAdminRuntimeFixture(t), configCenterZookeeper, e2eZKMesh, address)
	t.Run("Affinity", e2e.testAffinityLifecycle)
	t.Run("Script", e2e.testScriptLifecycle)
}

// The production components are registered as process-wide singletons, so
// both config-center tests share one Bootstrapped runtime and TestMain stops it.
type adminRuntimeFixture struct {
	baseURL string
	client  *http.Client
	stores  store.Router
	events  events.EventBus
	stop    chan struct{}
	done    chan error
	cleanup func()
}

var sharedFixture struct {
	once sync.Once
	val  *adminRuntimeFixture
	err  error
}

func requireAdminRuntimeFixture(t *testing.T) *adminRuntimeFixture {
	t.Helper()
	sharedFixture.once.Do(func() { sharedFixture.val, sharedFixture.err = bootstrapAdminRuntime() })
	require.NoError(t, sharedFixture.err)
	return sharedFixture.val
}

func shutdownAdminRuntimeFixture() {
	if sharedFixture.val == nil {
		return
	}
	close(sharedFixture.val.stop)
	select {
	case <-sharedFixture.val.done:
	case <-time.After(10 * time.Second):
		fmt.Fprintln(os.Stderr, "router-rule-chain E2E: runtime shutdown timed out")
	}
	if sharedFixture.val.cleanup != nil {
		sharedFixture.val.cleanup()
	}
}

func bootstrapAdminRuntime() (*adminRuntimeFixture, error) {
	zkAddress := configCenterAddress(configCenterZookeeper, os.Getenv(e2eZKAddressEnv))
	nacosAddress := configCenterAddress(configCenterNacos, os.Getenv(e2eNacosAddressEnv))
	if zkAddress == "" && nacosAddress == "" {
		return nil, fmt.Errorf("set %s or %s", e2eZKAddressEnv, e2eNacosAddressEnv)
	}
	var cleanup func()
	var err error
	if zkAddress != "" {
		cleanup, err = prepareZKRoots(zkAddress)
		if err != nil {
			return nil, err
		}
	}
	port, err := freeTCPPort()
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	cfg := app.DefaultAdminConfig()
	cfg.Store = &storecfg.Config{Type: storecfg.Memory}
	cfg.Engine = enginecfg.DefaultResourceEngineConfig()
	cfg.Engine.Type = enginecfg.Mock
	cfg.Console = consolecfg.DefaultConsoleConfig()
	cfg.Console.Port = port
	cfg.Console.Auth.Methods = []string{configauth.MethodPassword}
	cfg.Console.Auth.User = e2eConsoleUser
	cfg.Console.Auth.Password = e2eConsolePass
	cfg.Console.Auth.SessionSecret = e2eSessionSecret
	cfg.Console.Auth.SessionCookieSecure = false
	if zkAddress != "" {
		cfg.Discovery = append(cfg.Discovery, &discoverycfg.Config{ID: e2eZKMesh, Name: e2eZKMesh, Type: discoverycfg.Zookeeper,
			Address: discoverycfg.AddressConfig{Registry: zkAddress, ConfigCenter: zkAddress}})
	}
	if nacosAddress != "" {
		cfg.Discovery = append(cfg.Discovery, &discoverycfg.Config{ID: e2eNacosMesh, Name: e2eNacosMesh, Type: discoverycfg.Nacos2,
			Address:    discoverycfg.AddressConfig{Registry: nacosAddress, ConfigCenter: nacosAddress},
			Properties: discoverycfg.Properties{ConfigWatchPeriod: 1, ServiceWatchPeriod: 1}})
	}
	if err := cfg.PreProcess(); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	rt, err := bootstrap.Bootstrap(context.Background(), cfg)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	eventComponent, err := rt.GetComponent(runtime.EventBus)
	if err != nil {
		return nil, err
	}
	eventBus, ok := eventComponent.(events.EventBus)
	if !ok {
		return nil, fmt.Errorf("EventBus has unexpected type")
	}
	storeComponent, err := rt.GetComponent(runtime.ResourceStore)
	if err != nil {
		return nil, err
	}
	stores, ok := storeComponent.(store.Router)
	if !ok {
		return nil, fmt.Errorf("ResourceStore has unexpected type")
	}
	stop, done := make(chan struct{}), make(chan error, 1)
	go func() { done <- rt.Start(stop) }()
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)
	client, err := loginConsole(baseURL)
	if err != nil {
		close(stop)
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
		if cleanup != nil {
			cleanup()
		}
		return nil, err
	}
	return &adminRuntimeFixture{baseURL: baseURL, client: client, stores: stores, events: eventBus, stop: stop, done: done, cleanup: cleanup}, nil
}

// configCenterAddress permits concise host:port environment values while
// preserving complete config-center URLs, including optional parameters.
func configCenterAddress(center configCenter, raw string) string {
	address := strings.TrimSpace(raw)
	if address == "" || strings.Contains(address, "://") {
		return address
	}
	return string(center) + "://" + address
}

func loginConsole(baseURL string) (*http.Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Jar: jar, Timeout: e2eWait}
	deadline := time.Now().Add(e2eWait)
	var lastErr error
	for time.Now().Before(deadline) {
		form := url.Values{"user": {e2eConsoleUser}, "password": {e2eConsolePass}}
		response, requestErr := client.Post(baseURL+"/api/v1/auth/login", "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
		if requestErr == nil {
			var body struct {
				Code string `json:"code"`
			}
			decodeErr := json.NewDecoder(response.Body).Decode(&body)
			_ = response.Body.Close()
			if decodeErr == nil && response.StatusCode == http.StatusOK && body.Code == "Success" {
				return client, nil
			}
			lastErr = fmt.Errorf("login response status=%d code=%q decode=%v", response.StatusCode, body.Code, decodeErr)
		} else {
			lastErr = requestErr
		}
		time.Sleep(e2ePoll)
	}
	return nil, fmt.Errorf("console login did not become ready: %w", lastErr)
}

func freeTCPPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
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

type routerRuleChainE2E struct {
	fixture  *adminRuntimeFixture
	center   configCenter
	mesh     string
	dynamic  dubbogoconfigcenter.DynamicConfiguration
	listener *configEventRecorder
	consumer *common.URL
	chain    *chain.RouterChain
	invoke   base.Invocation
	provider string
	zkConn   *zk.Conn
	zkEvents *resourceEventRecorder
}

func newRouterRuleChainE2E(t *testing.T, fixture *adminRuntimeFixture, center configCenter, mesh, address string) *routerRuleChainE2E {
	t.Helper()
	extension.SetRouterFactory(dubbogoconstant.ScriptRouterFactoryKey, scriptrouter.NewScriptRouterFactory)
	url := mustURL(t, address)
	if center == configCenterNacos {
		url.AddParam(dubbogoconstant.ClientNameKey, fmt.Sprintf("router-rule-chain-%d", time.Now().UnixNano()))
		url.AddParam(dubbogoconstant.ConfigGroupKey, constants.NacosConfigGroup)
		if level := nacosSDKLogLevel(); level != "" {
			url.AddParam(dubbogoconstant.NacosLogLevelKey, level)
		}
	}
	factory, err := extension.GetConfigCenterFactory(string(center))
	require.NoError(t, err)
	dynamic, err := factory.GetDynamicConfiguration(url)
	require.NoError(t, err)
	old := dubbogoconfig.GetEnvInstance().GetDynamicConfiguration()
	dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(dynamic)
	t.Cleanup(func() { dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(old) })
	provider := fmt.Sprintf("router-rule-provider-%s-%d", center, time.Now().UnixNano())
	consumer := mustURL(t, fmt.Sprintf("consumer://127.0.0.1:20000/com.example.RouterRuleE2E?application=consumer-%s&region=beijing", provider))
	routerChain, err := chain.NewRouterChain(consumer)
	require.NoError(t, err)
	routerChain.SetInvokers([]base.Invoker{
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf("dubbo://127.0.0.1:20880/com.example.RouterRuleE2E?application=%s&region=beijing", provider))),
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf("dubbo://127.0.0.1:20881/com.example.RouterRuleE2E?application=%s&region=shanghai", provider))),
	})
	e2e := &routerRuleChainE2E{fixture: fixture, center: center, mesh: mesh, dynamic: dynamic, listener: newConfigEventRecorder(), consumer: consumer, chain: routerChain, invoke: invocation.NewRPCInvocation("sayHello", nil, nil), provider: provider}
	if center == configCenterZookeeper {
		e2e.zkConn, err = clients.NewZKConnection(address)
		require.NoError(t, err)
		t.Cleanup(e2e.zkConn.Close)
		e2e.zkEvents = e2e.subscribe(t, meshresource.ZKConfigKind, "zk-config")
	}
	return e2e
}

func (e *routerRuleChainE2E) testAffinityLifecycle(t *testing.T) {
	rule := e.newAffinityRule()
	ruleEvents := e.subscribe(t, meshresource.AffinityRouteKind, "affinity")
	e.dynamic.AddListener(rule.Name, e.listener)
	e.createAffinityRule(t, rule)
	e.requireUpsert(t, rule, ruleEvents, cache.Added, "ratio: 50", remoting.EventTypeAdd, affinityRatio(50))
	waitForRoute(t, e.chain, e.consumer, e.invoke, onePort("20880"))
	updated := rule.DeepCopyObject().(*meshresource.AffinityRouteResource)
	updated.Spec.Affinity.Ratio = 60
	e.updateAffinityRule(t, updated)
	e.requireUpsert(t, updated, ruleEvents, cache.Updated, "ratio: 60", remoting.EventTypeUpdate, affinityRatio(60))
	waitForRoute(t, e.chain, e.consumer, e.invoke, twoPorts)
	e.deleteRule(t, "affinity-rule", updated.Name)
	e.requireDelete(t, updated, ruleEvents, remoting.EventTypeDel)
	waitForRoute(t, e.chain, e.consumer, e.invoke, twoPorts)
}

func (e *routerRuleChainE2E) testScriptLifecycle(t *testing.T) {
	rule := e.newScriptRule()
	ruleEvents := e.subscribe(t, meshresource.ScriptRouteKind, "script")
	e.dynamic.AddListener(rule.Name, e.listener)
	e.createScriptRule(t, rule)
	e.requireUpsert(t, rule, ruleEvents, cache.Added, "invokers[1]", remoting.EventTypeAdd, scriptBody(scriptSelectingSecondInvoker))
	waitForRoute(t, e.chain, e.consumer, e.invoke, onePort("20881"))
	updated := rule.DeepCopyObject().(*meshresource.ScriptRouteResource)
	updated.Spec.Script = scriptSelectingFirstInvoker
	e.updateScriptRule(t, updated)
	e.requireUpsert(t, updated, ruleEvents, cache.Updated, "invokers[0]", remoting.EventTypeUpdate, scriptBody(scriptSelectingFirstInvoker))
	waitForRoute(t, e.chain, e.consumer, e.invoke, onePort("20880"))
	e.deleteRule(t, "script-rule", updated.Name)
	e.requireDelete(t, updated, ruleEvents, remoting.EventTypeDel)
	waitForRoute(t, e.chain, e.consumer, e.invoke, twoPorts)
}

func (e *routerRuleChainE2E) createAffinityRule(t *testing.T, rule *meshresource.AffinityRouteResource) {
	t.Helper()
	e.callRuleAPI(t, http.MethodPost, "affinity-rule", rule.Name, map[string]any{
		"configVersion": rule.Spec.ConfigVersion,
		"scope":         rule.Spec.Scope,
		"key":           rule.Spec.Key,
		"runtime":       rule.Spec.Runtime,
		"enabled":       rule.Spec.Enabled,
		"affinityAware": map[string]any{"key": rule.Spec.Affinity.Key, "ratio": rule.Spec.Affinity.Ratio},
	})
}

func (e *routerRuleChainE2E) updateAffinityRule(t *testing.T, rule *meshresource.AffinityRouteResource) {
	t.Helper()
	e.callRuleAPI(t, http.MethodPut, "affinity-rule", rule.Name, map[string]any{
		"configVersion": rule.Spec.ConfigVersion,
		"scope":         rule.Spec.Scope,
		"key":           rule.Spec.Key,
		"runtime":       rule.Spec.Runtime,
		"enabled":       rule.Spec.Enabled,
		"affinityAware": map[string]any{"key": rule.Spec.Affinity.Key, "ratio": rule.Spec.Affinity.Ratio},
	})
}

func (e *routerRuleChainE2E) createScriptRule(t *testing.T, rule *meshresource.ScriptRouteResource) {
	t.Helper()
	e.callRuleAPI(t, http.MethodPost, "script-rule", rule.Name, map[string]any{
		"configVersion": rule.Spec.ConfigVersion,
		"scope":         rule.Spec.Scope,
		"key":           rule.Spec.Key,
		"enabled":       rule.Spec.Enabled,
		"type":          rule.Spec.Type,
		"script":        rule.Spec.Script,
	})
}

func (e *routerRuleChainE2E) updateScriptRule(t *testing.T, rule *meshresource.ScriptRouteResource) {
	t.Helper()
	e.callRuleAPI(t, http.MethodPut, "script-rule", rule.Name, map[string]any{
		"configVersion": rule.Spec.ConfigVersion,
		"scope":         rule.Spec.Scope,
		"key":           rule.Spec.Key,
		"enabled":       rule.Spec.Enabled,
		"type":          rule.Spec.Type,
		"script":        rule.Spec.Script,
	})
}

func (e *routerRuleChainE2E) deleteRule(t *testing.T, kind, name string) {
	t.Helper()
	e.callRuleAPI(t, http.MethodDelete, kind, name, nil)
}

func (e *routerRuleChainE2E) callRuleAPI(t *testing.T, method, kind, name string, payload any) {
	t.Helper()
	var body []byte
	var err error
	if payload != nil {
		body, err = json.Marshal(payload)
		require.NoError(t, err)
	}
	request, err := http.NewRequest(method, e.fixture.baseURL+"/api/v1/"+kind+"/"+url.PathEscape(name)+"?mesh="+url.QueryEscape(e.mesh), bytes.NewReader(body))
	require.NoError(t, err)
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := e.fixture.client.Do(request)
	require.NoError(t, err)
	defer response.Body.Close()
	var result struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	require.NoError(t, json.NewDecoder(response.Body).Decode(&result))
	require.Equal(t, http.StatusOK, response.StatusCode, "%s %s: %s", method, kind, result.Message)
	require.Equal(t, "Success", result.Code, "%s %s: %s", method, kind, result.Message)
}

func (e *routerRuleChainE2E) newAffinityRule() *meshresource.AffinityRouteResource {
	rule := meshresource.NewAffinityRouteResourceWithAttributes(e.provider+".affinity-router", e.mesh)
	rule.Spec.ConfigVersion, rule.Spec.Scope, rule.Spec.Key = "v3.1", "application", e.provider
	rule.Spec.Runtime, rule.Spec.Enabled = true, true
	rule.Spec.Affinity = &meshproto.AffinityAware{Key: "region", Ratio: 50}
	return rule
}

func (e *routerRuleChainE2E) newScriptRule() *meshresource.ScriptRouteResource {
	rule := meshresource.NewScriptRouteResourceWithAttributes(e.provider+".script-router", e.mesh)
	rule.Spec.ConfigVersion, rule.Spec.Scope, rule.Spec.Key = "v3.0", "application", e.provider
	rule.Spec.Enabled, rule.Spec.Type, rule.Spec.Script = true, "javascript", scriptSelectingSecondInvoker
	return rule
}

func (e *routerRuleChainE2E) requireUpsert(t *testing.T, rule coremodel.Resource, events *resourceEventRecorder, rawType cache.DeltaType, content string, listenerType remoting.EventType, match func(coremodel.Resource) bool) {
	t.Helper()
	e.waitConfigCenter(t, rule.ResourceMeta().Name, content, true)
	e.waitStore(t, rule.ResourceKind(), rule.ResourceKey(), true, match)
	if e.center == configCenterZookeeper {
		e.zkEvents.waitFor(t, rule.ResourceKey(), rawType, "")
		e.waitStore(t, meshresource.ZKConfigKind, rule.ResourceKey(), true, func(resource coremodel.Resource) bool {
			config, ok := resource.(*meshresource.ZKConfigResource)
			return ok && config.Spec != nil && config.Spec.NodeName == rule.ResourceMeta().Name && strings.Contains(config.Spec.NodeData, content)
		})
		// The source context belongs to the typed rule event emitted by
		// ZKConfigEventSubscriber, not to the raw informer event.
		events.waitFor(t, rule.ResourceKey(), cache.Updated, "zookeeper")
	} else {
		// Nacos ConfigListerWatcher materializes typed rules directly; it has
		// no NacosConfig intermediate resource or source context. Its delete
		// event is asserted below, where it cannot be confused with the
		// governor's synchronous local-store removal.
	}
	e.listener.waitFor(t, rule.ResourceMeta().Name, e.listenerEventType(listenerType))
}

func (e *routerRuleChainE2E) requireDelete(t *testing.T, rule coremodel.Resource, events *resourceEventRecorder, listenerType remoting.EventType) {
	t.Helper()
	e.waitConfigCenter(t, rule.ResourceMeta().Name, "", false)
	if e.center == configCenterZookeeper {
		e.zkEvents.waitFor(t, rule.ResourceKey(), cache.Deleted, "")
		e.waitStore(t, meshresource.ZKConfigKind, rule.ResourceKey(), false, nil)
	} else {
		// The Nacos governor removes its local item synchronously. This delete
		// event is therefore produced by Nacos discovery, not the governor.
		events.waitFor(t, rule.ResourceKey(), cache.Deleted, "")
	}
	e.waitStore(t, rule.ResourceKind(), rule.ResourceKey(), false, nil)
	e.listener.waitFor(t, rule.ResourceMeta().Name, e.listenerEventType(listenerType))
}

func (e *routerRuleChainE2E) listenerEventType(expected remoting.EventType) remoting.EventType {
	if e.center == configCenterNacos {
		// Nacos delivers every configuration transition, including removal of a
		// dataId, as an Update callback. Removal is represented by an empty
		// payload and is separately checked through the Admin discovery delete.
		return remoting.EventTypeUpdate
	}
	if expected == remoting.EventTypeUpdate {
		// The pinned Dubbo-Go ZooKeeper cache listener reports znode data
		// changes as Add. The RouterChain assertion verifies replacement.
		return remoting.EventTypeAdd
	}
	return expected
}

func (e *routerRuleChainE2E) subscribe(t *testing.T, kind coremodel.ResourceKind, suffix string) *resourceEventRecorder {
	t.Helper()
	recorder := newResourceEventRecorder(fmt.Sprintf("%s-%d", suffix, time.Now().UnixNano()), kind)
	require.NoError(t, e.fixture.events.Subscribe(recorder))
	t.Cleanup(func() { _ = e.fixture.events.Unsubscribe(recorder) })
	return recorder
}

func (e *routerRuleChainE2E) waitConfigCenter(t *testing.T, name, content string, exists bool) {
	t.Helper()
	deadline := time.Now().Add(e2eWait)
	for time.Now().Before(deadline) {
		if e.center == configCenterZookeeper {
			data, _, getErr := e.zkConn.Get("/dubbo/config/dubbo/" + name)
			legacy, _, legacyErr := e.zkConn.Exists("/dubbo/config/" + name)
			if exists && getErr == nil && legacyErr == nil && !legacy && strings.Contains(string(data), content) {
				return
			}
			if !exists && getErr == zk.ErrNoNode && legacyErr == nil && !legacy {
				return
			}
		} else if value, err := e.dynamic.GetRule(name); err == nil && ((exists && strings.Contains(value, content)) || (!exists && value == "")) {
			return
		}
		time.Sleep(e2ePoll)
	}
	t.Fatalf("%s did not reach expected config state for %s", e.center, name)
}

func (e *routerRuleChainE2E) waitStore(t *testing.T, kind coremodel.ResourceKind, key string, exists bool, match func(coremodel.Resource) bool) {
	t.Helper()
	resourceStore, err := e.fixture.stores.ResourceKindRoute(kind)
	require.NoError(t, err)
	deadline := time.Now().Add(e2eWait)
	for time.Now().Before(deadline) {
		item, found, getErr := resourceStore.GetByKey(key)
		if getErr == nil && found == exists {
			if !exists {
				return
			}
			if resource, ok := item.(coremodel.Resource); ok && match(resource) {
				return
			}
		}
		time.Sleep(e2ePoll)
	}
	t.Fatalf("%s store did not reach expected state for %s", kind, key)
}

func affinityRatio(want int32) func(coremodel.Resource) bool {
	return func(resource coremodel.Resource) bool {
		r, ok := resource.(*meshresource.AffinityRouteResource)
		return ok && r.Spec.Affinity != nil && r.Spec.Affinity.Ratio == want
	}
}
func scriptBody(want string) func(coremodel.Resource) bool {
	return func(resource coremodel.Resource) bool {
		r, ok := resource.(*meshresource.ScriptRouteResource)
		return ok && r.Spec.Script == want
	}
}
func mustURL(t *testing.T, raw string) *common.URL {
	t.Helper()
	url, err := common.NewURL(raw)
	require.NoError(t, err)
	return url
}
func onePort(port string) func([]base.Invoker) bool {
	return func(result []base.Invoker) bool { return len(result) == 1 && result[0].GetURL().Port == port }
}
func twoPorts(result []base.Invoker) bool { return len(result) == 2 }
func waitForRoute(t *testing.T, routerChain *chain.RouterChain, consumer *common.URL, invocation base.Invocation, match func([]base.Invoker) bool) {
	t.Helper()
	deadline := time.Now().Add(e2eWait)
	for time.Now().Before(deadline) {
		if match(routerChain.Route(consumer, invocation)) {
			return
		}
		time.Sleep(e2ePoll)
	}
	t.Fatal("RouterChain did not reach the expected routing result")
}

type recordedEvent struct {
	typ      cache.DeltaType
	resource coremodel.Resource
	context  map[string]string
}
type resourceEventRecorder struct {
	name   string
	kind   coremodel.ResourceKind
	events chan recordedEvent
}

func newResourceEventRecorder(name string, kind coremodel.ResourceKind) *resourceEventRecorder {
	return &resourceEventRecorder{name: "router-rule-chain-e2e-" + name, kind: kind, events: make(chan recordedEvent, 64)}
}
func (r *resourceEventRecorder) ResourceKind() coremodel.ResourceKind { return r.kind }
func (r *resourceEventRecorder) Name() string                         { return r.name }
func (r *resourceEventRecorder) AsyncEnabled() bool                   { return false }
func (r *resourceEventRecorder) ProcessEvent(event events.Event) error {
	resource := event.NewObj()
	if resource == nil {
		resource = event.OldObj()
	}
	contextCopy := make(map[string]string, len(event.Context()))
	for key, value := range event.Context() {
		contextCopy[key] = value
	}
	r.events <- recordedEvent{typ: event.Type(), resource: resource, context: contextCopy}
	return nil
}
func (r *resourceEventRecorder) waitFor(t *testing.T, key string, typ cache.DeltaType, source string) {
	t.Helper()
	timer := time.NewTimer(e2eWait)
	defer timer.Stop()
	for {
		select {
		case event := <-r.events:
			if event.resource != nil && event.resource.ResourceKey() == key && event.typ == typ && (source == "" || event.context[events.SourceRegistryContextKey] == source) {
				return
			}
		case <-timer.C:
			t.Fatalf("did not receive %s %s for %s source=%q", r.kind, typ, key, source)
		}
	}
}

type configEventRecorder struct {
	events chan *dubbogoconfigcenter.ConfigChangeEvent
}

func newConfigEventRecorder() *configEventRecorder {
	return &configEventRecorder{events: make(chan *dubbogoconfigcenter.ConfigChangeEvent, 16)}
}
func (r *configEventRecorder) Process(event *dubbogoconfigcenter.ConfigChangeEvent) {
	copy := *event
	r.events <- &copy
}
func (r *configEventRecorder) waitFor(t *testing.T, key string, typ remoting.EventType) {
	t.Helper()
	timer := time.NewTimer(e2eWait)
	defer timer.Stop()
	for {
		select {
		case event := <-r.events:
			if event.Key == key && event.ConfigType == typ {
				return
			}
		case <-timer.C:
			t.Fatalf("Dubbo-Go listener did not receive %s for %s", typ, key)
		}
	}
}
