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
	"io"
	stdlog "log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	scriptrouter "dubbo.apache.org/dubbo-go/v3/cluster/router/script"
	dubbogocom "dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconfig "dubbo.apache.org/dubbo-go/v3/common/config"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	dubbogologger "dubbo.apache.org/dubbo-go/v3/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	gostlogger "github.com/dubbogo/gost/log/logger"
	nacoslogger "github.com/nacos-group/nacos-sdk-go/v2/common/logger"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	logcfg "github.com/apache/dubbo-admin/pkg/config/log"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	adminnacos "github.com/apache/dubbo-admin/pkg/governor/nacos2"
)

const (
	e2eNacosAddressEnv = "DUBBO_ADMIN_E2E_NACOS_ADDR"
	e2eLogEnv          = "DUBBO_ADMIN_E2E_LOG"
)

// TestMain keeps E2E output focused on test results by default. Set
// DUBBO_ADMIN_E2E_LOG to debug, info, warn, or error when diagnosing a
// propagation failure; its default value, off, silences dependency logs.
func TestMain(m *testing.M) {
	configureE2ELogging()
	os.Exit(m.Run())
}

func configureE2ELogging() {
	level := strings.ToLower(strings.TrimSpace(os.Getenv(e2eLogEnv)))
	if level == "" {
		level = "off"
	}

	if level == "off" {
		configureE2ELoggingOff()
		return
	}

	if level != string(logcfg.LevelDebug) && level != string(logcfg.LevelInfo) &&
		level != string(logcfg.LevelWarn) && level != string(logcfg.LevelError) {
		fmt.Fprintf(os.Stderr, "invalid %s=%q; using off\n", e2eLogEnv, level)
		configureE2ELoggingOff()
		return
	}
	logger.Init(&logcfg.Config{Level: logcfg.Level(level), OutputPath: os.DevNull})
	_ = dubbogologger.SetLoggerLevel(level)
	_ = gostlogger.SetLoggerLevel(level)
}

func configureE2ELoggingOff() {
	logger.Init(&logcfg.Config{Level: logcfg.LevelError, OutputPath: os.DevNull})
	// The ZooKeeper client writes connection lifecycle messages directly through
	// the standard logger, bypassing Dubbo-Go's logger facade.
	stdlog.SetOutput(io.Discard)
	discard := e2eDiscardLogger{}
	dubbogologger.SetLogger(discard)
	gostlogger.SetLogger(discard)
	nacoslogger.SetLogger(discard)
}

type e2eDiscardLogger struct{}

func (e2eDiscardLogger) Debug(...any)          {}
func (e2eDiscardLogger) Debugf(string, ...any) {}
func (e2eDiscardLogger) Info(...any)           {}
func (e2eDiscardLogger) Infof(string, ...any)  {}
func (e2eDiscardLogger) Warn(...any)           {}
func (e2eDiscardLogger) Warnf(string, ...any)  {}
func (e2eDiscardLogger) Error(...any)          {}
func (e2eDiscardLogger) Errorf(string, ...any) {}
func (e2eDiscardLogger) Fatal(...any)          {}
func (e2eDiscardLogger) Fatalf(string, ...any) {}
func (e2eDiscardLogger) Close() error          { return nil }

func TestNacosAffinityRouterRuleCreate(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	e2e.createAffinityRule(t)
}

func TestNacosAffinityRouterRuleUpdate(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	rule := e2e.createAffinityRule(t)

	rule.Spec.Affinity.Ratio = 60
	require.NoError(t, e2e.governor.UpdateRule(rule))
	waitForRoute(t, e2e.chain, e2e.consumerURL, e2e.invocation, func(result []base.Invoker) bool {
		return len(result) == 2
	})
}

func TestNacosAffinityRouterRuleDelete(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	rule := e2e.createAffinityRule(t)

	require.NoError(t, e2e.governor.DeleteRule(rule))
	waitForRoute(t, e2e.chain, e2e.consumerURL, e2e.invocation, func(result []base.Invoker) bool {
		return len(result) == 2
	})
}

func TestNacosScriptRouterRuleCreate(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	e2e.createScriptRule(t)
}

func TestNacosScriptRouterRuleUpdate(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	rule := e2e.createScriptRule(t)

	rule.Spec.Script = scriptSelectingFirstInvoker
	require.NoError(t, e2e.governor.UpdateRule(rule))
	waitForRoute(t, e2e.chain, e2e.consumerURL, e2e.invocation, func(result []base.Invoker) bool {
		return len(result) == 1 && result[0].GetURL().Port == "20880"
	})
}

