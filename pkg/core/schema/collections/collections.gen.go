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
	"reflect"

	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
)

var (
	DubboApacheOrgV1Alpha1AuthenticationPolicy = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/AuthenticationPolicy",
		VariableName: "DubboApacheOrgV1Alpha1AuthenticationPolicy",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "AuthenticationPolicy",
			Plural:        "authenticationpolicies",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.AuthenticationPolicy",
			ReflectType:   reflect.TypeOf(&api.AuthenticationPolicy{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboApacheOrgV1Alpha1AuthorizationPolicy = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/AuthorizationPolicy",
		VariableName: "DubboApacheOrgV1Alpha1AuthorizationPolicy",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "AuthorizationPolicy",
			Plural:        "authorizationpolicies",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.AuthorizationPolicy",
			ReflectType:   reflect.TypeOf(&api.AuthorizationPolicy{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboApacheOrgV1Alpha1ConditionRoute = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/ConditionRoute",
		VariableName: "DubboApacheOrgV1Alpha1ConditionRoute",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "ConditionRoute",
			Plural:        "conditionroutes",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.ConditionRoute",
			ReflectType:   reflect.TypeOf(&api.ConditionRoute{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboApacheOrgV1Alpha1DynamicConfig = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/DynamicConfig",
		VariableName: "DubboApacheOrgV1Alpha1DynamicConfig",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "DynamicConfig",
			Plural:        "dynamicconfigs",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.DynamicConfig",
			ReflectType:   reflect.TypeOf(&api.DynamicConfig{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboApacheOrgV1Alpha1ServiceNameMapping = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/ServiceNameMapping",
		VariableName: "DubboApacheOrgV1Alpha1ServiceNameMapping",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "ServiceNameMapping",
			Plural:        "servicenamemappings",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.ServiceNameMapping",
			ReflectType:   reflect.TypeOf(&api.ServiceNameMapping{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	DubboApacheOrgV1Alpha1TagRoute = collection.Builder{
		Name:         "dubbo/apache/org/v1alpha1/TagRoute",
		VariableName: "DubboApacheOrgV1Alpha1TagRoute",
		Resource: resource.Builder{
			Group:         "dubbo.apache.org",
			Kind:          "TagRoute",
			Plural:        "tagroutes",
			Version:       "v1alpha1",
			Proto:         "dubbo.apache.org.v1alpha1.TagRoute",
			ReflectType:   reflect.TypeOf(&api.TagRoute{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	Rule = collection.NewSchemasBuilder().
		MustAdd(DubboApacheOrgV1Alpha1AuthenticationPolicy).
		MustAdd(DubboApacheOrgV1Alpha1AuthorizationPolicy).
		MustAdd(DubboApacheOrgV1Alpha1ConditionRoute).
		MustAdd(DubboApacheOrgV1Alpha1DynamicConfig).
		MustAdd(DubboApacheOrgV1Alpha1ServiceNameMapping).
		MustAdd(DubboApacheOrgV1Alpha1TagRoute).
		Build()
)
