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

package collections

import (
	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
	"reflect"
)

var (
	DubboCAV1Alpha1Authentication = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/AuthenticationPolicy",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "AuthenticationPolicy",
			Plural:        "authenticationpolicies",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.AuthenticationPolicy",
			ReflectType:   reflect.TypeOf(&api.AuthenticationPolicy{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboCAV1Alpha1Authorization = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/AuthorizationPolicy",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "AuthorizationPolicy",
			Plural:        "authorizationpolicies",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.AuthorizationPolicy",
			ReflectType:   reflect.TypeOf(&api.AuthorizationPolicy{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboNetWorkV1Alpha1ConditionRoute = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/ConditionRoute",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "ConditionRoute",
			Plural:        "conditionroutes",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.ConditionRoute",
			ReflectType:   reflect.TypeOf(&api.ConditionRoute{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboNetWorkV1Alpha1TagRoute = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/TagRoute",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "TagRoute",
			Plural:        "tagroutes",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.TagRoute",
			ReflectType:   reflect.TypeOf(&api.TagRoute{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboNetWorkV1Alpha1DynamicConfig = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/DynamicConfig",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "DynamicConfig",
			Plural:        "dynamicconfigs",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.DynamicConfig",
			ReflectType:   reflect.TypeOf(&api.DynamicConfig{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboServiceV1Alpha1ServiceMapping = collection.Builder{
		Name: "dubbo/apache/org/v1alpha1/ServiceNameMapping",
		Resource: resource.Builder{
			ClusterScoped: false,
			Kind:          "ServiceNameMapping",
			Plural:        "servicenamemappings",
			Group:         "dubbo.apache.org",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.ServiceNameMapping",
			ReflectType:   reflect.TypeOf(&api.ServiceNameMapping{}).Elem(),
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	Rule = collection.NewSchemasBuilder().
		MustAdd(DubboCAV1Alpha1Authentication).
		MustAdd(DubboCAV1Alpha1Authorization).
		MustAdd(DubboServiceV1Alpha1ServiceMapping).
		MustAdd(DubboNetWorkV1Alpha1ConditionRoute).
		MustAdd(DubboNetWorkV1Alpha1DynamicConfig).
		MustAdd(DubboNetWorkV1Alpha1TagRoute).
		Build()
)
