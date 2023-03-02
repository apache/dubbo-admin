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

package k8s

import (
	"github.com/apache/dubbo-admin/ca/pkg/apis/dubbo.apache.org/v1beta1"
	clientset "github.com/apache/dubbo-admin/ca/pkg/generated/clientset/versioned"
	informers "github.com/apache/dubbo-admin/ca/pkg/generated/informers/externalversions/dubbo.apache.org/v1beta1"
	listers "github.com/apache/dubbo-admin/ca/pkg/generated/listers/dubbo.apache.org/v1beta1"
	"github.com/apache/dubbo-admin/ca/pkg/logger"
	"github.com/apache/dubbo-admin/ca/pkg/rule/authentication"
	"k8s.io/client-go/tools/cache"
)

type NotificationType int

const (
	// AddNotification is a notification type for add events.
	AddNotification NotificationType = iota
	// UpdateNotification is a notification type for update events.
	UpdateNotification
	// DeleteNotification is a notification type for delete events.
	DeleteNotification
)

// Controller is the controller implementation for Foo resources
type Controller struct {
	// dubboClientSet is a clientset for our own API group
	dubboClientSet clientset.Interface

	paListener listers.PeerAuthenticationLister
	paSynced   cache.InformerSynced

	paHandler authentication.Handler
}

// NewController returns a new sample controller
func NewController(
	clientSet clientset.Interface,
	paHandler authentication.Handler,
	paInformer informers.PeerAuthenticationInformer) *Controller {

	controller := &Controller{
		dubboClientSet: clientSet,
		paListener:     paInformer.Lister(),
		paSynced:       paInformer.Informer().HasSynced,

		//workQueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "Dubbo-Authority"),
		paHandler: paHandler,
	}

	logger.Sugar.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	_, err := paInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.handleEvent(obj, AddNotification)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			controller.handleEvent(newObj, UpdateNotification)
		},
		DeleteFunc: func(obj interface{}) {
			controller.handleEvent(obj, DeleteNotification)
		},
	})
	if err != nil {
		return nil
	}

	return controller
}

func (c *Controller) handleEvent(obj interface{}, eventType NotificationType) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Sugar.Errorf("error getting key for object: %v", err)
		return
	}

	pa, ok := obj.(*v1beta1.PeerAuthentication)
	if !ok {
		logger.Sugar.Errorf("unexpected object type: %v", obj)
		return
	}

	a := CopyToAuthentication(key, pa)

	switch eventType {
	case AddNotification:
		c.paHandler.Add(key, a)
	case UpdateNotification:
		c.paHandler.Update(key, a)
	case DeleteNotification:
		c.paHandler.Delete(key)
	}
}

func CopyToAuthentication(key string, pa *v1beta1.PeerAuthentication) *authentication.PeerAuthentication {
	a := &authentication.PeerAuthentication{}
	a.Name = key
	a.Spec = &authentication.PeerAuthenticationSpec{}
	a.Spec.Action = pa.Spec.Action
	if pa.Spec.Rules != nil {
		for _, rule := range pa.Spec.Rules {
			r := &authentication.Rule{
				From: &authentication.Source{
					Namespaces:    rule.From.Namespaces,
					NotNamespaces: rule.From.NotNamespaces,
					IpBlocks:      rule.From.IpBlocks,
					NotIpBlocks:   rule.From.NotIpBlocks,
					Principals:    rule.From.Principals,
					NotPrincipals: rule.From.NotPrincipals,
				},
				To: &authentication.Target{
					IpBlocks:      rule.To.IpBlocks,
					NotIpBlocks:   rule.To.NotIpBlocks,
					Principals:    rule.To.Principals,
					NotPrincipals: rule.To.NotPrincipals,
				},
			}
			if rule.From.Extends != nil {
				for _, extends := range rule.From.Extends {
					r.From.Extends = append(r.From.Extends, &authentication.ExtendConfig{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.From.NotExtends != nil {
				for _, notExtend := range rule.From.NotExtends {
					r.From.NotExtends = append(r.From.NotExtends, &authentication.ExtendConfig{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			if rule.To.Extends != nil {
				for _, extends := range rule.To.Extends {
					r.To.Extends = append(r.To.Extends, &authentication.ExtendConfig{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.To.NotExtends != nil {
				for _, notExtend := range rule.To.NotExtends {
					r.To.NotExtends = append(r.To.NotExtends, &authentication.ExtendConfig{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			a.Spec.Rules = append(a.Spec.Rules, r)
		}
	}
	a.Spec.Order = pa.Spec.Order
	a.Spec.MatchType = pa.Spec.MatchType
	return a
}
