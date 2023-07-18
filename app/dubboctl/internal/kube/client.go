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

// CtlClient wraps controller-runtime clientgen and is used by dubboctl
type CtlClient struct {
	opts *CtlClientOptions
	client.Client
}

type CtlClientOptions struct {
	// for normal use
	// path to kubeconfig
	KubeConfigPath string
	// specify cluster in kubeconfig to use
	Context string

	// for test
	Cli client.Client
}

type CtlClientOption func(*CtlClientOptions)

func WithKubeConfigPath(path string) CtlClientOption {
	return func(opts *CtlClientOptions) {
		opts.KubeConfigPath = path
	}
}

func WithContext(ctx string) CtlClientOption {
	return func(opts *CtlClientOptions) {
		opts.Context = ctx
	}
}

func WithCli(cli client.Client) CtlClientOption {
	return func(opts *CtlClientOptions) {
		opts.Cli = cli
	}
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
		if obj.Namespace == "" {
			obj.SetNamespace(ns)
		}
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

func (cli *CtlClient) RemoveManifest(manifest string, ns string) error {
	objs, err := ParseObjectsFromManifest(manifest, false)
	if err != nil {
		return err
	}
	for _, obj := range objs {
		if obj.Namespace == "" {
			obj.SetNamespace(ns)
		}
		if err := cli.RemoveObject(obj.Unstructured()); err != nil {
			return err
		}
	}
	if err := cli.deleteNamespace(ns); err != nil {
		return err
	}
	return nil
}

func (cli *CtlClient) RemoveObject(obj *unstructured.Unstructured) error {
	if obj.GetKind() == "List" {
		objList, err := obj.ToList()
		if err != nil {
			return err
		}
		for _, item := range objList.Items {
			if err := cli.RemoveObject(&item); err != nil {
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
			if !errors.IsNotFound(err) {
				// log
				return err
			}
			return nil
		}
		if err := cli.Delete(context.Background(), receiver, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (cli *CtlClient) deleteNamespace(ns string) error {
	key := client.ObjectKey{
		Namespace: metav1.NamespaceSystem,
		Name:      ns,
	}
	nsObj := &corev1.Namespace{}
	if err := cli.Get(context.Background(), key, nsObj); err != nil {
		if !errors.IsNotFound(err) {
			// log
			return fmt.Errorf("failed to check if namespace %v exists: %v", ns, err)
		}
		return nil
	} else {
		if err := cli.Delete(context.Background(), nsObj); err != nil {
			return fmt.Errorf("failed to delete namespace: %s, err: %s", ns, err)
		}
		return nil
	}
}

func NewCtlClient(opts ...CtlClientOption) (*CtlClient, error) {
	var ctlCli *CtlClient
	newOptions := &CtlClientOptions{}
	for _, opt := range opts {
		opt(newOptions)
	}
	// for test
	if newOptions.Cli != nil {
		ctlCli = &CtlClient{
			Client: newOptions.Cli,
			opts:   newOptions,
		}
		return ctlCli, nil
	}

	cfg, err := BuildConfig(newOptions.KubeConfigPath, newOptions.Context)
	if err != nil {
		return nil, err
	}
	cli, err := client.New(cfg, client.Options{})
	if err != nil {
		return nil, err
	}
	ctlCli = &CtlClient{
		Client: cli,
		opts:   newOptions,
	}
	return ctlCli, nil
}
