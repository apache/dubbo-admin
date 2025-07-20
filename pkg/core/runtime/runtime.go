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
	"time"

	"github.com/duke-git/lancet/v2/maputil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/config/app"

	"github.com/apache/dubbo-admin/pkg/config/mode"
)

// Runtime represents initialized application state.
type Runtime interface {
	RuntimeInfo
	RuntimeContext
	ComponentManager
}

type RuntimeInfo interface {
	GetInstanceId() string
	GetClusterId() string
	GetStartTime() time.Time
	GetMode() mode.Mode
}

type RuntimeContext interface {
	Config() app.AdminConfig
	GetComponent(typ ComponentType) (Component, error)
	// AppContext returns a context.Context which tracks the lifetime of the apps, it gets cancelled when the app is starting to shutdown.
	AppContext() context.Context
}

var _ RuntimeInfo = &runtimeInfo{}

type runtimeInfo struct {
	instanceId string
	clusterId  string
	startTime  time.Time
	mode       mode.Mode
}

func (i *runtimeInfo) GetInstanceId() string {
	return i.instanceId
}

func (i *runtimeInfo) GetClusterId() string {
	return i.clusterId
}

func (i *runtimeInfo) GetStartTime() time.Time {
	return i.startTime
}

func (i *runtimeInfo) GetMode() mode.Mode {
	return i.mode
}

var _ RuntimeContext = &runtimeContext{}

// TODO add console
type runtimeContext struct {
	cfg        app.AdminConfig
	components map[ComponentType]Component
	appCtx     context.Context
}

func (r *runtimeContext) Config() app.AdminConfig {
	return r.cfg
}

func (r *runtimeContext) GetComponent(typ ComponentType) (Component, error) {
	if comp, exists := r.components[typ]; exists {
		return comp, nil
	}
	return nil, errors.Errorf("component %s not found", typ)
}

func (r *runtimeContext) AppContext() context.Context {
	return r.appCtx
}

var _ Runtime = &runtime{}

type runtime struct {
	runtimeInfo
	runtimeContext
}

func (rt *runtime) Add(components ...Component) {
	for _, c := range components {
		rt.components[c.Type()] = c
	}
}

func (rt *runtime) Start(stop <-chan struct{}) error {
	components := maputil.Values(rt.components)
	slice.SortBy(components, func(a, b Component) bool {
		return a.Order() < b.Order()
	})
	for _, com := range components {
		err := com.Start(rt, stop)
		if err != nil {
			return err
		}
	}
	return nil
}
