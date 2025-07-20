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

	k8sioapiextensionsapiserverpkgapisapiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	"github.com/apache/dubbo-admin/operator/pkg/schema"
	"github.com/apache/dubbo-admin/pkg/kube/collection"
)

var (
	CustomResourceDefinition = schema.Builder{
		Identifier:    "CustomResourceDefinition",
		Group:         "apiextensions.k8s.io",
		Kind:          "CustomResourceDefinition",
		Plural:        "customresourcedefinitions",
		Version:       "v1",
		Proto:         "k8s.io.apiextensions_apiserver.pkg.apis.apiextensions.v1.CustomResourceDefinition",
		ReflectType:   reflect.TypeOf(&k8sioapiextensionsapiserverpkgapisapiextensionsv1.CustomResourceDefinition{}).Elem(),
		ProtoPackage:  "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1",
		ClusterScoped: true,
		Synthetic:     false,
		Builtin:       true,
	}.MustBuild()

	All = collection.NewSchemasBuilder().
		MustAdd(CustomResourceDefinition).
		Build()
)
