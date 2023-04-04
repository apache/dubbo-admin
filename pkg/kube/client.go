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
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CtlClient wraps controller-runtime client and is used by dubboctl
type CtlClient struct {
	client.Client
}

// ApplyManifest applies manifest to certain namespace
// If there is not this namespace, create it first
func (cli *CtlClient) ApplyManifest(manifest string, ns string) error {
	if err := cli.createNamespace(ns); err != nil {
		return err
	}
	objs, err := ParseObjectsFromManifest(manifest, false)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if err := cli.ApplyObject(obj.Unstructured()); err != nil {
			return err
		}
	}
	return nil
}

// ApplyObject creates or updates unstructured object
func (cli *CtlClient) ApplyObject(obj *unstructured.Unstructured) error {
	if obj.GetKind() == "List" {
		objList, err := obj.ToList()
		if err != nil {
			return err
		}
		for _, item := range objList.Items {
			if err := cli.ApplyObject(&item); err != nil {
				return err
			}
		}
		return nil
	}
	key := client.ObjectKeyFromObject(obj)
	receiver := &unstructured.Unstructured{}
	receiver.SetGroupVersionKind(obj.GroupVersionKind())

	if err := retry.RetryOnConflict(wait.Backoff{
		Duration: time.Millisecond * 10,
		Factor:   2,
		Steps:    3,
	}, func() error {
		if err := cli.Get(context.Background(), key, receiver); err != nil {
			if errors.IsNotFound(err) {
				if err := cli.Create(context.Background(), obj); err != nil {
					return err
				}
			}
			// log
			return nil
		}
		if err := OverlayObject(receiver, obj); err != nil {
			return err
		}
		if err := cli.Update(context.Background(), receiver); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (cli *CtlClient) createNamespace(ns string) error {
	key := client.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      ns,
	}
	if err := cli.Get(context.Background(), key, &corev1.Namespace{}); err != nil {
		if errors.IsNotFound(err) {
			nsObj := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: metav1.NamespaceSystem,
					Name:      ns,
				},
			}
			if err := cli.Create(context.Background(), nsObj); err != nil {
				return err
			}
			return nil
		}

		return fmt.Errorf("failed to check if namespace %v exists: %v", ns, err)
	}

	return nil
}

func NewCtlClient(kubeConfigPath string, ctx string) (*CtlClient, error) {
	cfg, err := BuildConfig(kubeConfigPath, ctx)
	if err != nil {
		return nil, err
	}
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	ctlCli := &CtlClient{
		cli,
	}
	return ctlCli, nil
}
