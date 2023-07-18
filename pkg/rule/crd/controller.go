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

package crd

import (
	"reflect"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/queue"
	apiV1beta1 "github.com/apache/dubbo-admin/pkg/rule/apis/dubbo.apache.org/v1beta1"
	informerV1beta1 "github.com/apache/dubbo-admin/pkg/rule/clientgen/informers/externalversions/dubbo.apache.org/v1beta1"
	"github.com/apache/dubbo-admin/pkg/rule/crd/authentication"
	"github.com/apache/dubbo-admin/pkg/rule/crd/authorization"
	"github.com/apache/dubbo-admin/pkg/rule/crd/conditionroute"
	"github.com/apache/dubbo-admin/pkg/rule/crd/dynamicconfig"
	"github.com/apache/dubbo-admin/pkg/rule/crd/servicemapping"
	"github.com/apache/dubbo-admin/pkg/rule/crd/tagroute"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/strings/slices"
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
	rootNamespace string

	Queue queue.Instance

	authenticationSynced cache.InformerSynced
	authorizationSynced  cache.InformerSynced
	serviceMappingSynced cache.InformerSynced
	tagRouteSynced       cache.InformerSynced
	conditionRouteSynced cache.InformerSynced
	dynamicConfigSynced  cache.InformerSynced

	authenticationHandler authentication.Handler
	authorizationHandler  authorization.Handler
	serviceMappingHandler servicemapping.Handler
	conditionRouteHandler conditionroute.Handler
	tagRouteHandler       tagroute.Handler
	dynamicConfigHandler  dynamicconfig.Handler
}

// NewController returns a new sample controller
func NewController(
	rootNamespace string,
	authenticationHandler authentication.Handler,
	authorizationHandler authorization.Handler,
	serviceMappingHandler servicemapping.Handler,
	tagRouteHandler tagroute.Handler,
	conditionRouteHandler conditionroute.Handler,
	dynamicConfigHandler dynamicconfig.Handler,
	acInformer informerV1beta1.AuthenticationPolicyInformer,
	apInformer informerV1beta1.AuthorizationPolicyInformer,
	smInformer informerV1beta1.ServiceNameMappingInformer,
	tgInformer informerV1beta1.TagRouteInformer,
	cdInformer informerV1beta1.ConditionRouteInformer,
	dcInformer informerV1beta1.DynamicConfigInformer,
) *Controller {
	controller := &Controller{
		rootNamespace: rootNamespace,

		Queue: queue.NewQueue(time.Second * 1),

		authenticationSynced: acInformer.Informer().HasSynced,
		authorizationSynced:  apInformer.Informer().HasSynced,
		serviceMappingSynced: smInformer.Informer().HasSynced,
		tagRouteSynced:       tgInformer.Informer().HasSynced,
		conditionRouteSynced: cdInformer.Informer().HasSynced,
		dynamicConfigSynced:  dcInformer.Informer().HasSynced,

		authenticationHandler: authenticationHandler,
		authorizationHandler:  authorizationHandler,
		serviceMappingHandler: serviceMappingHandler,
		conditionRouteHandler: conditionRouteHandler,
		tagRouteHandler:       tagRouteHandler,
		dynamicConfigHandler:  dynamicConfigHandler,
	}
	logger.Sugar().Info("Setting up event handlers")
	_, err := acInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}

	_, err = apInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}

	_, err = smInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}

	_, err = tgInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}

	_, err = cdInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}

	_, err = dcInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, AddNotification)
			})
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			if reflect.DeepEqual(oldObj, newObj) {
				return
			}
			controller.Queue.Push(func() error {
				return controller.handleEvent(newObj, UpdateNotification)
			})
		},
		DeleteFunc: func(obj interface{}) {
			controller.Queue.Push(func() error {
				return controller.handleEvent(obj, DeleteNotification)
			})
		},
	})
	if err != nil {
		return nil
	}
	return controller
}

