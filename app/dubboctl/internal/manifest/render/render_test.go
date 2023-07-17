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
	"testing"

	"github.com/apache/dubbo-admin/app/dubboctl/identifier"
)

const (
	TestDir       string = "testchart"
	TestName      string = "nginx"
	TestNamespace string = "dubbo-operator"
)

var TestFS = os.DirFS(".")

func TestNewLocalRenderer(t *testing.T) {
	tests := []struct {
		desc    string
		opts    []RendererOption
		wantErr bool
	}{
		{
			desc: "correct invocation",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
				WithFS(TestFS),
				WithDir(TestDir),
			},
		},
		{
			desc:    "missing name",
			opts:    []RendererOption{},
			wantErr: true,
		},
		{
			desc: "missing chart FS",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
			},
			wantErr: true,
		},
		{
			desc: "missing chart dir",
			opts: []RendererOption{
				WithName(TestName),
				WithNamespace(TestNamespace),
				WithFS(TestFS),
			},
			wantErr: true,
		},
		{
			desc: "using default namespace",
			opts: []RendererOption{
				WithName(TestName),
				WithFS(TestFS),
				WithDir(TestDir),
			},
		},
	}

	for _, test := range tests {
		_, err := NewLocalRenderer(test.opts...)
		if err != nil {
			if !test.wantErr {
				t.Errorf("NewLocalRenderer failed, err: %s", err)
				return
			}
		}
	}
}

