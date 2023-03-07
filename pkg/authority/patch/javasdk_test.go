// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package patch

import (
	"reflect"
	"testing"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	v1 "k8s.io/api/core/v1"
)

type fakeKubeClient struct {
	k8s.Client
}

func (f *fakeKubeClient) GetNamespaceLabels(namespace string) map[string]string {
	if namespace == "matched" {
		return map[string]string{
			"dubbo-ca.inject": "true",
		}
	} else {
		return map[string]string{}
	}
}

func TestEmpty(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestInjectFromLabel(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Labels = make(map[string]string)
	pod.Labels["dubbo-ca.inject"] = "true"

	newPod, _ := sdk.NewPod(pod)

	if reflect.DeepEqual(newPod, pod) {
		t.Error("should not be equal")
	}
}

func TestInjectFromNs(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	newPod, _ := sdk.NewPod(pod)

	if reflect.DeepEqual(newPod, pod) {
		t.Error("should not be equal")
	}
}

func TestInjectVolumes(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	newPod, _ := sdk.NewPod(pod)

	if reflect.DeepEqual(newPod, pod) {
		t.Error("should not be equal")
	}

	if len(newPod.Spec.Volumes) != 2 {
		t.Error("should have 1 volume")
	}

	if newPod.Spec.Volumes[0].Name != "dubbo-ca-token" {
		t.Error("should have dubbo-ca-token volume")
	}

	if len(newPod.Spec.Volumes[0].Projected.Sources) != 1 {
		t.Error("should have 1 projected source")
	}

	if newPod.Spec.Volumes[0].Projected.Sources[0].ServiceAccountToken.Path != "token" {
		t.Error("should have token path")
	}

	if newPod.Spec.Volumes[0].Projected.Sources[0].ServiceAccountToken.Audience != "dubbo-ca" {
		t.Error("should have dubbo-ca audience")
	}

	if *newPod.Spec.Volumes[0].Projected.Sources[0].ServiceAccountToken.ExpirationSeconds != 1800 {
		t.Error("should have 1800 expiration seconds")
	}

	if newPod.Spec.Volumes[1].Name != "dubbo-ca-cert" {
		t.Error("should have dubbo-ca-cert volume")
	}

	if len(newPod.Spec.Volumes[1].Projected.Sources) != 1 {
		t.Error("should have 1 projected source")
	}

	if newPod.Spec.Volumes[1].Projected.Sources[0].ConfigMap.Name != "dubbo-ca-cert" {
		t.Error("should have dubbo-ca-cert configmap")
	}

	if len(newPod.Spec.Volumes[1].Projected.Sources[0].ConfigMap.Items) != 1 {
		t.Error("should have 1 item")
	}

	if newPod.Spec.Volumes[1].Projected.Sources[0].ConfigMap.Items[0].Key != "ca.crt" {
		t.Error("should have ca.crt key")
	}

	if newPod.Spec.Volumes[1].Projected.Sources[0].ConfigMap.Items[0].Path != "ca.crt" {
		t.Error("should have ca.crt path")
	}
}

func TestInjectOneContainer(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	newPod, _ := sdk.NewPod(pod)

	if reflect.DeepEqual(newPod, pod) {
		t.Error("should not be equal")
	}

	if len(newPod.Spec.Containers) != 1 {
		t.Error("should have 1 container")
	}

	container := newPod.Spec.Containers[0]
	checkContainer(t, container)
}

func TestInjectTwoContainer(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 2)
	pod.Spec.Containers[0].Name = "test"
	pod.Spec.Containers[1].Name = "test"

	newPod, _ := sdk.NewPod(pod)

	if reflect.DeepEqual(newPod, pod) {
		t.Error("should not be equal")
	}

	if len(newPod.Spec.Containers) != 2 {
		t.Error("should have 2 container")
	}

	container := newPod.Spec.Containers[0]
	checkContainer(t, container)

	container = newPod.Spec.Containers[1]
	checkContainer(t, container)
}

