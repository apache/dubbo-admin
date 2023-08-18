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
	"fmt"

	dubbo_apache_org_v1alpha1 "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/gen/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/gen/generated/clientset/versioned"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func create(ic versioned.Interface, cfg model.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthenticationPolicies(cfg.Namespace).Create(context.TODO(), &v1alpha1.AuthenticationPolicy{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)),
		}, metav1.CreateOptions{})
	case collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthorizationPolicies(cfg.Namespace).Create(context.TODO(), &v1alpha1.AuthorizationPolicy{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)),
		}, metav1.CreateOptions{})
	case collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ConditionRoutes(cfg.Namespace).Create(context.TODO(), &v1alpha1.ConditionRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.ConditionRoute)),
		}, metav1.CreateOptions{})
	case collections.DubboApacheOrgV1Alpha1DynamicConfig.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().DynamicConfigs(cfg.Namespace).Create(context.TODO(), &v1alpha1.DynamicConfig{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.DynamicConfig)),
		}, metav1.CreateOptions{})
	case collections.DubboApacheOrgV1Alpha1ServiceNameMapping.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ServiceNameMappings(cfg.Namespace).Create(context.TODO(), &v1alpha1.ServiceNameMapping{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.ServiceNameMapping)),
		}, metav1.CreateOptions{})
	case collections.DubboApacheOrgV1Alpha1TagRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().TagRoutes(cfg.Namespace).Create(context.TODO(), &v1alpha1.TagRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.TagRoute)),
		}, metav1.CreateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func update(ic versioned.Interface, cfg model.Config, objMeta metav1.ObjectMeta) (metav1.Object, error) {
	switch cfg.GroupVersionKind {
	case collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthenticationPolicies(cfg.Namespace).Update(context.TODO(), &v1alpha1.AuthenticationPolicy{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.AuthenticationPolicy)),
		}, metav1.UpdateOptions{})
	case collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthorizationPolicies(cfg.Namespace).Update(context.TODO(), &v1alpha1.AuthorizationPolicy{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.AuthorizationPolicy)),
		}, metav1.UpdateOptions{})
	case collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ConditionRoutes(cfg.Namespace).Update(context.TODO(), &v1alpha1.ConditionRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.ConditionRoute)),
		}, metav1.UpdateOptions{})
	case collections.DubboApacheOrgV1Alpha1DynamicConfig.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().DynamicConfigs(cfg.Namespace).Update(context.TODO(), &v1alpha1.DynamicConfig{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.DynamicConfig)),
		}, metav1.UpdateOptions{})
	case collections.DubboApacheOrgV1Alpha1ServiceNameMapping.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ServiceNameMappings(cfg.Namespace).Update(context.TODO(), &v1alpha1.ServiceNameMapping{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.ServiceNameMapping)),
		}, metav1.UpdateOptions{})
	case collections.DubboApacheOrgV1Alpha1TagRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().TagRoutes(cfg.Namespace).Update(context.TODO(), &v1alpha1.TagRoute{
			ObjectMeta: objMeta,
			Spec:       *(cfg.Spec.(*dubbo_apache_org_v1alpha1.TagRoute)),
		}, metav1.UpdateOptions{})
	default:
		return nil, fmt.Errorf("unsupported type: %v", cfg.GroupVersionKind)
	}
}

func delete(ic versioned.Interface, typ model.GroupVersionKind, name, namespace string, resourceVersion *string) error {
	var deleteOptions metav1.DeleteOptions
	if resourceVersion != nil {
		deleteOptions.Preconditions = &metav1.Preconditions{ResourceVersion: resourceVersion}
	}
	switch typ {
	case collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthenticationPolicies(namespace).Delete(context.TODO(), name, deleteOptions)
	case collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().AuthorizationPolicies(namespace).Delete(context.TODO(), name, deleteOptions)
	case collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ConditionRoutes(namespace).Delete(context.TODO(), name, deleteOptions)
	case collections.DubboApacheOrgV1Alpha1DynamicConfig.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().DynamicConfigs(namespace).Delete(context.TODO(), name, deleteOptions)
	case collections.DubboApacheOrgV1Alpha1ServiceNameMapping.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().ServiceNameMappings(namespace).Delete(context.TODO(), name, deleteOptions)
	case collections.DubboApacheOrgV1Alpha1TagRoute.Resource().GroupVersionKind():
		return ic.DubboV1alpha1().TagRoutes(namespace).Delete(context.TODO(), name, deleteOptions)
	default:
		return fmt.Errorf("unsupported type: %v", typ)
	}
}

var translationMap = map[model.GroupVersionKind]func(r runtime.Object) *model.Config{
	collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.AuthenticationPolicy)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1AuthenticationPolicy.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.AuthorizationPolicy)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1AuthorizationPolicy.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.ConditionRoute)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1ConditionRoute.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	collections.DubboApacheOrgV1Alpha1DynamicConfig.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.DynamicConfig)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1DynamicConfig.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	collections.DubboApacheOrgV1Alpha1ServiceNameMapping.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.ServiceNameMapping)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1ServiceNameMapping.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
	collections.DubboApacheOrgV1Alpha1TagRoute.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.TagRoute)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboApacheOrgV1Alpha1TagRoute.Resource().GroupVersionKind(),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				UID:               string(obj.UID),
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
}
