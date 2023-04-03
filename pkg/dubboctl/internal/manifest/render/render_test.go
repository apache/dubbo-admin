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

package render

import (
	"os"
	"reflect"
	"testing"
)

const (
	TestDir       string = "testchart"
	TestName      string = "nginx"
	TestNamespace string = "dubbo-operator"
)

var (
	TestFS = os.DirFS(".")
)

func TestNewLocalRenderer(t *testing.T) {
	tests := []struct {
		desc      string
		opts      []RendererOption
		expectErr string
	}{
		{
			desc: "correct invocation",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
				WithFS(TestFS),
				WithDir(TestDir),
			},
			expectErr: "",
		},
		{
			desc:      "missing name",
			opts:      []RendererOption{},
			expectErr: "Render - NewLocalRenderer - verify err: missing component name for Renderer",
		},
		{
			desc: "missing chart FS",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
			},
			expectErr: "Render - NewLocalRenderer - verify err: missing chart FS for Renderer",
		},
		{
			desc: "missing chart dir",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
				WithFS(TestFS),
			},
			expectErr: "Render - NewLocalRenderer - verify err: missing chart dir for Renderer",
		},
		{
			desc: "using default namespace",
			opts: []RendererOption{
				WithName(TestName),
				WithFS(TestFS),
				WithDir(TestDir),
			},
			expectErr: "",
		},
	}

	for _, test := range tests {
		_, err := NewLocalRenderer(test.opts...)
		if err == nil {
			if test.expectErr != "" {
				t.Fatalf("expect err: %s", test.expectErr)
			}
		} else {
			if err.Error() != test.expectErr {
				t.Fatalf("expect err: %s\nbut got %s", test.expectErr, err)
			}
		}
	}
}

func TestLocalRenderer_RenderManifest(t *testing.T) {
	renderer, err := NewLocalRenderer(
		WithName("test"),
		WithNamespace("dubbo-system"),
		WithFS(os.DirFS(".")),
		WithDir("testchart"))
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		renderer Renderer
		vals     string
		expect   string
	}{
		{
			renderer: renderer,
			vals:     "",
			expect: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: testchart
      app.kubernetes.io/instance: dubbo
  template:
    metadata:
      labels:
        app.kubernetes.io/name: testchart
        app.kubernetes.io/instance: dubbo
    spec:
      serviceAccountName: dubbo-testchart
      securityContext:
        {}
      containers:
        - name: testchart
          securityContext:
            {}
          image: "nginx:1.16.0"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /
              port: http
          readinessProbe:
            httpGet:
              path: /
              port: http
          resources:
            {}

---


---


---
apiVersion: v1
kind: Service
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm

---
apiVersion: v1
kind: Pod
metadata:
  name: "dubbo-testchart-test-connection"
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['dubbo-testchart:80']
  restartPolicy: Never

---
`,
		},
		{
			renderer: renderer,
			vals: `
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
`,
			expect: `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: testchart
      app.kubernetes.io/instance: dubbo
  template:
    metadata:
      labels:
        app.kubernetes.io/name: testchart
        app.kubernetes.io/instance: dubbo
    spec:
      serviceAccountName: dubbo-testchart
      securityContext:
        {}
      containers:
        - name: testchart
          securityContext:
            {}
          image: "nginx:1.16.0"
          imagePullPolicy: IfNotPresent
          ports:
            - name: http
              containerPort: 80
              protocol: TCP
          livenessProbe:
            httpGet:
              path: /
              port: http
          readinessProbe:
            httpGet:
              path: /
              port: http
          resources:
            limits:
              cpu: 100m
              memory: 128Mi
            requests:
              cpu: 100m
              memory: 128Mi

---


---


---
apiVersion: v1
kind: Service
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo

---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: dubbo-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm

---
apiVersion: v1
kind: Pod
metadata:
  name: "dubbo-testchart-test-connection"
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: dubbo
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['dubbo-testchart:80']
  restartPolicy: Never

---
`,
		},
	}

	for _, test := range tests {
		if err := test.renderer.Init(); err != nil {
			t.Fatal(err)
		}
		manifest, err := test.renderer.RenderManifest(test.vals)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(manifest, test.expect) {
			t.Errorf("expect %s\nbut got %s\n", test.expect, manifest)
		}
	}
}

func TestRemoteRenderer_Init(t *testing.T) {
	rr, err := NewRemoteRenderer([]RendererOption{
		WithName("grafana"),
		WithNamespace(TestNamespace),
		WithRepoURL("https://grafana.github.io/helm-charts/"),
		WithVersion("6.31.1"),
	}...)
	if err != nil {
		t.Fatal(err)
	}
	if err := rr.Init(); err != nil {
		t.Fatal(err)
	}
}

func TestRemoteRenderer_RenderManifest(t *testing.T) {
	
}
