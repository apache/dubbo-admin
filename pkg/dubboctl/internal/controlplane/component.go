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
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/dubboctl/internal/manifest/render"
	"os"
	"sigs.k8s.io/yaml"
)

type ComponentName string

const (
	Admin     ComponentName = "admin"
	Grafana   ComponentName = "grafana"
	Nacos     ComponentName = "nacos"
	Zookeeper ComponentName = "zookeeper"
)

var (
	ComponentMap = map[string]ComponentName{
		"admin":     Admin,
		"grafana":   Grafana,
		"nacos":     Nacos,
		"zookeeper": Zookeeper,
	}
)

type Component interface {
	Run() error
	RenderManifest() (string, error)
}

type AdminComponent struct {
	spec     *v1alpha1.AdminSpec
	renderer render.Renderer
	started  bool
}

func (ac *AdminComponent) Run() error {
	if err := ac.renderer.Init(); err != nil {
		return err
	}
	ac.started = true
	return nil
}

func (ac *AdminComponent) RenderManifest() (string, error) {
	if !ac.started {
		return "", nil
	}
	// todo: considering operator action(CR change or whatever), we may introduce a valsYaml field to reduce Marshal cost
	valsYaml, err := yaml.Marshal(ac.spec)
	if err != nil {
		return "", err
	}
	manifest, err := ac.renderer.RenderManifest(string(valsYaml))
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewAdminComponent(spec *v1alpha1.AdminSpec, namespace string, chartPath string) (Component, error) {
	// todo: consider using chartPath to distinguish the type of renderer
	// using LocalRenderer by default
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Admin)),
		render.WithNameSpace(namespace),
		render.WithFS(os.DirFS(chartPath)),
		render.WithDir("dubbo-admin"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	admin := &AdminComponent{
		spec:     spec,
		renderer: renderer,
	}
	return admin, nil
}

type GrafanaComponent struct {
	spec     *v1alpha1.GrafanaSpec
	renderer render.Renderer
	started  bool
}

func (gc *GrafanaComponent) Run() error {
	if err := gc.renderer.Init(); err != nil {
		return err
	}
	gc.started = true
	return nil
}

func (gc *GrafanaComponent) RenderManifest() (string, error) {
	if !gc.started {
		return "", nil
	}
	valsYaml, err := yaml.Marshal(gc.spec)
	if err != nil {
		return "", err
	}
	manifest, err := gc.renderer.RenderManifest(string(valsYaml))
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewGrafanaComponent(spec *v1alpha1.GrafanaSpec, namespace string, chartPath string) (Component, error) {
	// todo: consider using chartPath to distinguish the type of renderer
	// using LocalRenderer by default
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Grafana)),
		render.WithNameSpace(namespace),
		render.WithFS(os.DirFS(chartPath)),
		render.WithDir("grafana"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	grafana := &GrafanaComponent{
		spec:     spec,
		renderer: renderer,
	}
	return grafana, nil
}

type NacosComponent struct {
	spec     *v1alpha1.NacosSpec
	renderer render.Renderer
	started  bool
}

func (nc *NacosComponent) Run() error {
	if err := nc.renderer.Init(); err != nil {
		return err
	}
	nc.started = true
	return nil
}

func (nc *NacosComponent) RenderManifest() (string, error) {
	if !nc.started {
		return "", nil
	}
	valsYaml, err := yaml.Marshal(nc.spec)
	if err != nil {
		return "", err
	}
	manifest, err := nc.renderer.RenderManifest(string(valsYaml))
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewNacosComponent(spec *v1alpha1.NacosSpec, namespace string, chartPath string) (Component, error) {
	// todo: consider using chartPath to distinguish the type of renderer
	// using LocalRenderer by default
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Nacos)),
		render.WithNameSpace(namespace),
		render.WithFS(os.DirFS(chartPath)),
		render.WithDir("nacos"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	nacos := &NacosComponent{
		spec:     spec,
		renderer: renderer,
	}
	return nacos, nil
}

type ZookeeperComponent struct {
	spec     *v1alpha1.ZookeeperSpec
	renderer render.Renderer
	started  bool
}

func (zc *ZookeeperComponent) Run() error {
	if err := zc.renderer.Init(); err != nil {
		return err
	}
	zc.started = true
	return nil
}

func (zc *ZookeeperComponent) RenderManifest() (string, error) {
	if !zc.started {
		return "", nil
	}
	valsYaml, err := yaml.Marshal(zc.spec)
	if err != nil {
		return "", err
	}
	manifest, err := zc.renderer.RenderManifest(string(valsYaml))
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewZookeeperComponent(spec *v1alpha1.ZookeeperSpec, namespace string, chartPath string) (Component, error) {
	// todo: consider using chartPath to distinguish the type of renderer
	// using LocalRenderer by default
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Zookeeper)),
		render.WithNameSpace(namespace),
		render.WithFS(os.DirFS(chartPath)),
		render.WithDir("zookeeper"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	zookeeper := &ZookeeperComponent{
		spec:     spec,
		renderer: renderer,
	}
	return zookeeper, nil
}