func TestLocalRenderer_RenderManifest(t *testing.T) {
	renderer, err := NewLocalRenderer(
		WithName(TestName),
		WithNamespace(identifier.DubboSystemNamespace),
		WithFS(TestFS),
		WithDir(TestDir))
	if err != nil {
		t.Fatalf("NewLocalRenderer failed, err: %s", err)
	}
	if err := renderer.Init(); err != nil {
		t.Fatalf("LocalRenderer Init failed, err: %s", err)
	}
	tests := []struct {
		vals   string
		expect string
	}{
		{
			vals: "",
			expect: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: testchart
      app.kubernetes.io/instance: nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: testchart
        app.kubernetes.io/instance: nginx
    spec:
      serviceAccountName: nginx-testchart
      securityContext: {}
      containers:
        - name: testchart
          securityContext: {}
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
          resources: {}
---
apiVersion: v1
kind: Service
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
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
    app.kubernetes.io/instance: nginx
---
kind: ServiceAccount
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
---
apiVersion: v1
kind: Pod
metadata:
  name: "nginx-testchart-test-storage"
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['nginx-testchart:80']
  restartPolicy: Never
---
`,
		},
		{
			vals: `
resources:
  limits:
    cpu: 100m
    memory: 128Mi
  requests:
    cpu: 100m
    memory: 128Mi
`,
			expect: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: testchart
      app.kubernetes.io/instance: nginx
  template:
    metadata:
      labels:
        app.kubernetes.io/name: testchart
        app.kubernetes.io/instance: nginx
    spec:
      serviceAccountName: nginx-testchart
      securityContext: {}
      containers:
        - name: testchart
          securityContext: {}
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
apiVersion: v1
kind: Service
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
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
    app.kubernetes.io/instance: nginx
---
kind: ServiceAccount
metadata:
  name: nginx-testchart
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
---
apiVersion: v1
kind: Pod
metadata:
  name: "nginx-testchart-test-storage"
  labels:
    helm.sh/chart: testchart-0.1.0
    app.kubernetes.io/name: testchart
    app.kubernetes.io/instance: nginx
    app.kubernetes.io/version: "1.16.0"
    app.kubernetes.io/managed-by: Helm
  annotations:
    "helm.sh/hook": test
spec:
  containers:
    - name: wget
      image: busybox
      command: ['wget']
      args: ['nginx-testchart:80']
  restartPolicy: Never
---
`,
		},
	}

	for _, test := range tests {
		manifest, err := renderer.RenderManifest(test.vals)
		if err != nil {
			t.Fatalf("LocalRenderer RenderManifest failed, err: %s", err)
		}
		if manifest != test.expect {
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
		t.Fatalf("NewRemoteRenderer failed, err: %s", err)
	}
	if err := rr.Init(); err != nil {
		t.Fatalf("RemoteRenderer failed, err: %s", err)
	}
}

//func TestRemoteRenderer_RenderManifest(t *testing.T) {
//	renderer, err := NewRemoteRenderer([]RendererOption{
//		WithName("grafana"),
//		WithNamespace(TestNamespace),
//		WithRepoURL("https://grafana.github.io/helm-charts/"),
//		WithVersion("6.31.1"),
//	}...)
//	if err != nil {
//		t.Fatalf("NewRemoteRenderer failed, err: %s", err)
//	}
//	if err := renderer.Init(); err != nil {
//		t.Fatalf("RemoteRenderer failed, err: %s", err)
//	}
//	tests := []struct {
//		vals   string
//		expect string
//	}{
//		{
//			vals: "",
//			expect: `kind: ClusterRole
//apiVersion: rbac.authorization.k8s.io/v1
//metadata:
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//  name: grafana-clusterrole
//rules: []
//---
//kind: ClusterRoleBinding
//apiVersion: rbac.authorization.k8s.io/v1
//metadata:
//  name: grafana-clusterrolebinding
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//subjects:
//  - kind: ServiceAccount
//    name: grafana
//    namespace: dubbo-operator
//roleRef:
//  kind: ClusterRole
//  name: grafana-clusterrole
//  apiGroup: rbac.authorization.k8s.io
//---
//apiVersion: v1
//kind: ConfigMap
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//data:
//  allow-snippet-annotations: "false"
//  grafana.ini: |
//    [analytics]
//    check_for_updates = true
//    [grafana_net]
//    url = https://grafana.net
//    [log]
//    mode = console
//    [paths]
//    data = /var/lib/grafana/
//    logs = /var/log/grafana
//    plugins = /var/lib/grafana/plugins
//    provisioning = /etc/grafana/provisioning
//---
//apiVersion: apps/v1
//kind: Deployment
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//spec:
//  replicas: 1
//  revisionHistoryLimit: 10
//  selector:
//    matchLabels:
//      app.kubernetes.io/name: grafana
//      app.kubernetes.io/instance: grafana
//  strategy:
//    type: RollingUpdate
//  template:
//    metadata:
//      labels:
//        app.kubernetes.io/name: grafana
//        app.kubernetes.io/instance: grafana
//      annotations:
//        checksum/config: 437246927f77c7545ee3969448fa4813d7455f10583d7b2d93e89ed06d8c2259
//        checksum/dashboards-json-config: 01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b
//        checksum/sc-dashboard-provider-config: 01ba4719c80b6fe911b091a7c05124b64eeece964e09c058ef8f9805daca546b
//        checksum/secret: 974af8ad1507a390f6d207b529770645659e266a9aabafe285520782f9a92277
//    spec:
//
//      serviceAccountName: grafana
//      automountServiceAccountToken: true
//      securityContext:
//        fsGroup: 472
//        runAsGroup: 472
//        runAsUser: 472
//      enableServiceLinks: true
//      containers:
//        - name: grafana
//          image: "grafana/grafana:9.0.1"
//          imagePullPolicy: IfNotPresent
//          volumeMounts:
//            - name: config
//              mountPath: "/etc/grafana/grafana.ini"
//              subPath: grafana.ini
//            - name: storage
//              mountPath: "/var/lib/grafana"
//          ports:
//            - name: service
//              containerPort: 80
//              protocol: TCP
//            - name: grafana
//              containerPort: 3000
//              protocol: TCP
//          env:
//            - name: GF_SECURITY_ADMIN_USER
//              valueFrom:
//                secretKeyRef:
//                  name: grafana
//                  key: admin-user
//            - name: GF_SECURITY_ADMIN_PASSWORD
//              valueFrom:
//                secretKeyRef:
//                  name: grafana
//                  key: admin-password
//            - name: GF_PATHS_DATA
//              value: /var/lib/grafana/
//            - name: GF_PATHS_LOGS
//              value: /var/log/grafana
//            - name: GF_PATHS_PLUGINS
//              value: /var/lib/grafana/plugins
//            - name: GF_PATHS_PROVISIONING
//              value: /etc/grafana/provisioning
//          livenessProbe:
//            failureThreshold: 10
//            httpGet:
//              path: /api/health
//              port: 3000
//            initialDelaySeconds: 60
//            timeoutSeconds: 30
//          readinessProbe:
//            httpGet:
//              path: /api/health
//              port: 3000
//          resources:
//            {}
//      volumes:
//        - name: config
//          configMap:
//            name: grafana
//        - name: storage
//          emptyDir: {}
//---
//apiVersion: policy/v1beta1
//kind: PodSecurityPolicy
//metadata:
//  name: grafana
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//  annotations:
//    seccomp.security.alpha.kubernetes.io/allowedProfileNames: 'docker/default,runtime/default'
//    seccomp.security.alpha.kubernetes.io/defaultProfileName:  'docker/default'
//    apparmor.security.beta.kubernetes.io/allowedProfileNames: 'runtime/default'
//    apparmor.security.beta.kubernetes.io/defaultProfileName:  'runtime/default'
//spec:
//  privileged: false
//  allowPrivilegeEscalation: false
//  requiredDropCapabilities:
//    # Default set from Docker, with DAC_OVERRIDE and CHOWN
//      - ALL
//  volumes:
//    - 'configMap'
//    - 'emptyDir'
//    - 'projected'
//    - 'csi'
//    - 'secret'
//    - 'downwardAPI'
//    - 'persistentVolumeClaim'
//  hostNetwork: false
//  hostIPC: false
//  hostPID: false
//  runAsUser:
//    rule: 'RunAsAny'
//  seLinux:
//    rule: 'RunAsAny'
//  supplementalGroups:
//    rule: 'MustRunAs'
//    ranges:
//      # Forbid adding the root group.
//      - min: 1
//        max: 65535
//  fsGroup:
//    rule: 'MustRunAs'
//    ranges:
//      # Forbid adding the root group.
//      - min: 1
//        max: 65535
//  readOnlyRootFilesystem: false
//---
//apiVersion: rbac.authorization.k8s.io/v1
//kind: Role
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//rules:
//- apiGroups:      ['extensions']
//  resources:      ['podsecuritypolicies']
//  verbs:          ['use']
//  resourceNames:  [grafana]
//---
//apiVersion: rbac.authorization.k8s.io/v1
//kind: RoleBinding
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//roleRef:
//  apiGroup: rbac.authorization.k8s.io
//  kind: Role
//  name: grafana
//subjects:
//- kind: ServiceAccount
//  name: grafana
//  namespace: dubbo-operator
//---
//apiVersion: v1
//kind: Secret
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//type: Opaque
//data:
//  admin-user: "YWRtaW4="
//  admin-password: "UTRGSmJCeGJIYmk4RzJLdW1ZSjNGRkE1bWdnVmVnZDQzdHJteEFWSQ=="
//  ldap-toml: ""
//---
//apiVersion: v1
//kind: Service
//metadata:
//  name: grafana
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//spec:
//  type: ClusterIP
//  ports:
//    - name: service
//      port: 80
//      protocol: TCP
//      targetPort: 3000
//  selector:
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//---
//apiVersion: v1
//kind: ServiceAccount
//metadata:
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//  name: grafana
//  namespace: dubbo-operator
//---
//apiVersion: v1
//kind: ConfigMap
//metadata:
//  name: grafana-test
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//data:
//  run.sh: |-
//    @test "Test Health" {
//      url="http://grafana/api/health"
//      code=$(wget --cp-server-response --spider --timeout 10 --tries 1 ${url} 2>&1 | awk '/^  HTTP/{print $2}')
//      [ "$code" == "200" ]
//    }
//---
//apiVersion: policy/v1beta1
//kind: PodSecurityPolicy
//metadata:
//  name: grafana-test
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//spec:
//  allowPrivilegeEscalation: true
//  privileged: false
//  hostNetwork: false
//  hostIPC: false
//  hostPID: false
//  fsGroup:
//    rule: RunAsAny
//  seLinux:
//    rule: RunAsAny
//  supplementalGroups:
//    rule: RunAsAny
//  runAsUser:
//    rule: RunAsAny
//  volumes:
//  - configMap
//  - downwardAPI
//  - emptyDir
//  - projected
//  - csi
//  - secret
//---
//apiVersion: rbac.authorization.k8s.io/v1
//kind: Role
//metadata:
//  name: grafana-test
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//rules:
//- apiGroups:      ['policy']
//  resources:      ['podsecuritypolicies']
//  verbs:          ['use']
//  resourceNames:  [grafana-test]
//---
//apiVersion: rbac.authorization.k8s.io/v1
//kind: RoleBinding
//metadata:
//  name: grafana-test
//  namespace: dubbo-operator
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//roleRef:
//  apiGroup: rbac.authorization.k8s.io
//  kind: Role
//  name: grafana-test
//subjects:
//- kind: ServiceAccount
//  name: grafana-test
//  namespace: dubbo-operator
//---
//apiVersion: v1
//kind: ServiceAccount
//metadata:
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//  name: grafana-test
//  namespace: dubbo-operator
//---
//apiVersion: v1
//kind: Pod
//metadata:
//  name: grafana-test
//  labels:
//    helm.sh/chart: grafana-6.31.1
//    app.kubernetes.io/name: grafana
//    app.kubernetes.io/instance: grafana
//    app.kubernetes.io/version: "9.0.1"
//    app.kubernetes.io/managed-by: Helm
//  annotations:
//    "helm.sh/hook": test-success
//    "helm.sh/hook-delete-policy": "before-hook-creation,hook-succeeded"
//  namespace: dubbo-operator
//spec:
//  serviceAccountName: grafana-test
//  containers:
//    - name: grafana-test
//      image: "bats/bats:v1.4.1"
//      imagePullPolicy: "IfNotPresent"
//      command: ["/opt/bats/bin/bats", "-t", "/tests/run.sh"]
//      volumeMounts:
//        - mountPath: /tests
//          name: tests
//          readOnly: true
//  volumes:
//  - name: tests
//    configMap:
//      name: grafana-test
//  restartPolicy: Never
//---
//`,
//		},
//		{
//			vals: `
//testFramework:
//  enabled: false
//`,
//			expect: `apiVersion: apps/v1
//kind: Deployment
//metadata:
//  name: nginx-testchart
//  labels:
//    helm.sh/chart: testchart-0.1.0
//    app.kubernetes.io/name: testchart
//    app.kubernetes.io/instance: nginx
//    app.kubernetes.io/version: "1.16.0"
//    app.kubernetes.io/managed-by: Helm
//spec:
//  replicas: 1
//  selector:
//    matchLabels:
//      app.kubernetes.io/name: testchart
//      app.kubernetes.io/instance: nginx
//  template:
//    metadata:
//      labels:
//        app.kubernetes.io/name: testchart
//        app.kubernetes.io/instance: nginx
//    spec:
//      serviceAccountName: nginx-testchart
//      securityContext:
//        {}
//      containers:
//        - name: testchart
//          securityContext:
//            {}
//          image: "nginx:1.16.0"
//          imagePullPolicy: IfNotPresent
//          ports:
//            - name: http
//              containerPort: 80
//              protocol: TCP
//          livenessProbe:
//            httpGet:
//              path: /
//              port: http
//          readinessProbe:
//            httpGet:
//              path: /
//              port: http
//          resources:
//            limits:
//              cpu: 100m
//              memory: 128Mi
//            requests:
//              cpu: 100m
//              memory: 128Mi
//---
//apiVersion: v1
//kind: Service
//metadata:
//  name: nginx-testchart
//  labels:
//    helm.sh/chart: testchart-0.1.0
//    app.kubernetes.io/name: testchart
//    app.kubernetes.io/instance: nginx
//    app.kubernetes.io/version: "1.16.0"
//    app.kubernetes.io/managed-by: Helm
//spec:
//  type: ClusterIP
//  ports:
//    - port: 80
//      targetPort: http
//      protocol: TCP
//      name: http
//  selector:
//    app.kubernetes.io/name: testchart
//    app.kubernetes.io/instance: nginx
//---
//kind: ServiceAccount
//metadata:
//  name: nginx-testchart
//  labels:
//    helm.sh/chart: testchart-0.1.0
//    app.kubernetes.io/name: testchart
//    app.kubernetes.io/instance: nginx
//    app.kubernetes.io/version: "1.16.0"
//    app.kubernetes.io/managed-by: Helm
//---
//apiVersion: v1
//kind: Pod
//metadata:
//  name: "nginx-testchart-test-storage"
//  labels:
//    helm.sh/chart: testchart-0.1.0
//    app.kubernetes.io/name: testchart
//    app.kubernetes.io/instance: nginx
//    app.kubernetes.io/version: "1.16.0"
//    app.kubernetes.io/managed-by: Helm
//  annotations:
//    "helm.sh/hook": test
//spec:
//  containers:
//    - name: wget
//      image: busybox
//      command: ['wget']
//      args: ['nginx-testchart:80']
//  restartPolicy: Never
//---
//`,
//		},
//	}
//
//	for _, test := range tests {
//		manifest, err := renderer.RenderManifest(test.vals)
//		if err != nil {
//			t.Fatalf("RemoteRenderer RenderManifest failed, err: %s", err)
//		}
//		if manifest != test.expect {
//			t.Errorf("expect %s\nbut got %s\n", test.expect, manifest)
//			return
//		}
//	}
//}
