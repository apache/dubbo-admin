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

package crdclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/gen/generated/clientset/versioned"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient/client"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"

	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/queue"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

// Client is a client for Dubbo CRDs, implementing config store cache
// This is used for handling of events on config changes
type Client struct {
	// schemas defines the set of schemas used by this client.
	schemas collection.Schemas

	// domainSuffix for the config metadata
	domainSuffix string

	// kinds keeps track of all cache handlers for known types
	kinds map[model.GroupVersionKind]*cacheHandler
	queue queue.Instance

	dubboClient versioned.Interface
}

// Create implements store interface
func (cl *Client) Create(cfg model.Config) (string, error) {
	if cfg.Spec == nil {
		return "", fmt.Errorf("nil spec for %v/%v", cfg.Name, cfg.Namespace)
	}

	meta, err := create(cl.dubboClient, cfg, getObjectMetadata(cfg))
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) Update(cfg model.Config) (string, error) {
	if cfg.Spec == nil {
		return "", fmt.Errorf("nil spec for %v/%v", cfg.Name, cfg.Namespace)
	}

	meta, err := update(cl.dubboClient, cfg, getObjectMetadata(cfg))
	if err != nil {
		return "", err
	}
	return meta.GetResourceVersion(), nil
}

func (cl *Client) Delete(typ model.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	return delete(cl.dubboClient, typ, name, namespace, resourceVersion)
}

func getObjectMetadata(config model.Config) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:            config.Name,
		Namespace:       config.Namespace,
		Labels:          config.Labels,
		Annotations:     config.Annotations,
		ResourceVersion: config.ResourceVersion,
		OwnerReferences: config.OwnerReferences,
		UID:             types.UID(config.UID),
	}
}

func (cl *Client) HasSynced() bool {
	for kind, ctl := range cl.kinds {
		if !ctl.informer.HasSynced() {
			logger.Sugar().Infof("[DDS] controller %q is syncing...", kind)
			return false
		}
	}
	return true
}

// Start the queue and all informers. Callers should  wait for HasSynced() before depending on results.
func (cl *Client) Start(stop <-chan struct{}) error {
	t0 := time.Now()
	logger.Sugar().Info("[DDS] Starting Rule K8S CRD controller")

	go func() {
		cache.WaitForCacheSync(stop, cl.HasSynced)
		logger.Sugar().Info("[DDS] Rule K8S CRD controller synced", time.Since(t0))
		cl.queue.Run(stop)
	}()

	<-stop
	logger.Sugar().Info("[DDS] controller terminated")
	return nil
}

func (cl *Client) RegisterEventHandler(kind model.GroupVersionKind, handler EventHandler) {
	h, exists := cl.kinds[kind]
	if !exists {
		return
	}

	h.handlers = append(h.handlers, handler)
}

// Validate we are ready to handle events. Until the informers are synced, we will block the queue
func (cl *Client) checkReadyForEvents(curr interface{}) error {
	if !cl.HasSynced() {
		return errors.New("waiting till full synchronization")
	}
	_, err := cache.DeletionHandlingMetaNamespaceKeyFunc(curr)
	if err != nil {
		logger.Sugar().Infof("[DDS] Error retrieving key: %v", err)
	}
	return nil
}

// knownCRDs returns all CRDs present in the cluster, with retries
func knownCRDs(crdClient apiextensionsclient.Interface) map[string]struct{} {
	delay := time.Second
	maxDelay := time.Minute
	var res *v1.CustomResourceDefinitionList
	for {
		var err error
		res, err = crdClient.ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
		if err == nil {
			break
		}
		logger.Sugar().Errorf("[DDS] failed to list CRDs: %v", err)
		time.Sleep(delay)
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	mp := map[string]struct{}{}
	for _, r := range res.Items {
		mp[r.Name] = struct{}{}
	}
	return mp
}

// List implements store interface
func (cl *Client) List(kind model.GroupVersionKind, namespace string) ([]model.Config, error) {
	h, f := cl.kinds[kind]
	if !f {
		return nil, nil
	}

	list, err := h.lister(namespace).List(klabels.Everything())
	if err != nil {
		return nil, err
	}
	out := make([]model.Config, 0, len(list))
	for _, item := range list {
		cfg := TranslateObject(item, kind, cl.domainSuffix)
		out = append(out, *cfg)
	}
	return out, err
}

func (cl *Client) Schemas() collection.Schemas {
	return cl.schemas
}

func (cl *Client) Get(typ model.GroupVersionKind, name, namespace string) *model.Config {
	h, f := cl.kinds[typ]
	if !f {
		logger.Sugar().Warnf("[DDS] unknown type: %s", typ)
		return nil
	}

	obj, err := h.lister(namespace).Get(name)
	if err != nil {
		logger.Sugar().Warnf("[DDS] error on get %v/%v: %v", name, namespace, err)
		return nil
	}

	cfg := TranslateObject(obj, typ, cl.domainSuffix)
	return cfg
}

func TranslateObject(r runtime.Object, gvk model.GroupVersionKind, domainSuffix string) *model.Config {
	translateFunc, f := translationMap[gvk]
	if !f {
		logger.Sugar().Errorf("[DDS] unknown type %v", gvk)
		return nil
	}
	c := translateFunc(r)
	c.Domain = domainSuffix
	return c
}

func New(client *client.KubeClient, domainSuffix string) (ConfigStoreCache, error) {
	schemas := collections.Rule
	return NewForSchemas(client, domainSuffix, schemas)
}

func NewForSchemas(client *client.KubeClient, domainSuffix string, schemas collection.Schemas) (ConfigStoreCache, error) {
	out := &Client{
		schemas:      schemas,
		domainSuffix: domainSuffix,
		kinds:        map[model.GroupVersionKind]*cacheHandler{},
		queue:        queue.NewQueue(1 * time.Second),
		dubboClient:  client.DubboClientSet(),
	}
	known := knownCRDs(client.Ext())
	for _, s := range out.schemas.All() {
		name := fmt.Sprintf("%s.%s", s.Resource().Plural(), s.Resource().Group())
		if _, f := known[name]; f {
			var i informers.GenericInformer
			var err error
			i, err = client.DubboInformer().ForResource(s.Resource().GroupVersionResource())
			if err != nil {
				return nil, err
			}
			out.kinds[s.Resource().GroupVersionKind()] = createCacheHandler(out, s, i)
		} else {
			logger.Sugar().Warnf("[DDS] Skipping CRD %v as it is not present", s.Resource().GroupVersionKind())
		}
	}

	return out, nil
}
