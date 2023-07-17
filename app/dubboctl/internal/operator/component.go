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
	"os"
	"path"
	"strings"
	"unicode/utf8"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/manifest/render"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"

	"sigs.k8s.io/yaml"
)

type ComponentName string

const (
	Admin      ComponentName = "admin"
	Grafana    ComponentName = "grafana"
	Nacos      ComponentName = "nacos"
	Zookeeper  ComponentName = "zookeeper"
	Prometheus ComponentName = "prometheus"
	Skywalking ComponentName = "skywalking"
	Zipkin     ComponentName = "zipkin"
)

var ComponentMap = map[string]ComponentName{
	"admin":      Admin,
	"grafana":    Grafana,
	"nacos":      Nacos,
	"zookeeper":  Zookeeper,
	"prometheus": Prometheus,
	"skywalking": Skywalking,
	"zipkin":     Zipkin,
}

// Component is used to represent dubbo control plane module, eg: zookeeper
type Component interface {
	Run() error
	RenderManifest() (string, error)
}

type ComponentOptions struct {
	Namespace string

	// local
	ChartPath string

	// remote
	RepoURL string
	Version string
}

type ComponentOption func(*ComponentOptions)

func WithNamespace(namespace string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.Namespace = namespace
	}
}

func WithChartPath(path string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.ChartPath = path
	}
}

func WithRepoURL(url string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.RepoURL = url
	}
}

func WithVersion(version string) ComponentOption {
	return func(opts *ComponentOptions) {
		opts.Version = version
	}
}