func checkContainer(t *testing.T, container v1.Container) {
	if container.Name != "test" {
		t.Error("should have test container")
	}

	if len(container.Env) != 3 {
		t.Error("should have 3 env")
	}

	if container.Env[0].Name != "DUBBO_CA_ADDRESS" {
		t.Error("should have DUBBO_CA_ADDRESS env")
	}

	if container.Env[0].Value != "dubbo-ca.dubbo-system.svc:30062" {
		t.Error("should have dubbo-ca.dubbo-system.svc:30062 value")
	}

	if container.Env[1].Name != "DUBBO_CA_CERT_PATH" {
		t.Error("should have DUBBO_CA_TOKEN_PATH env")
	}

	if container.Env[1].Value != "/var/run/secrets/dubbo-ca-cert/ca.crt" {
		t.Error("should have /var/run/secrets/dubbo-ca-cert/ca.crt value")
	}

	if container.Env[2].Name != "DUBBO_OIDC_TOKEN" {
		t.Error("should have DUBBO_OIDC_TOKEN env")
	}

	if container.Env[2].Value != "/var/run/secrets/dubbo-ca-token/token" {
		t.Error("should have /var/run/secrets/dubbo-ca-token/token value")
	}

	if len(container.VolumeMounts) != 2 {
		t.Error("should have 2 volume mounts")
	}

	if container.VolumeMounts[0].Name != "dubbo-ca-token" {
		t.Error("should have dubbo-ca-token volume mount")
	}

	if container.VolumeMounts[0].MountPath != "/var/run/secrets/dubbo-ca-token" {
		t.Error("should have /var/run/secrets/dubbo-ca-token mount path")
	}

	if container.VolumeMounts[1].Name != "dubbo-ca-cert" {
		t.Error("should have dubbo-ca-cert volume mount")
	}

	if container.VolumeMounts[1].MountPath != "/var/run/secrets/dubbo-ca-cert" {
		t.Error("should have /var/run/secrets/dubbo-ca-cert mount path")
	}
}

func TestCheckVolume1(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Volumes = make([]v1.Volume, 1)
	pod.Spec.Volumes[0].Name = "dubbo-ca-token"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckVolume2(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Volumes = make([]v1.Volume, 1)
	pod.Spec.Volumes[0].Name = "dubbo-ca-cert"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckEnv1(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Containers[0].Env = make([]v1.EnvVar, 1)
	pod.Spec.Containers[0].Env[0].Name = "DUBBO_CA_ADDRESS"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckEnv2(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Containers[0].Env = make([]v1.EnvVar, 1)
	pod.Spec.Containers[0].Env[0].Name = "DUBBO_CA_CERT_PATH"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckEnv3(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Containers[0].Env = make([]v1.EnvVar, 1)
	pod.Spec.Containers[0].Env[0].Name = "DUBBO_OIDC_TOKEN"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckEnv4(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 2)
	pod.Spec.Containers[0].Name = "test"
	pod.Spec.Containers[1].Name = "test"

	pod.Spec.Containers[1].Env = make([]v1.EnvVar, 1)
	pod.Spec.Containers[1].Env[0].Name = "DUBBO_OIDC_TOKEN"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckContainerVolume1(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Containers[0].VolumeMounts = make([]v1.VolumeMount, 1)
	pod.Spec.Containers[0].VolumeMounts[0].Name = "dubbo-ca-token"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckContainerVolume2(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 1)
	pod.Spec.Containers[0].Name = "test"

	pod.Spec.Containers[0].VolumeMounts = make([]v1.VolumeMount, 1)
	pod.Spec.Containers[0].VolumeMounts[0].Name = "dubbo-ca-cert"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}

func TestCheckContainerVolume3(t *testing.T) {
	t.Parallel()

	options := &config.Options{
		IsKubernetesConnected: false,
		Namespace:             "dubbo-system",
		ServiceName:           "dubbo-ca",
		PlainServerPort:       30060,
		SecureServerPort:      30062,
		DebugPort:             30070,
		WebhookPort:           30080,
		WebhookAllowOnErr:     false,
		CaValidity:            30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity:          1 * 60 * 60 * 1000,       // 1 hour
	}

	sdk := NewJavaSdk(options, &fakeKubeClient{})
	pod := &v1.Pod{}

	pod.Namespace = "matched"

	pod.Spec.Containers = make([]v1.Container, 2)
	pod.Spec.Containers[0].Name = "test"
	pod.Spec.Containers[1].Name = "test"

	pod.Spec.Containers[1].VolumeMounts = make([]v1.VolumeMount, 1)
	pod.Spec.Containers[1].VolumeMounts[0].Name = "dubbo-ca-cert"

	newPod, _ := sdk.NewPod(pod)

	if !reflect.DeepEqual(newPod, pod) {
		t.Error("should be equal")
	}
}
