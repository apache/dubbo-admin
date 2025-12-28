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

package manager

import (
	"math"

	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	runtime.RegisterComponent(&resourceManagerComponent{})
}

type ResourceManagerComponent interface {
	runtime.Component
	ResourceManager() ResourceManager
}

var _ ResourceManagerComponent = &resourceManagerComponent{}

type resourceManagerComponent struct {
	rm ResourceManager
}

func (r *resourceManagerComponent) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.ResourceStore, // Manager needs Store to be initialized first
	}
}

func (r *resourceManagerComponent) Type() runtime.ComponentType {
	return runtime.ResourceManager
}

func (r *resourceManagerComponent) Order() int {
	return math.MaxInt - 4
}

func (r *resourceManagerComponent) Init(ctx runtime.BuilderContext) error {
	rsc, err := ctx.GetActivatedComponent(runtime.ResourceStore)
	if err != nil {
		return errors.Wrap(err, "failed to init resource manager")
	}
	r.rm = NewResourceManager(rsc.(store.Router))
	return nil
}

func (r *resourceManagerComponent) Start(runtime.Runtime, <-chan struct{}) error {
	return nil
}

func (r *resourceManagerComponent) ResourceManager() ResourceManager {
	return r.rm
}