type AdminComponent struct {
	spec     *v1alpha1.AdminSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
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
	manifest, err := renderManifest(ac.spec, ac.renderer, false, Admin, ac.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewAdminComponent(spec *v1alpha1.AdminSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// verify newOpts
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Admin)),
		render.WithNamespace(newOpts.Namespace),
		render.WithFS(os.DirFS(newOpts.ChartPath)),
		render.WithDir("dubbo-admin"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	admin := &AdminComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return admin, nil
}

type GrafanaComponent struct {
	spec     *v1alpha1.GrafanaSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
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
	manifest, err := renderManifest(gc.spec, gc.renderer, true, Grafana, gc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewGrafanaComponent(spec *v1alpha1.GrafanaSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// todo: verify newOpts
	var renderer render.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = render.NewRemoteRenderer(
			render.WithName(string(Grafana)),
			render.WithNamespace(newOpts.Namespace),
			render.WithRepoURL(newOpts.RepoURL),
			render.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = render.NewLocalRenderer(
			render.WithName(string(Grafana)),
			render.WithNamespace(newOpts.Namespace),
			render.WithFS(os.DirFS(newOpts.ChartPath)),
			render.WithDir("grafana"),
		)
		if err != nil {
			return nil, err
		}
	}

	// todo: verify spec
	grafana := &GrafanaComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return grafana, nil
}

type NacosComponent struct {
	spec     *v1alpha1.NacosSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
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
	manifest, err := renderManifest(nc.spec, nc.renderer, false, Nacos, nc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewNacosComponent(spec *v1alpha1.NacosSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// verify newOpts
	renderer, err := render.NewLocalRenderer(
		render.WithName(string(Nacos)),
		render.WithNamespace(newOpts.Namespace),
		render.WithFS(os.DirFS(newOpts.ChartPath)),
		render.WithDir("nacos"))
	if err != nil {
		return nil, err
	}
	// todo: verify spec
	nacos := &NacosComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return nacos, nil
}

type ZookeeperComponent struct {
	spec     *v1alpha1.ZookeeperSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
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
	manifest, err := renderManifest(zc.spec, zc.renderer, true, Zookeeper, zc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewZookeeperComponent(spec *v1alpha1.ZookeeperSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// verify newOpts
	var renderer render.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = render.NewRemoteRenderer(
			render.WithName(string(Zookeeper)),
			render.WithNamespace(newOpts.Namespace),
			render.WithRepoURL(newOpts.RepoURL),
			render.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = render.NewLocalRenderer(
			render.WithName(string(Zookeeper)),
			render.WithNamespace(newOpts.Namespace),
			render.WithFS(os.DirFS(newOpts.ChartPath)),
			render.WithDir("zookeeper"),
		)
		if err != nil {
			return nil, err
		}
	}

	// todo: verify spec
	zookeeper := &ZookeeperComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return zookeeper, nil
}

type PrometheusComponent struct {
	spec     *v1alpha1.PrometheusSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
}

func (pc *PrometheusComponent) Run() error {
	if err := pc.renderer.Init(); err != nil {
		return err
	}
	pc.started = true
	return nil
}

func (pc *PrometheusComponent) RenderManifest() (string, error) {
	if !pc.started {
		return "", nil
	}
	manifest, err := renderManifest(pc.spec, pc.renderer, true, Prometheus, pc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewPrometheusComponent(spec *v1alpha1.PrometheusSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// todo: verify newOpts
	var renderer render.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = render.NewRemoteRenderer(
			render.WithName(string(Prometheus)),
			render.WithNamespace(newOpts.Namespace),
			render.WithRepoURL(newOpts.RepoURL),
			render.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = render.NewLocalRenderer(
			render.WithName(string(Prometheus)),
			render.WithNamespace(newOpts.Namespace),
			render.WithFS(os.DirFS(newOpts.ChartPath)),
			render.WithDir("prometheus"),
		)
		if err != nil {
			return nil, err
		}
	}

	// todo: verify spec
	prometheus := &PrometheusComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return prometheus, nil
}

type SkywalkingComponent struct {
	spec     *v1alpha1.SkywalkingSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
}

func (sc *SkywalkingComponent) Run() error {
	if err := sc.renderer.Init(); err != nil {
		return err
	}
	sc.started = true
	return nil
}

func (sc *SkywalkingComponent) RenderManifest() (string, error) {
	if !sc.started {
		return "", nil
	}
	manifest, err := renderManifest(sc.spec, sc.renderer, true, Skywalking, sc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewSkywalkingComponent(spec *v1alpha1.SkywalkingSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// todo: verify newOpts
	var renderer render.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = render.NewRemoteRenderer(
			render.WithName(string(Skywalking)),
			render.WithNamespace(newOpts.Namespace),
			render.WithRepoURL(newOpts.RepoURL),
			render.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = render.NewLocalRenderer(
			render.WithName(string(Skywalking)),
			render.WithNamespace(newOpts.Namespace),
			render.WithFS(os.DirFS(newOpts.ChartPath)),
			render.WithDir("skywalking"),
		)
		if err != nil {
			return nil, err
		}
	}

	// todo: verify spec
	skywalking := &SkywalkingComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return skywalking, nil
}

type ZipkinComponent struct {
	spec     *v1alpha1.ZipkinSpec
	renderer render.Renderer
	started  bool
	opts     *ComponentOptions
}

func (zc *ZipkinComponent) Run() error {
	if err := zc.renderer.Init(); err != nil {
		return err
	}
	zc.started = true
	return nil
}

func (zc *ZipkinComponent) RenderManifest() (string, error) {
	if !zc.started {
		return "", nil
	}
	manifest, err := renderManifest(zc.spec, zc.renderer, true, Zipkin, zc.opts.Namespace)
	if err != nil {
		return "", err
	}
	return manifest, nil
}

func NewZipkinComponent(spec *v1alpha1.ZipkinSpec, opts ...ComponentOption) (Component, error) {
	newOpts := &ComponentOptions{}
	for _, opt := range opts {
		opt(newOpts)
	}
	// todo: verify newOpts
	var renderer render.Renderer
	var err error
	if newOpts.RepoURL != "" {
		renderer, err = render.NewRemoteRenderer(
			render.WithName(string(Zipkin)),
			render.WithNamespace(newOpts.Namespace),
			render.WithRepoURL(newOpts.RepoURL),
			render.WithVersion(newOpts.Version),
		)
		if err != nil {
			return nil, err
		}
	} else {
		renderer, err = render.NewLocalRenderer(
			render.WithName(string(Zipkin)),
			render.WithNamespace(newOpts.Namespace),
			render.WithFS(os.DirFS(newOpts.ChartPath)),
			render.WithDir("zipkin"),
		)
		if err != nil {
			return nil, err
		}
	}

	// todo: verify spec
	zipkin := &ZipkinComponent{
		spec:     spec,
		renderer: renderer,
		opts:     newOpts,
	}
	return zipkin, nil
}

func renderManifest(spec any, renderer render.Renderer, addOn bool, name ComponentName, namespace string) (string, error) {
	var valsBytes []byte
	var valsYaml string
	var err error
	if addOn {
		// see /deploy/addons
		// values-*.yaml is the base yaml for addon bootstrap
		valsYaml, err = manifest.ReadAndOverlayYamls([]string{
			path.Join(identifier.Addons, "values-"+string(name)+".yaml"),
		})
		if err != nil {
			return "", err
		}
	}

	// do not use spec != nil cause spec's type is not nil
	if !util.IsValueNil(spec) {
		valsBytes, err = yaml.Marshal(spec)
		if err != nil {
			return "", err
		}
		valsYaml, err = util.OverlayYAML(valsYaml, string(valsBytes))
		if err != nil {
			return "", err
		}
	}
	final, err := renderer.RenderManifest(valsYaml)
	if err != nil {
		return "", err
	}
	// grafana needs to add dashboard json file as configmap
	if name == Grafana {
		final, err = addDashboards(final, namespace)
		if err != nil {
			return "", err
		}
	}
	// manifest rendered by some charts may lack namespace, we set it there
	final, err = setNamespace(final, namespace)
	if err != nil {
		return "", err
	}
	return final, nil
}

// Hack function of grafana bootstrap. It needs external dashboard json file.
// We would consider design a more robust way to add dashboards files or merge these files to grafana chart.
// Assume that base ends with yaml separator.
func addDashboards(base string, namespace string) (string, error) {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "admin-extra-dashboards",
			Namespace: namespace,
		},
	}
	configMap.Data = make(map[string]string)
	dashboardName := "external-dashboard.json"
	dashboardPath := path.Join(identifier.AddonDashboards, dashboardName)
	content, err := os.ReadFile(dashboardPath)
	if err != nil {
		return "", err
	}
	// values with non-UTF-8 byte sequences must use the BinaryData field.
	if utf8.Valid(content) {
		configMap.Data[dashboardName] = string(content)
	} else {
		configMap.BinaryData[dashboardName] = content
	}
	yamlBytes, err := yaml.Marshal(configMap)
	if err != nil {
		return "", err
	}
	yamlStr := strings.TrimSpace(string(yamlBytes))
	final := base + yamlStr + render.YAMLSeparator
	return final, nil
}

// setNamespace split base and set namespace.
func setNamespace(base string, namespace string) (string, error) {
	var newSegs []string
	segs, err := util.SplitYAML(base)
	if err != nil {
		return "", err
	}
	for _, seg := range segs {
		segMap := make(map[string]interface{})
		if err := yaml.Unmarshal([]byte(seg), &segMap); err != nil {
			return "", err
		}
		pathCtx, _, err := manifest.GetPathContext(segMap, util.PathFromString("metadata.namespace"), true)
		if err != nil {
			return "", err
		}
		if err := manifest.WritePathContext(pathCtx, manifest.ParseValue(namespace), false); err != nil {
			return "", err
		}
		newSeg, err := yaml.Marshal(segMap)
		if err != nil {
			return "", err
		}
		newSegs = append(newSegs, string(newSeg))
	}
	final := util.JoinYAML(newSegs)
	return final, nil
}
