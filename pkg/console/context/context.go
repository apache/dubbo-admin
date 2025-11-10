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

package context

import (
	ctx "context"

	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/console/counter"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

type Context interface {
	ResourceManager() manager.ResourceManager
	CounterManager() counter.CounterManager

	Config() app.AdminConfig

	AppContext() ctx.Context
}

var _ Context = &context{}

func NewConsoleContext(coreRt runtime.Runtime) Context {
	return &context{
		coreRt: coreRt,
	}
}

type context struct {
	coreRt runtime.Runtime
}

func (c *context) AppContext() ctx.Context {
	return c.coreRt.AppContext()
}

func (c *context) Config() app.AdminConfig {
	return c.coreRt.Config()
}

func (c *context) ResourceManager() manager.ResourceManager {
	rmc, _ := c.coreRt.GetComponent(runtime.ResourceManager)
	return rmc.(manager.ResourceManagerComponent).ResourceManager()
}

func (c *context) CounterManager() counter.CounterManager {
	comp, err := c.coreRt.GetComponent(counter.ComponentType)
	if err != nil {
		return nil
	}
	managerComp, ok := comp.(counter.ManagerComponent)
	if !ok {
		return nil
	}
	return managerComp.CounterManager()
}