func TestNacosScriptRouterRuleDelete(t *testing.T) {
	e2e := newNacosRouterRuleChainE2E(t)
	rule := e2e.createScriptRule(t)

	require.NoError(t, e2e.governor.DeleteRule(rule))
	waitForRoute(t, e2e.chain, e2e.consumerURL, e2e.invocation, func(result []base.Invoker) bool {
		return len(result) == 2
	})
}

// newNacosRouterRuleChainE2E exercises the same production path as the ZK
// fixture: Admin governor -> Nacos -> Dubbo-Go listener -> RouterChain.
func newNacosRouterRuleChainE2E(t *testing.T) *routerRuleChainE2E {
	t.Helper()
	nacosAddress := os.Getenv(e2eNacosAddressEnv)
	if nacosAddress == "" {
		t.Skipf("set %s to run the Nacos integration test", e2eNacosAddressEnv)
	}
	// Script Router registration remains opt-in in the pinned Dubbo-Go runtime.
	// Register it explicitly, exactly as the ZooKeeper fixture does.
	extension.SetRouterFactory(dubbogoconstant.ScriptRouterFactoryKey, scriptrouter.NewScriptRouterFactory)

	storeRouter := newE2EStoreRouter(t)
	governor, err := adminnacos.NewNacos2Governor(&discoverycfg.Config{
		ID:   "router-rule-chain-nacos-e2e",
		Name: "router-rule-chain-nacos-e2e",
		Type: discoverycfg.Nacos2,
		Address: discoverycfg.AddressConfig{
			Registry:     nacosAddress,
			ConfigCenter: nacosAddress,
		},
	}, storeRouter, discardEmitter{})
	require.NoError(t, err)

	configURL, err := dubbogocom.NewURL(nacosAddress)
	require.NoError(t, err)
	// Dubbo-Go's Nacos client pool requires an explicit client name. Keep it
	// stable for the fixture while giving the Admin governor its own client.
	configURL.AddParam(dubbogoconstant.ClientNameKey, "router-rule-chain-nacos-e2e-listener")
	configURL.AddParam(dubbogoconstant.ConfigGroupKey, constants.NacosConfigGroup)
	if level := nacosSDKLogLevel(); level != "" {
		configURL.AddParam(dubbogoconstant.NacosLogLevelKey, level)
	}
	dynamicConfigFactory, err := extension.GetConfigCenterFactory("nacos")
	require.NoError(t, err)
	dynamicConfig, err := dynamicConfigFactory.GetDynamicConfiguration(configURL)
	require.NoError(t, err)
	if destroyer, ok := dynamicConfig.(interface{ Destroy() }); ok {
		t.Cleanup(destroyer.Destroy)
	}

	oldDynamicConfig := dubbogoconfig.GetEnvInstance().GetDynamicConfiguration()
	dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(dynamicConfig)
	t.Cleanup(func() {
		dubbogoconfig.GetEnvInstance().SetDynamicConfiguration(oldDynamicConfig)
	})

	providerApplication := fmt.Sprintf("router-rule-nacos-e2e-%d", time.Now().UnixNano())
	consumerURL := mustURL(t, fmt.Sprintf(
		"consumer://127.0.0.1:20000/com.example.RouterRuleE2E?application=consumer-%s&region=beijing",
		providerApplication))
	routerChain, err := chain.NewRouterChain(consumerURL)
	require.NoError(t, err)
	routerChain.SetInvokers([]base.Invoker{
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf(
			"dubbo://127.0.0.1:20880/com.example.RouterRuleE2E?application=%s&region=beijing", providerApplication))),
		base.NewBaseInvoker(mustURL(t, fmt.Sprintf(
			"dubbo://127.0.0.1:20881/com.example.RouterRuleE2E?application=%s&region=shanghai", providerApplication))),
	})

	return &routerRuleChainE2E{
		governor:            governor,
		providerApplication: providerApplication,
		consumerURL:         consumerURL,
		chain:               routerChain,
		invocation:          invocation.NewRPCInvocation("sayHello", nil, nil),
	}
}

func nacosSDKLogLevel() string {
	level := strings.ToLower(strings.TrimSpace(os.Getenv(e2eLogEnv)))
	if level == "" || level == "off" {
		return string(logcfg.LevelError)
	}
	return level
}
