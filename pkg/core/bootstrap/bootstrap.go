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

package bootstrap

import (
	"context"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/diagnostics"
)

func Bootstrap(appCtx context.Context, cfg app.AdminConfig) (runtime.Runtime, error) {
	builder, err := runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}
	// 0. initialize event bus
	if err := initEventBus(builder); err != nil {
		return nil, err
	}
	// 1. initialize resource store
	if err := initResourceStore(cfg, builder); err != nil {
		return nil, err
	}
	// 2. initialize discovery engine
	if err := initializeResourceDiscovery(builder); err != nil {
		return nil, err
	}
	// 3. initialize runtime engine
	if err := initializeResourceEngine(builder); err != nil {
		return nil, err
	}
	// 4. initialize resource manager
	if err := initResourceManager(builder); err != nil {
		return nil, err
	}
	// 5. initialize console
	if err := initializeConsole(builder); err != nil {
		return nil, err
	}
	// 6. initialize diagnotics
	if err := initializeDiagnoticsServer(builder); err != nil {
		logger.Errorf("got error when init diagnotics server %s", err)
	}
	rt, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func initEventBus(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().EventBus()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initResourceStore(cfg app.AdminConfig, builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceStore()
	if err != nil {
		return errors.Wrapf(err, "could not retrieve resource store %s component", cfg.Store.Type)
	}
	return initAndActivateComponent(builder, comp)
}
func initResourceManager(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceManager()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeConsole(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().Console()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeResourceDiscovery(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceDiscovery()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeResourceEngine(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().ResourceEngine()
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initializeDiagnoticsServer(builder *runtime.Builder) error {
	comp, err := runtime.ComponentRegistry().Get(diagnostics.DiagnosticsServer)
	if err != nil {
		return err
	}
	return initAndActivateComponent(builder, comp)
}

func initAndActivateComponent(builder *runtime.Builder, comp runtime.Component) error {
	logger.Infof("initializing %s ...", comp.Type())
	if err := comp.Init(builder); err != nil {
		return err
	}
	logger.Infof("%s initialized successfully", comp.Type())
	if err := builder.ActivateComponent(comp); err != nil {
		return errors.Wrapf(err, "failed to activate %s", comp.Type())
	}
	return nil
}
