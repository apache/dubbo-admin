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

package operator

import (
	"errors"
	"fmt"

	"github.com/apache/dubbo-admin/pkg/dubboctl/identifier"
	"github.com/apache/dubbo-admin/pkg/kube"

	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
)

type DubboOperator struct {
	spec       *v1alpha1.DubboConfigSpec
	started    bool
	components map[ComponentName]Component
	kubeCli    *kube.CtlClient
	namespace  string
}

func (do *DubboOperator) Run() error {
	for name, component := range do.components {
		if err := component.Run(); err != nil {
			return fmt.Errorf("component %s run failed, err: %s", name, err)
		}
	}
	do.started = true
	return nil
}

func (do *DubboOperator) RenderManifest() (map[ComponentName]string, error) {
	if !do.started {
		return nil, errors.New("DubboOperator is not running")
	}
	res := make(map[ComponentName]string)
	for name, component := range do.components {
		manifest, err := component.RenderManifest()
		if err != nil {
			return nil, fmt.Errorf("component %s RenderManifest err: %v", name, err)
		}
		res[name] = manifest
	}
	return res, nil
}

func (do *DubboOperator) ApplyManifest(manifestMap map[ComponentName]string) error {
	for _, manifest := range manifestMap {
		if err := do.kubeCli.ApplyManifest(manifest, do.namespace); err != nil {
			// log component
			return err
		}
	}
	return nil
}

func NewDubboOperator(spec *v1alpha1.DubboConfigSpec, kubecfgPath string, ctx string, dryRun bool) (*DubboOperator, error) {
	if spec == nil {
		return nil, errors.New("DubboConfigSpec is empty")
	}
	ns := spec.Namespace
	if ns == "" {
		ns = identifier.DubboSystemNamespace
	}
	// initialized
	components := make(map[ComponentName]Component)
	if spec.IsAdminEnabled() {
		admin, err := NewAdminComponent(spec.Components.Admin,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
		)
		if err != nil {
			return nil, err
		}
		components[Admin] = admin
	}
	if spec.IsGrafanaEnabled() {
		grafana, err := NewGrafanaComponent(spec.Components.Grafana,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
			WithRepoURL(spec.ComponentsMeta.Grafana.RepoURL),
			WithVersion(spec.ComponentsMeta.Grafana.Version),
		)
		if err != nil {
			return nil, err
		}
		components[Grafana] = grafana
	}
	if spec.IsNacosEnabled() {
		nacos, err := NewNacosComponent(spec.Components.Nacos,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
		)
		if err != nil {
			return nil, err
		}
		components[Nacos] = nacos
	}
	if spec.IsZookeeperEnabled() {
		zookeeper, err := NewZookeeperComponent(spec.Components.Zookeeper,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
			WithRepoURL(spec.ComponentsMeta.Zookeeper.RepoURL),
			WithVersion(spec.ComponentsMeta.Zookeeper.Version),
		)
		if err != nil {
			return nil, err
		}
		components[Zookeeper] = zookeeper
	}
	do := &DubboOperator{
		spec:       spec,
		components: components,
		namespace:  ns,
	}
	if !dryRun {
		ctlCli, err := kube.NewCtlClient(kubecfgPath, ctx)
		if err != nil {
			return nil, err
		}
		do.kubeCli = ctlCli
	}

	return do, nil
}