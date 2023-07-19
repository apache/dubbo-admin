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
	"testing"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/app/dubboctl/internal/util"
)

const (
	wantPath = "./testdata/want"
)

type newComponentFunc func(t *testing.T) Component

func TestRenderManifest(t *testing.T) {
	tests := []struct {
		name          string
		golden        string
		componentFunc newComponentFunc
	}{
		{
			name:   "AdminComponent",
			golden: "admin_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.AdminSpec{}
				admin, err := NewAdminComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithChartPath(identifier.Charts),
				}...)
				if err != nil {
					t.Fatalf("NewAdminComponent failed, err: %s", err)
				}
				return admin
			},
		},
		{
			name:   "NacosComponent",
			golden: "nacos_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.NacosSpec{}
				nacos, err := NewNacosComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithChartPath(identifier.Charts),
				}...)
				if err != nil {
					t.Fatalf("NewNacosComponent failed, err: %s", err)
				}
				return nacos
			},
		},
		{
			name:   "ZookeeperComponent",
			golden: "zookeeper_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.ZookeeperSpec{}
				zookeeper, err := NewZookeeperComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithRepoURL("https://charts.bitnami.com/bitnami"),
					WithVersion("11.1.6"),
				}...)
				if err != nil {
					t.Fatalf("NewZookeeperComponent failed, err: %s", err)
				}
				return zookeeper
			},
		},
		{
			name:   "PrometheusComponent",
			golden: "prometheus_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.PrometheusSpec{}
				prometheus, err := NewPrometheusComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithRepoURL("https://prometheus-community.github.io/helm-charts"),
					WithVersion("20.0.2"),
				}...)
				if err != nil {
					t.Fatalf("NewPrometheusComponent failed, err: %s", err)
				}
				return prometheus
			},
		},
		{
			name:   "SkywalkingComponent",
			golden: "skywalking_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.SkywalkingSpec{}
				skywalking, err := NewSkywalkingComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithRepoURL("https://apache.jfrog.io/artifactory/skywalking-helm"),
					WithVersion("4.3.0"),
				}...)
				if err != nil {
					t.Fatalf("NewSkywalkingComponent failed, err: %s", err)
				}
				return skywalking
			},
		},
		{
			name:   "ZipkinComponent",
			golden: "zipkin_component-render_manifest.golden.yaml",
			componentFunc: func(t *testing.T) Component {
				testSpec := &v1alpha1.ZipkinSpec{}
				zipkin, err := NewZipkinComponent(testSpec, []ComponentOption{
					WithNamespace(identifier.DubboSystemNamespace),
					WithRepoURL("https://openzipkin.github.io/zipkin"),
					WithVersion("0.3.0"),
				}...)
				if err != nil {
					t.Fatalf("NewZipkinComponent failed, err: %s", err)
				}
				return zipkin
			},
		},
	}

	for _, test := range tests {
		tt := test
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			golden, err := readManifest(path.Join(wantPath, tt.golden))
			if err != nil {
				t.Fatalf("readManifest failed, err: %s", err)
			}
			comp := tt.componentFunc(t)
			if err := comp.Run(); err != nil {
				t.Fatalf("%s Run failed, err: %s", tt.name, err)
			}
			res, err := comp.RenderManifest()
			if err != nil {
				t.Errorf("%s RenderManifest failed, err: %s", tt.name, err)
				return
			}
			flag, diff, err := util.TestYAMLEqual(golden, res)
			if err != nil {
				t.Fatalf("IsTestYAMLEqual failed, err: %s", err)
			}
			if !flag {
				t.Errorf("diff:\n%s", diff)
				return
			}
		})
	}
}

func readManifest(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}
