// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controlplane

import (
	"errors"
	"fmt"

	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
)

type DubboControlPlane struct {
	spec       *v1alpha1.DubboOperatorSpec
	started    bool
	components map[ComponentName]Component
}

func (dcp *DubboControlPlane) Run() error {
	for name, component := range dcp.components {
		if err := component.Run(); err != nil {
			return fmt.Errorf("component %s run failed, err: %s", name, err)
		}
	}
	dcp.started = true
	return nil
}

func (dcp *DubboControlPlane) RenderManifest() (map[ComponentName]string, error) {
	if !dcp.started {
		return nil, errors.New("DubboControlPlane is not running")
	}
	res := make(map[ComponentName]string)
	for name, component := range dcp.components {
		manifest, err := component.RenderManifest()
		if err != nil {
			return nil, fmt.Errorf("component %s RenderManifest err: %v", name, err)
		}
		res[name] = manifest
	}
	return res, nil
}

func NewDubboControlPlane(spec *v1alpha1.DubboOperatorSpec) (*DubboControlPlane, error) {
	if spec == nil {
		return nil, errors.New("DubboOperatorSpec is empty")
	}
	// initialized
	components := make(map[ComponentName]Component)
	if spec.IsAdminEnabled() {
		admin, err := NewAdminComponent(spec.Components.Admin, spec.ComponentsMeta.Admin.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Admin] = admin
	}
	if spec.IsGrafanaEnabled() {
		grafana, err := NewGrafanaComponent(spec.Components.Grafana, spec.ComponentsMeta.Grafana.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Grafana] = grafana
	}
	if spec.IsNacosEnabled() {
		nacos, err := NewNacosComponent(spec.Components.Nacos, spec.ComponentsMeta.Nacos.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Nacos] = nacos
	}
	if spec.IsZookeeperEnabled() {
		zookeeper, err := NewZookeeperComponent(spec.Components.Zookeeper, spec.ComponentsMeta.Zookeeper.Namespace, spec.ChartPath)
		if err != nil {
			return nil, err
		}
		components[Zookeeper] = zookeeper
	}
	dcp := &DubboControlPlane{
		spec:       spec,
		components: components,
	}
	return dcp, nil
}