func (c *Controller) handleEvent(obj interface{}, eventType NotificationType) error {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		logger.Sugar().Errorf("error getting key for object: %v", err)
		return nil
	}
	switch o := obj.(type) {
	case *apiV1beta1.AuthenticationPolicy:
		a := CopyToAuthentication(key, c.rootNamespace, o)
		switch eventType {
		case AddNotification:
			c.authenticationHandler.Add(key, a)
		case UpdateNotification:
			c.authenticationHandler.Update(key, a)
		case DeleteNotification:
			c.authenticationHandler.Delete(key)
		}
	case *apiV1beta1.AuthorizationPolicy:
		a := CopyToAuthorization(key, c.rootNamespace, o)

		switch eventType {
		case AddNotification:
			c.authorizationHandler.Add(key, a)
		case UpdateNotification:
			c.authorizationHandler.Update(key, a)
		case DeleteNotification:
			c.authorizationHandler.Delete(key)
		}
	case *apiV1beta1.ServiceNameMapping:
		a := CopyToServiceMapping(key, c.rootNamespace, o)

		switch eventType {
		case AddNotification:
			c.serviceMappingHandler.Add(key, a)
		case UpdateNotification:
			c.serviceMappingHandler.Update(key, a)
		case DeleteNotification:
			c.serviceMappingHandler.Delete(key)
		}
	case *apiV1beta1.TagRoute:
		a := CopyToTagRoute(key, c.rootNamespace, o)

		switch eventType {
		case AddNotification:
			c.tagRouteHandler.Add(key, a)
		case UpdateNotification:
			c.tagRouteHandler.Update(key, a)
		case DeleteNotification:
			c.tagRouteHandler.Delete(key)
		}
	case *apiV1beta1.ConditionRoute:
		a := CopyToConditionRoute(key, c.rootNamespace, o)

		switch eventType {
		case AddNotification:
			c.conditionRouteHandler.Add(key, a)
		case UpdateNotification:
			c.conditionRouteHandler.Update(key, a)
		case DeleteNotification:
			c.conditionRouteHandler.Delete(key)
		}
	case *apiV1beta1.DynamicConfig:
		a := CopyToDynamicConfig(key, c.rootNamespace, o)

		switch eventType {
		case AddNotification:
			c.dynamicConfigHandler.Add(key, a)
		case UpdateNotification:
			c.dynamicConfigHandler.Update(key, a)
		case DeleteNotification:
			c.dynamicConfigHandler.Delete(key)
		}
	default:
		logger.Sugar().Errorf("unexpected object type: %v", obj)
	}
	return nil
}

func CopyToServiceMapping(key, rootNamespace string, pa *apiV1beta1.ServiceNameMapping) *servicemapping.Policy {
	a := &servicemapping.Policy{}
	a.Name = key
	a.Spec = &servicemapping.PolicySpec{}
	a.Spec.ApplicationNames = pa.Spec.ApplicationNames
	a.Spec.InterfaceName = pa.Spec.InterfaceName
	return a
}

func CopyToTagRoute(key, rootNamespace string, pa *apiV1beta1.TagRoute) *tagroute.Policy {
	a := &tagroute.Policy{}
	a.Name = key
	a.Spec = &tagroute.PolicySpec{}
	a.Spec.Priority = pa.Spec.Priority
	a.Spec.Enabled = pa.Spec.Enabled
	a.Spec.Force = pa.Spec.Force
	a.Spec.Runtime = pa.Spec.Runtime
	a.Spec.Key = pa.Spec.Key
	a.Spec.ConfigVersion = pa.Spec.ConfigVersion
	if pa.Spec.Tags != nil {
		for _, tag := range pa.Spec.Tags {
			t := &tagroute.Tag{
				Name: tag.Name,
			}
			if tag.Addresses != nil {
				for _, address := range tag.Addresses {
					t.Addresses = append(t.Addresses, address)
				}
			}
			if tag.Match != nil {
				for _, match := range tag.Match {
					p := &tagroute.ParamMatch{
						Key: match.Key,
						Value: &tagroute.StringMatch{
							Exact:    match.Value.Exact,
							Prefix:   match.Value.Prefix,
							Regex:    match.Value.Regex,
							Noempty:  match.Value.Noempty,
							Empty:    match.Value.Empty,
							Wildcard: match.Value.Wildcard,
						},
					}

					t.Match = append(t.Match, p)
				}
			}
			a.Spec.Tags = append(a.Spec.Tags, t)
		}
	}
	return a
}

