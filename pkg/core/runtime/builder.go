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

package runtime

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/app"
)

// BuilderContext provides access to Builder's interim state.
type BuilderContext interface {
	Config() app.AdminConfig
	GetActivatedComponent(typ ComponentType) (Component, error)
	ActivateComponent(comp Component) error
	AppContext() context.Context
}

var _ BuilderContext = &Builder{}

// Builder represents a multi-step initialization process.
type Builder struct {
	cfg        app.AdminConfig
	components map[ComponentType]Component
	appCtx     context.Context
	runtimeInfo
}

func (b *Builder) GetActivatedComponent(typ ComponentType) (Component, error) {
	comp, exists := b.components[typ]
	if !exists {
		return nil, errors.Errorf("no such component: %v", typ)
	}
	return comp, nil
}

func (b *Builder) AppContext() context.Context {
	return b.appCtx
}

func BuilderFor(appCtx context.Context, cfg app.AdminConfig) (*Builder, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not get hostname")
	}
	suffix := uuid.NewString()[0:4]
	return &Builder{
		cfg:    cfg,
		appCtx: appCtx,
		runtimeInfo: runtimeInfo{
			instanceId: fmt.Sprintf("%s-%s", hostname, suffix),
			clusterId:  fmt.Sprintf("%s-%s", hostname, suffix),
			startTime:  time.Now(),
		},
		components: make(map[ComponentType]Component),
	}, nil
}

func (b *Builder) ActivateComponent(comp Component) error {
	_, exist := b.components[comp.Type()]
	if exist {
		// if already activated
		return errors.Errorf("component with type %v has already been activated", comp.Type())
	}
	// if not activated
	b.components[comp.Type()] = comp
	return nil
}

func (b *Builder) Build() (Runtime, error) {
	for _, typ := range CoreComponentTypes {
		if _, exists := b.components[typ]; !exists {
			return nil, errors.Errorf("%v has not been configured", typ)
		}
	}
	rt := &runtime{
		runtimeInfo: b.runtimeInfo,
		runtimeContext: runtimeContext{
			cfg:        b.cfg,
			appCtx:     b.appCtx,
			components: b.components,
		},
	}
	return rt, nil
}

func (b *Builder) Config() app.AdminConfig {
	return b.cfg
}

func (b *Builder) AppCtx() context.Context {
	return b.appCtx
}
