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

package webhook_test

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"

	admissionV1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"

	"github.com/apache/dubbo-admin/pkg/authority/config"

	"github.com/apache/dubbo-admin/pkg/authority/webhook"

	"github.com/apache/dubbo-admin/pkg/authority/cert"
)

func TestServe(t *testing.T) {
	t.Parallel()

	authority := cert.GenerateAuthorityCert(nil, 60*60*1000)
	c := cert.SignServerCert(authority, []string{"localhost"}, 60*60*1000)

	server := webhook.NewWebhook(func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return c.GetTlsCert(), nil
	})

	port := getAvailablePort()

	server.Init(&config.Options{
		WebhookPort: port,
	})

	go server.Serve()

	assert.Eventually(t, func() bool {
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM([]byte(authority.CertPem))
		trans := &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: caCertPool,
			},
		}

		client := http.Client{Transport: trans, Timeout: 15 * time.Second}
		res, err := client.Get("https://localhost:" + strconv.Itoa(int(port)) + "/health")
		if err != nil {
			t.Log("server is not ready: ", err)

			return false
		}

		if res.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code: ", res.StatusCode)

			return false
		}

		return true
	}, 30*time.Second, 1*time.Second, "server should be ready")

	server.Stop()
}

func getAvailablePort() int32 {
	address, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:0")
	listener, _ := net.ListenTCP("tcp", address)

	defer listener.Close()

	return int32(listener.Addr().(*net.TCPAddr).Port)
}

func TestMutate_MediaError1(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	request, err := http.NewRequest("POST", "/mutating-services", nil)
	if err != nil {
		t.Fatal(err)
	}

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestMutate_MediaError2(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	request, err := http.NewRequest("POST", "/mutating-services", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/xml")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusUnsupportedMediaType, w.Code)
}

func TestMutate_BodyError(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	data := "{"

	request, err := http.NewRequest("POST", "/mutating-services", strings.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	assert.Nil(t, err)

	expected, err := json.Marshal(admissionV1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionV1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: "unexpected end of JSON input",
				Reason:  metav1.StatusReason("unexpected end of JSON input"),
			},
		},
	})

	assert.Equal(t, string(expected), string(body))
	assert.Nil(t, err)
}

func TestMutate_AdmitEmpty(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	data, err := json.Marshal(admissionV1.AdmissionReview{})

	assert.Nil(t, err)

	request, err := http.NewRequest("POST", "/mutating-services", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	assert.Nil(t, err)

	expected, err := json.Marshal(admissionV1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionV1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: "[Webhook] AdmissionReview request is nil",
				Reason:  "[Webhook] AdmissionReview request is nil",
			},
		},
	})

	assert.Equal(t, string(expected), string(body))
	assert.Nil(t, err)
}

func TestMutate_AdmitErrorType(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	data, err := json.Marshal(admissionV1.AdmissionReview{
		Request: &admissionV1.AdmissionRequest{
			UID:      "123",
			Resource: metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "deployments"},
		},
	})

	assert.Nil(t, err)

	request, err := http.NewRequest("POST", "/mutating-services", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	assert.Nil(t, err)

	expected, err := json.Marshal(admissionV1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionV1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: "[Webhook] expect resource to be pods, but actual is { v1 deployments}",
				Reason:  "[Webhook] expect resource to be pods, but actual is { v1 deployments}",
			},
		},
	})

	assert.Equal(t, string(expected), string(body))
	assert.Nil(t, err)
}

func TestMutate_AdmitPodPatchErr(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	server.Patches = []webhook.PodPatch{
		func(pod *v1.Pod) (*v1.Pod, error) {
			if pod.Name == "" {
				return nil, fmt.Errorf("Name is empty")
			}
			pod.Name = "Target"
			return pod, nil
		},
	}

	data, err := json.Marshal(admissionV1.AdmissionReview{
		Request: &admissionV1.AdmissionRequest{
			UID:      "123",
			Resource: metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Object: runtime.RawExtension{
				Raw: []byte("{}"),
			},
		},
	})

	assert.Nil(t, err)

	request, err := http.NewRequest("POST", "/mutating-services", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	assert.Nil(t, err)

	expected, err := json.Marshal(admissionV1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionV1.AdmissionResponse{
			Allowed: true,
			Result: &metav1.Status{
				Status:  "Failure",
				Message: "[Webhook] Patch error: . Msg: [Webhook] Pod patch failed: Name is empty",
				Reason:  "[Webhook] Patch error: . Msg: [Webhook] Pod patch failed: Name is empty",
			},
		},
	})

	assert.Equal(t, string(expected), string(body))
	assert.Nil(t, err)
}

func TestMutate_AdmitPodPatch(t *testing.T) {
	t.Parallel()

	server := webhook.NewWebhook(nil)

	server.Patches = []webhook.PodPatch{
		func(pod *v1.Pod) (*v1.Pod, error) {
			if pod.Name == "" {
				return nil, fmt.Errorf("Name is empty")
			}
			pod.Name = "Target"
			return pod, nil
		},
	}

	originPod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}

	origin, err := json.Marshal(originPod)
	assert.Nil(t, err)

	data, err := json.Marshal(admissionV1.AdmissionReview{
		Request: &admissionV1.AdmissionRequest{
			UID:      "123",
			Resource: metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
			Object: runtime.RawExtension{
				Raw: origin,
			},
		},
	})

	assert.Nil(t, err)

	request, err := http.NewRequest("POST", "/mutating-services", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.Mutate(w, request)

	assert.Equal(t, http.StatusOK, w.Code)

	body, err := io.ReadAll(w.Body)
	assert.Nil(t, err)
	patchType := admissionV1.PatchTypeJSONPatch

	expected, err := json.Marshal(admissionV1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Response: &admissionV1.AdmissionResponse{
			UID:       "123",
			Allowed:   true,
			Patch:     []byte("[{\"op\":\"replace\",\"path\":\"/metadata/name\",\"value\":\"Target\"}]"),
			PatchType: &patchType,
		},
	})

	assert.Equal(t, string(expected), string(body))
	assert.Nil(t, err)
}
