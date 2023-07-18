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

	"github.com/apache/dubbo-admin/pkg/core/logger"

	"github.com/apache/dubbo-admin/app/dubboctl/internal/kube"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
)

type DubboOperator struct {
	spec       *v1alpha1.DubboConfigSpec
	started    bool
	components map[ComponentName]Component
	kubeCli    *kube.CtlClient
}

// Run must be invoked before invoking other functions.
func (do *DubboOperator) Run() error {
	for name, component := range do.components {
		if err := component.Run(); err != nil {
			return fmt.Errorf("bootstrap %s run failed, err: %s", name, err)
		}
	}
	do.started = true
	return nil
}

// RenderManifest renders bootstrap manifests specified by DubboConfigSpec.
func (do *DubboOperator) RenderManifest() (map[ComponentName]string, error) {
	if !do.started {
		return nil, errors.New("DubboOperator is not running")
	}
	res := make(map[ComponentName]string)
	for name, component := range do.components {
		manifest, err := component.RenderManifest()
		if err != nil {
			return nil, fmt.Errorf("bootstrap %s RenderManifest err: %v", name, err)
		}
		res[name] = manifest
	}
	return res, nil
}

// ApplyManifest apply bootstrap manifests to k8s cluster
func (do *DubboOperator) ApplyManifest(manifestMap map[ComponentName]string) error {
	if do.kubeCli == nil {
		return errors.New("no injected k8s cli into DubboOperator")
	}
	for name, manifest := range manifestMap {
		logger.CmdSugar().Infof("Start applying bootstrap %s\n", name)
		if err := do.kubeCli.ApplyManifest(manifest, do.spec.Namespace); err != nil {
			return fmt.Errorf("bootstrap %s ApplyManifest err: %v", name, err)
		}
		logger.CmdSugar().Infof("Applying bootstrap %s successfully\n", name)
	}
	return nil
}

// RemoveManifest removes resources represented by bootstrap manifests
func (do *DubboOperator) RemoveManifest(manifestMap map[ComponentName]string) error {
	if do.kubeCli == nil {
		return errors.New("no injected k8s cli into DubboOperator")
	}
	for name, manifest := range manifestMap {
		logger.CmdSugar().Infof("Start removing bootstrap %s\n", name)
		if err := do.kubeCli.RemoveManifest(manifest, do.spec.Namespace); err != nil {
			return fmt.Errorf("bootstrap %s RemoveManifest err: %v", name, err)
		}
		logger.CmdSugar().Infof("Removing bootstrap %s successfully\n", name)
	}
	return nil
}

// NewDubboOperator accepts cli directly for testing and normal use.
// For now, every related command needs a dedicated DubboOperator.
func NewDubboOperator(spec *v1alpha1.DubboConfigSpec, cli *kube.CtlClient) (*DubboOperator, error) {
	if spec == nil {
		return nil, errors.New("DubboConfigSpec is empty")
	}
	ns := spec.Namespace
	if ns == "" {
		ns = identifier.DubboSystemNamespace
	}
	// initialize components
	components := make(map[ComponentName]Component)
	if spec.IsAdminEnabled() {
		admin, err := NewAdminComponent(spec.Components.Admin,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
		)
		if err != nil {
			return nil, fmt.Errorf("NewAdminComponent failed, err: %s", err)
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
			return nil, fmt.Errorf("NewGrafanaComponent failed, err: %s", err)
		}
		components[Grafana] = grafana
	}
	if spec.IsNacosEnabled() {
		nacos, err := NewNacosComponent(spec.Components.Nacos,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
		)
		if err != nil {
			return nil, fmt.Errorf("NewNacosComponent failed, err: %s", err)
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
			return nil, fmt.Errorf("NewZookeeperComponent failed, err: %s", err)
		}
		components[Zookeeper] = zookeeper
	}
	if spec.IsPrometheusEnabled() {
		prometheus, err := NewPrometheusComponent(spec.Components.Prometheus,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
			WithRepoURL(spec.ComponentsMeta.Prometheus.RepoURL),
			WithVersion(spec.ComponentsMeta.Prometheus.Version),
		)
		if err != nil {
			return nil, fmt.Errorf("NewPrometheusComponent failed, err: %s", err)
		}
		components[Prometheus] = prometheus
	}
	if spec.IsSkywalkingEnabled() {
		skywalking, err := NewSkywalkingComponent(spec.Components.Skywalking,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
			WithRepoURL(spec.ComponentsMeta.Skywalking.RepoURL),
			WithVersion(spec.ComponentsMeta.Skywalking.Version),
		)
		if err != nil {
			return nil, fmt.Errorf("NewSkywalkingComponent failed, err: %s", err)
		}
		components[Skywalking] = skywalking
	}
	if spec.IsZipkinEnabled() {
		zipkin, err := NewZipkinComponent(spec.Components.Zipkin,
			WithNamespace(ns),
			WithChartPath(spec.ChartPath),
			WithRepoURL(spec.ComponentsMeta.Zipkin.RepoURL),
			WithVersion(spec.ComponentsMeta.Zipkin.Version),
		)
		if err != nil {
			return nil, fmt.Errorf("NewZipkinComponent failed, err: %s", err)
		}
		components[Zipkin] = zipkin
	}
	do := &DubboOperator{
		spec:       spec,
		components: components,
		kubeCli:    cli,
	}

	return do, nil
}
