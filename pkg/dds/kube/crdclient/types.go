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
	"github.com/apache/dubbo-admin/pkg/core/gen/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"k8s.io/apimachinery/pkg/runtime"
)

var translationMap = map[model.GroupVersionKind]func(r runtime.Object) *model.Config{
	collections.DubboCAV1Alpha1Authentication.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.AuthenticationPolicy)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboCAV1Alpha1Authentication.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},

	collections.DubboCAV1Alpha1Authorization.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.AuthorizationPolicy)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboCAV1Alpha1Authorization.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},

	collections.DubboServiceV1Alpha1ServiceMapping.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.ServiceNameMapping)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboServiceV1Alpha1ServiceMapping.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},

	collections.DubboNetWorkV1Alpha1TagRoute.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.TagRoute)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboNetWorkV1Alpha1TagRoute.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},

	collections.DubboNetWorkV1Alpha1DynamicConfig.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.DynamicConfig)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboNetWorkV1Alpha1DynamicConfig.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},

	collections.DubboNetWorkV1Alpha1ConditionRoute.Resource().GroupVersionKind(): func(r runtime.Object) *model.Config {
		obj := r.(*v1alpha1.ConditionRoute)
		return &model.Config{
			Meta: model.Meta{
				GroupVersionKind:  collections.DubboNetWorkV1Alpha1ConditionRoute.Resource().GroupVersionKind(),
				UID:               string(obj.UID),
				Name:              obj.Name,
				Namespace:         obj.Namespace,
				Labels:            obj.Labels,
				Annotations:       obj.Annotations,
				ResourceVersion:   obj.ResourceVersion,
				CreationTimestamp: obj.CreationTimestamp.Time,
				OwnerReferences:   obj.OwnerReferences,
				Generation:        obj.Generation,
			},
			Spec: &obj.Spec,
		}
	},
}
