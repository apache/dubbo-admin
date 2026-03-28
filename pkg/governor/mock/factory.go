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
	discoverycfg "github.com/apache/dubbo-admin/pkg/config/discovery"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/governor"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

func init() {
	governor.RegisterFactory(&mockGovernorFactory{})
}

type mockGovernorFactory struct{}

var _ governor.Factory = &mockGovernorFactory{}

func (f *mockGovernorFactory) Support(t discoverycfg.Type) bool {
	return t == discoverycfg.Mock
}

func (f *mockGovernorFactory) New(_ string, _ *discoverycfg.Config, _ store.Router, _ events.Emitter) (governor.RuleGovernor, error) {
	return &mockGovernor{}, nil
}

type mockGovernor struct{}

var _ governor.RuleGovernor = &mockGovernor{}

func (g *mockGovernor) CreateRule(_ coremodel.Resource) error { return nil }
func (g *mockGovernor) UpdateRule(_ coremodel.Resource) error { return nil }
func (g *mockGovernor) DeleteRule(_ coremodel.Resource) error { return nil }