func CopyToConditionRoute(key, rootNamespace string, pa *apiV1beta1.ConditionRoute) *conditionroute.Policy {
	a := &conditionroute.Policy{}
	a.Name = key
	a.Spec = &conditionroute.PolicySpec{}
	a.Spec.Key = pa.Spec.Key
	a.Spec.Conditions = pa.Spec.Conditions
	a.Spec.Runtime = pa.Spec.Runtime
	a.Spec.ConfigVersion = pa.Spec.ConfigVersion
	a.Spec.Force = pa.Spec.Force
	a.Spec.Scope = pa.Spec.Scope
	a.Spec.Priority = pa.Spec.Priority
	if pa.Spec.Conditions != nil {
		for _, rule := range pa.Spec.Conditions {
			a.Spec.Conditions = append(a.Spec.Conditions, rule)
		}
	}
	return nil
}

func CopyToDynamicConfig(key, rootNamespace string, pa *apiV1beta1.DynamicConfig) *dynamicconfig.Policy {
	a := &dynamicconfig.Policy{}
	a.Name = key
	a.Spec = &dynamicconfig.PolicySpec{}
	a.Spec.Key = pa.Spec.Key
	a.Spec.Scope = pa.Spec.Scope
	a.Spec.ConfigVersion = pa.Spec.ConfigVersion
	a.Spec.Enabled = pa.Spec.Enabled
	if pa.Spec.Configs != nil {
		for _, config := range pa.Spec.Configs {
			o := &dynamicconfig.OverrideConfig{
				Side:    config.Side,
				Type:    config.Type,
				Enabled: config.Enabled,
			}
			if config.Addresses != nil {
				for _, address := range config.Addresses {
					o.Addresses = append(o.Addresses, address)
				}
			}
			if config.ProviderAddresses != nil {
				for _, providerAddresses := range config.ProviderAddresses {
					o.ProviderAddresses = append(o.ProviderAddresses, providerAddresses)
				}
			}
			if config.Applications != nil {
				for _, application := range config.Applications {
					o.Applications = append(o.Applications, application)
				}
			}
			if config.Services != nil {
				for _, service := range config.Services {
					o.Services = append(o.Services, service)
				}
			}
			newMap := make(map[string]string)
			if config.Parameters != nil {
				for key, value := range config.Parameters {
					newMap[key] = value
				}
			}
			o.Parameters = newMap
			match := &dynamicconfig.ConditionMatch{
				Address: &dynamicconfig.AddressMatch{
					Wildcard: config.Match.Address.Wildcard,
					Cird:     config.Match.Address.Cird,
					Exact:    config.Match.Address.Exact,
				},
			}
			service := &dynamicconfig.ListStringMatch{}
			if config.Match.Service.Oneof != nil {
				for _, one := range config.Match.Service.Oneof {
					s := &dynamicconfig.StringMatch{
						Exact:    one.Exact,
						Prefix:   one.Prefix,
						Regex:    one.Regex,
						Noempty:  one.Noempty,
						Empty:    one.Empty,
						Wildcard: one.Wildcard,
					}
					service.Oneof = append(service.Oneof, s)
				}
			}
			application := &dynamicconfig.ListStringMatch{}
			if config.Match.Application.Oneof != nil {
				for _, one := range config.Match.Application.Oneof {
					s := &dynamicconfig.StringMatch{
						Exact:    one.Exact,
						Prefix:   one.Prefix,
						Regex:    one.Regex,
						Noempty:  one.Noempty,
						Empty:    one.Empty,
						Wildcard: one.Wildcard,
					}
					application.Oneof = append(application.Oneof, s)
				}
			}

			match.Service = service
			match.Application = application
			if config.Match.Param != nil {
				for _, param := range config.Match.Param {
					p := &dynamicconfig.ParamMatch{
						Key: param.Key,
						Value: &dynamicconfig.StringMatch{
							Exact:    param.Value.Exact,
							Prefix:   param.Value.Prefix,
							Regex:    param.Value.Regex,
							Noempty:  param.Value.Noempty,
							Empty:    param.Value.Empty,
							Wildcard: param.Value.Wildcard,
						},
					}
					match.Param = append(match.Param, p)
				}
			}

			o.Match = match

			a.Spec.Configs = append(a.Spec.Configs, o)
		}
	}
	return a
}

