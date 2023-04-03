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

package kube

import (
	"context"
	"fmt"
	jsonpatch "github.com/evanphx/json-patch/v5"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
	"os"
	"strconv"
	"strings"
)

// OverlayObject uses JSON patch strategy to overlay two unstructured objects
func OverlayObject(base *unstructured.Unstructured, overlay *unstructured.Unstructured) error {
	baseRaw, err := runtime.Encode(unstructured.UnstructuredJSONScheme, base)
	if err != nil {
		return err
	}

	overlayUpdated := overlay.DeepCopy()
	if strings.EqualFold(base.GetKind(), "service") {
		if err := saveClusterIP(base, overlayUpdated); err != nil {
			return err
		}

		saveNodePorts(base, overlayUpdated)
	}

	overlayRaw, err := runtime.Encode(unstructured.UnstructuredJSONScheme, overlayUpdated)
	if err != nil {
		return err
	}
	merged, err := jsonpatch.MergePatch(baseRaw, overlayRaw)
	if err != nil {
		return err
	}
	return runtime.DecodeInto(unstructured.UnstructuredJSONScheme, merged, base)
}

// saveClusterIP copies the cluster IP from the current cluster into the overlay
func saveClusterIP(current, overlay *unstructured.Unstructured) error {
	// Save the value of spec.clusterIP set by the cluster
	if clusterIP, found, err := unstructured.NestedString(current.Object, "spec",
		"clusterIP"); err != nil {
		return err
	} else if found {
		if err := unstructured.SetNestedField(overlay.Object, clusterIP, "spec",
			"clusterIP"); err != nil {
			return err
		}
	}
	return nil
}

// saveNodePorts transfers the port values from the current cluster into the overlay
func saveNodePorts(current, overlay *unstructured.Unstructured) {
	portMap := createPortMap(current)
	ports, _, _ := unstructured.NestedFieldNoCopy(overlay.Object, "spec", "ports")
	portList, ok := ports.([]any)
	if !ok {
		return
	}
	for _, port := range portList {
		m, ok := port.(map[string]any)
		if !ok {
			continue
		}
		if nodePortNum, ok := m["nodePort"]; ok && fmt.Sprintf("%v", nodePortNum) == "0" {
			if portNum, ok := m["port"]; ok {
				if v, ok := portMap[fmt.Sprintf("%v", portNum)]; ok {
					m["nodePort"] = v
				}
			}
		}
	}
}

// createPortMap returns a map, mapping the value of the port and value of the nodePort
func createPortMap(current *unstructured.Unstructured) map[string]uint32 {
	portMap := make(map[string]uint32)
	svc := &v1.Service{}
	if err := scheme.Scheme.Convert(current, svc, nil); err != nil {
		//log.Error(err.Error())
		return portMap
	}
	for _, p := range svc.Spec.Ports {
		portMap[strconv.Itoa(int(p.Port))] = uint32(p.NodePort)
	}
	return portMap
}

func BuildConfig(kubecfgPath string, ctx string) (*rest.Config, error) {
	if kubecfgPath != "" {
		info, err := os.Stat(kubecfgPath)
		if err != nil || info.Size() == 0 {
			// If the specified kube config file doesn't exist / empty file / any other error
			// from file stat, fall back to default
			kubecfgPath = ""
		}
	}

	// Config loading rules:
	// 1. kubeconfig if it not empty string
	// 2. Config(s) in KUBECONFIG environment variable
	// 3. In cluster config if running in-cluster
	// 4. Use $HOME/.kube/config
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
	loadingRules.ExplicitPath = kubecfgPath
	configOverrides := &clientcmd.ConfigOverrides{
		ClusterDefaults: clientcmd.ClusterDefaults,
		CurrentContext:  ctx,
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}
	setDefaults(cfg)
	return cfg, nil
}

func setDefaults(cfg *rest.Config) {
	// todo:// add schema
}

// CreateNamespace creates a namespace using the given k8s interface.
func CreateNamespace(cs kubernetes.Interface, namespace string, network string, dryRun bool) error {
	if dryRun {
		//scope.Infof("Not applying Namespace %s because of dry run.", namespace)
		return nil
	}

	// check if the namespace already exists. If yes, do nothing. If no, create a new one.
	if _, err := cs.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			ns := &v1.Namespace{ObjectMeta: metav1.ObjectMeta{
				Name:   namespace,
				Labels: map[string]string{},
			}}
			_, err := cs.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("failed to create namespace %v: %v", namespace, err)
			}

			return nil
		}

		return fmt.Errorf("failed to check if namespace %v exists: %v", namespace, err)
	}

	return nil
}
