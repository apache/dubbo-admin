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

	"k8s.io/client-go/tools/cache"

	"dubbo.apache.org/dubbo-go/v3/common"
	dubbogoconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/config_center/nacos"
	dubbogologger "dubbo.apache.org/dubbo-go/v3/logger"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	gostlogger "github.com/dubbogo/gost/log/logger"
	nacoslogger "github.com/nacos-group/nacos-sdk-go/v2/common/logger"

	"github.com/apache/dubbo-admin/pkg/common/constants"
	logcfg "github.com/apache/dubbo-admin/pkg/config/log"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	_ "github.com/apache/dubbo-admin/pkg/discovery/nacos2"
	_ "github.com/apache/dubbo-admin/pkg/governor/nacos2"
)

const (
	e2eNacosAddressEnv = "DUBBO_ADMIN_E2E_NACOS_ADDR"
	e2eLogEnv          = "DUBBO_ADMIN_E2E_LOG"
)

func TestMain(m *testing.M) {
	configureE2ELogging()
	code := m.Run()
	shutdownAdminRuntimeFixture()
	os.Exit(code)
}

// TestNacosRouterRuleChainLifecycle uses the same real Admin runtime as the
// ZooKeeper test. Nacos discovery converts configurations directly into typed
// route resources, so there is no NacosConfig wrapper equivalent to ZKConfig.
func TestNacosRouterRuleChainLifecycle(t *testing.T) {
	address := configCenterAddress(configCenterNacos, os.Getenv(e2eNacosAddressEnv))
	if address == "" {
		t.Skipf("set %s to run the Nacos integration test", e2eNacosAddressEnv)
	}
	e2e := newNacosRouterRuleChainE2E(t, requireAdminRuntimeFixture(t), address)
	t.Run("Affinity", e2e.testAffinityLifecycle)
	t.Run("Script", e2e.testScriptLifecycle)
}

func newNacosRouterRuleChainE2E(t *testing.T, fixture *adminRuntimeFixture, address string) *routerRuleChainE2E {
	t.Helper()
	return newRouterRuleChainE2E(t, fixture, configCenterNacos, e2eNacosMesh, address, nacosConfigCenterE2EAdapter{})
}

// nacosConfigCenterE2EAdapter documents Nacos-specific propagation behavior.
// Its ListWatcher materializes typed rules directly, without a raw config
// resource equivalent to ZooKeeper's ZKConfig.
type nacosConfigCenterE2EAdapter struct{}

func (nacosConfigCenterE2EAdapter) configureURL(url *common.URL) {
	url.AddParam(dubbogoconstant.ClientNameKey, fmt.Sprintf("router-rule-chain-%d", time.Now().UnixNano()))
	url.AddParam(dubbogoconstant.ConfigGroupKey, constants.NacosConfigGroup)
	if level := nacosSDKLogLevel(); level != "" {
		url.AddParam(dubbogoconstant.NacosLogLevelKey, level)
	}
}

func (nacosConfigCenterE2EAdapter) configMatches(e2e *routerRuleChainE2E, name, content string, exists bool) bool {
	value, err := e2e.dynamic.GetRule(name)
	return err == nil && ((exists && strings.Contains(value, content)) || (!exists && value == ""))
}

func (nacosConfigCenterE2EAdapter) verifyUpsert(*testing.T, *routerRuleChainE2E, coremodel.Resource, *resourceEventRecorder, cache.DeltaType, string) {
	// The typed route resource is asserted by the shared store check.
}

func (nacosConfigCenterE2EAdapter) verifyDelete(t *testing.T, _ *routerRuleChainE2E, rule coremodel.Resource, events *resourceEventRecorder) {
	t.Helper()
	// Nacos governor removes its local item synchronously, so this event is
	// produced by Nacos discovery rather than by the governor itself.
	events.waitFor(t, rule.ResourceKey(), cache.Deleted, "")
}

func (nacosConfigCenterE2EAdapter) listenerEventType(remoting.EventType) remoting.EventType {
	// Nacos represents both updates and removals as Update callbacks; removal
	// is separately verified through the Admin discovery delete event above.
	return remoting.EventTypeUpdate
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
	if level != string(logcfg.LevelDebug) && level != string(logcfg.LevelInfo) && level != string(logcfg.LevelWarn) && level != string(logcfg.LevelError) {
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
	stdlog.SetOutput(io.Discard)
	discard := e2eDiscardLogger{}
	dubbogologger.SetLogger(discard)
	gostlogger.SetLogger(discard)
	nacoslogger.SetLogger(discard)
}

func nacosSDKLogLevel() string {
	level := strings.ToLower(strings.TrimSpace(os.Getenv(e2eLogEnv)))
	if level == "" || level == "off" {
		return string(logcfg.LevelError)
	}
	return level
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