func CopyToAuthentication(key, rootNamespace string, pa *apiV1beta1.AuthenticationPolicy) *authentication.Policy {
	a := &authentication.Policy{}
	a.Name = key
	a.Spec = &authentication.PolicySpec{}
	a.Spec.Action = pa.Spec.Action
	if pa.Spec.Selector != nil {
		for _, selector := range pa.Spec.Selector {
			r := &authentication.Selector{
				Namespaces:    selector.Namespaces,
				NotNamespaces: selector.NotNamespaces,
				IpBlocks:      selector.IpBlocks,
				NotIpBlocks:   selector.NotIpBlocks,
				Principals:    selector.Principals,
				NotPrincipals: selector.NotPrincipals,
			}
			if selector.Extends != nil {
				for _, extends := range selector.Extends {
					r.Extends = append(r.Extends, &authentication.Extend{
						Key:   extends.Key,
						Value: extends.Value,
					})
				}
			}
			if selector.NotExtends != nil {
				for _, notExtend := range selector.NotExtends {
					r.NotExtends = append(r.NotExtends, &authentication.Extend{
						Key:   notExtend.Key,
						Value: notExtend.Value,
					})
				}
			}
			a.Spec.Selector = append(a.Spec.Selector, r)
		}
	}

	if pa.Spec.PortLevel != nil {
		for _, portLevel := range pa.Spec.PortLevel {
			r := &authentication.PortLevel{
				Port:   portLevel.Port,
				Action: portLevel.Action,
			}

			a.Spec.PortLevel = append(a.Spec.PortLevel, r)
		}
	}

	if rootNamespace == pa.Namespace {
		return a
	}

	if len(a.Spec.Selector) == 0 {
		a.Spec.Selector = append(a.Spec.Selector, &authentication.Selector{
			Namespaces: []string{pa.Namespace},
		})
	} else {
		for _, selector := range a.Spec.Selector {
			if !slices.Contains(selector.Namespaces, pa.Namespace) {
				selector.Namespaces = append(selector.Namespaces, pa.Namespace)
			}
		}
	}

	return a
}

func CopyToAuthorization(key, rootNamespace string, pa *apiV1beta1.AuthorizationPolicy) *authorization.Policy {
	a := &authorization.Policy{}
	a.Name = key
	a.Spec = &authorization.PolicySpec{}
	a.Spec.Action = pa.Spec.Action
	if pa.Spec.Rules != nil {
		for _, rule := range pa.Spec.Rules {
			r := &authorization.PolicyRule{
				From: &authorization.Source{
					Namespaces:    rule.From.Namespaces,
					NotNamespaces: rule.From.NotNamespaces,
					IpBlocks:      rule.From.IpBlocks,
					NotIpBlocks:   rule.From.NotIpBlocks,
					Principals:    rule.From.Principals,
					NotPrincipals: rule.From.NotPrincipals,
				},
				To: &authorization.Target{
					Namespaces:    rule.To.Namespaces,
					NotNamespaces: rule.To.NotNamespaces,
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
	a.Spec.Order = pa.Spec.Order
	a.Spec.MatchType = pa.Spec.MatchType

	if rootNamespace == pa.Namespace {
		return a
	}

	if len(a.Spec.Rules) == 0 {
		a.Spec.Rules = append(a.Spec.Rules, &authorization.PolicyRule{
			To: &authorization.Target{
				Namespaces: []string{pa.Namespace},
			},
		})
	} else {
		for _, rule := range a.Spec.Rules {
			if rule.To != nil {
				rule.To = &authorization.Target{}
			}
			if !slices.Contains(rule.To.Namespaces, pa.Namespace) {
				rule.To.Namespaces = append(rule.To.Namespaces, pa.Namespace)
			}
		}
	}
	return a
}

func (c *Controller) WaitSynced(stop <-chan struct{}) {
	logger.Sugar().Info("Waiting for informer caches to sync")

	if !cache.WaitForCacheSync(stop,
		c.authenticationSynced,
		c.authorizationSynced,
		c.serviceMappingSynced,
		c.conditionRouteSynced,
		c.tagRouteSynced,
		c.dynamicConfigSynced,
	) {
		logger.Sugar().Error("Timed out waiting for caches to sync")
		return
	} else {
		logger.Sugar().Info("Caches synced")
	}
}
