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
	"github.com/apache/dubbo-admin/ca/pkg/logger"
	"github.com/apache/dubbo-admin/ca/pkg/rule/authentication"
	"github.com/apache/dubbo-admin/ca/pkg/rule/authorization"
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

	acSynced cache.InformerSynced
	apSynced cache.InformerSynced

	acHandler authentication.Handler
	apHandler authorization.Handler
}

// NewController returns a new sample controller
func NewController(
	clientSet clientset.Interface,
	acHandler authentication.Handler,
	apHandler authorization.Handler,
	acInformer informers.AuthenticationPolicyInformer,
	apInformer informers.AuthorizationPolicyInformer) *Controller {

	controller := &Controller{
		dubboClientSet: clientSet,
		acSynced:       acInformer.Informer().HasSynced,
		apSynced:       apInformer.Informer().HasSynced,

		acHandler: acHandler,
		apHandler: apHandler,
	}

	logger.Sugar.Info("Setting up event handlers")
	// Set up an event handler for when Foo resources change
	_, err := acInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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
	_, err = apInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
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

	switch o := obj.(type) {
	case *v1beta1.AuthenticationPolicy:
		a := CopyToAuthentication(key, o)

		switch eventType {
		case AddNotification:
			c.acHandler.Add(key, a)
		case UpdateNotification:
			c.acHandler.Update(key, a)
		case DeleteNotification:
			c.acHandler.Delete(key)
		}
		return
	case *v1beta1.AuthorizationPolicy:
		a := CopyToAuthorization(key, o)

		switch eventType {
		case AddNotification:
			c.apHandler.Add(key, a)
		case UpdateNotification:
			c.apHandler.Update(key, a)
		case DeleteNotification:
			c.apHandler.Delete(key)
		}
	default:
		logger.Sugar.Errorf("unexpected object type: %v", obj)
		return
	}

}

func CopyToAuthentication(key string, pa *v1beta1.AuthenticationPolicy) *authentication.Policy {
	a := &authentication.Policy{}
	a.Name = key
	a.Spec = &authentication.PolicySpec{}
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
					r.From.Extends = append(r.From.Extends, &authentication.Extend{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.From.NotExtends != nil {
				for _, notExtend := range rule.From.NotExtends {
					r.From.NotExtends = append(r.From.NotExtends, &authentication.Extend{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			if rule.To.Extends != nil {
				for _, extends := range rule.To.Extends {
					r.To.Extends = append(r.To.Extends, &authentication.Extend{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.To.NotExtends != nil {
				for _, notExtend := range rule.To.NotExtends {
					r.To.NotExtends = append(r.To.NotExtends, &authentication.Extend{
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

func CopyToAuthorization(key string, pa *v1beta1.AuthorizationPolicy) *authorization.Policy {
	a := &authorization.Policy{}
	a.Name = key
	a.Spec = &authorization.PolicySpec{}
	a.Spec.Action = pa.Spec.Action
	if pa.Spec.Rules != nil {
		for _, rule := range pa.Spec.Rules {
			r := &authorization.Rule{
				From: &authorization.Source{
					Namespaces:    rule.From.Namespaces,
					NotNamespaces: rule.From.NotNamespaces,
					IpBlocks:      rule.From.IpBlocks,
					NotIpBlocks:   rule.From.NotIpBlocks,
					Principals:    rule.From.Principals,
					NotPrincipals: rule.From.NotPrincipals,
				},
				To: &authorization.Target{
					IpBlocks:      rule.To.IpBlocks,
					NotIpBlocks:   rule.To.NotIpBlocks,
					Principals:    rule.To.Principals,
					NotPrincipals: rule.To.NotPrincipals,
				},
				When: &authorization.Condition{
					Key: rule.When.Key,
				},
			}
			if rule.From.Extends != nil {
				for _, extends := range rule.From.Extends {
					r.From.Extends = append(r.From.Extends, &authorization.Extend{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.From.NotExtends != nil {
				for _, notExtend := range rule.From.NotExtends {
					r.From.NotExtends = append(r.From.NotExtends, &authorization.Extend{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			if rule.To.Extends != nil {
				for _, extends := range rule.To.Extends {
					r.To.Extends = append(r.To.Extends, &authorization.Extend{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if rule.To.NotExtends != nil {
				for _, notExtend := range rule.To.NotExtends {
					r.To.NotExtends = append(r.To.NotExtends, &authorization.Extend{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			if rule.When.Values != nil {
				for _, value := range rule.When.Values {
					r.When.Values = append(r.When.Values, &authorization.Match{
						Type:  value.Type,
						Value: value.Value,
					})
				}
			}
			if rule.When.NotValues != nil {
				for _, notValue := range rule.When.NotValues {
					r.When.Values = append(r.When.Values, &authorization.Match{
						Type:  notValue.Type,
						Value: notValue.Value,
					})
				}
			}

			a.Spec.Rules = append(a.Spec.Rules, r)
		}
	}
	a.Spec.Samples = pa.Spec.Samples
	a.Spec.MatchType = pa.Spec.MatchType
	return a
}
