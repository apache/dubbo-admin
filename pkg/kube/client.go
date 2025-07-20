/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package kube

import (
	"fmt"

	kubeExtClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	kubeVersion "k8s.io/apimachinery/pkg/version"

	"net/http"
	"time"

	"github.com/apache/dubbo-admin/operator/pkg/config"
	"github.com/apache/dubbo-admin/pkg/common/lazy"
	"github.com/apache/dubbo-admin/pkg/kube/collections"
	"github.com/apache/dubbo-admin/pkg/kube/informerfactory"

	// "k8s.io/client-go/discoveryengine"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type client struct {
	extSet          kubeExtClient.Interface
	config          *rest.Config
	revision        string
	factory         *clientFactory
	version         lazy.Lazy[*kubeVersion.Info]
	informerFactory informerfactory.InformerFactory
	dynamic         dynamic.Interface
	kube            kubernetes.Interface
	mapper          meta.ResettableRESTMapper
	http            *http.Client
}

type Client interface {
	// Ext returns the API extensions client.
	Ext() kubeExtClient.Interface

	// Kube returns the core kube client
	Kube() kubernetes.Interface
}

type CLIClient interface {
	Client
	DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error)
}

type ClientOption func(cliClient CLIClient) CLIClient

func NewCLIClient(clientCfg clientcmd.ClientConfig, opts ...ClientOption) (CLIClient, error) {
	return newInternalClient(newClientFactory(clientCfg, false), opts...)
}

func newInternalClient(factory *clientFactory, opts ...ClientOption) (CLIClient, error) {
	var c client
	var err error
	c.factory = factory
	c.config, err = factory.ToRestConfig()
	if err != nil {
		return nil, err
	}
	for _, opt := range opts {
		opt(&c)
	}
	c.mapper, err = factory.mapper.Get()
	if err != nil {
		return nil, err
	}
	c.kube, err = kubernetes.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.dynamic, err = dynamic.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.extSet, err = kubeExtClient.NewForConfig(c.config)
	if err != nil {
		return nil, err
	}
	c.informerFactory = informerfactory.NewSharedInformerFactory()
	c.http = &http.Client{
		Timeout: time.Second * 15,
	}
	var clientWithTimeout kubernetes.Interface
	clientWithTimeout = c.kube
	restConfig := c.RESTConfig()
	if restConfig != nil {
		restConfig.Timeout = time.Second * 5
		kubeClient, err := kubernetes.NewForConfig(restConfig)
		if err == nil {
			clientWithTimeout = kubeClient
		}
	}
	c.version = lazy.NewWithRetry(clientWithTimeout.Discovery().ServerVersion)
	return &c, nil
}

func (c *client) RESTConfig() *rest.Config {
	if c.config == nil {
		return nil
	}
	cpy := *c.config
	return &cpy
}

var (
	_ Client    = &client{}
	_ CLIClient = &client{}
)

func (c *client) Ext() kubeExtClient.Interface {
	return c.extSet
}

func (c *client) Kube() kubernetes.Interface {
	return c.kube
}

func (c *client) DynamicClientFor(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (dynamic.ResourceInterface, error) {
	gvr, namespaced := c.bestEffortToGVR(gvk, obj, namespace)
	var dr dynamic.ResourceInterface
	if namespaced {
		ns := ""
		if obj != nil {
			ns = obj.GetNamespace()
		}
		if ns == "" {
			ns = namespace
		} else if namespace != "" && ns != namespace {
			return nil, fmt.Errorf("object %v/%v provided namespace %q but apply called with %q", gvk, obj.GetName(), ns, namespace)
		}
		dr = c.dynamic.Resource(gvr).Namespace(ns)
	} else {
		dr = c.dynamic.Resource(gvr)
	}
	return dr, nil
}

func (c *client) bestEffortToGVR(gvk schema.GroupVersionKind, obj *unstructured.Unstructured, namespace string) (schema.GroupVersionResource, bool) {
	if s, f := collections.All.FindByGroupVersionAliasesKind(config.FromKubernetesGVK(gvk)); f {
		gvr := s.GroupVersionResource()
		gvr.Version = gvk.Version
		return gvr, !s.IsClusterScoped()
	}
	if c.mapper != nil {
		mapping, err := c.mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err == nil {
			return mapping.Resource, mapping.Scope.Name() == meta.RESTScopeNameNamespace
		}
	}
	gvr, _ := meta.UnsafeGuessKindToResource(gvk)
	namespaced := (obj != nil && obj.GetNamespace() != "") || namespace != ""
	return gvr, namespaced
}

func WithRevision(revision string) ClientOption {
	return func(cliClient CLIClient) CLIClient {
		client := cliClient.(*client)
		client.revision = revision
		return client
	}
}
