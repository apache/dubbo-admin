package render

import (
	"os"
	"reflect"
	"testing"
)

const (
	TestChart string = "./testchart"
)

func TestRenderManifest(t *testing.T) {
	renderer, err := NewLocalRenderer(
		WithNameSpace("dubbo-system"),
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
