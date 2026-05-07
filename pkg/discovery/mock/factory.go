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
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/discovery"
)

func init() {
	discovery.RegisterListWatcherFactory(&mockListerWaterFactory{})
}

type mockListerWaterFactory struct{}

var _ discovery.Factory = &mockListerWaterFactory{}

func (l *mockListerWaterFactory) Support(d discoverycfg.Type) bool {
	return d == discoverycfg.Mock
}

func (l *mockListerWaterFactory) NewListWatchers(_ *discoverycfg.Config) ([]controller.ResourceListerWatcher, error) {
	return make([]controller.ResourceListerWatcher, 0), nil
}
