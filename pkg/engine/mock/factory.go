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

package mock

import (
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/engine"
)

func init() {
	engine.RegisterFactory(&EngineFactory{})
}

var _ engine.Factory = &EngineFactory{}

type EngineFactory struct{}

func (e *EngineFactory) Support(typ enginecfg.Type) bool {
	return enginecfg.Mock == typ
}

func (e *EngineFactory) NewListWatchers(cfg *enginecfg.Config) ([]controller.ResourceListerWatcher, error) {
	return make([]controller.ResourceListerWatcher, 0), nil
}
