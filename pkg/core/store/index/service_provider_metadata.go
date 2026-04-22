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

package index

import (
	"reflect"

	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

const (
	ByServiceProviderAppName     = "idx_service_provider_app_name"
	ByServiceProviderServiceName = "idx_service_provider_service_name"
)

func init() {
	RegisterIndexers(meshresource.ServiceProviderMetadataKind, map[string]cache.IndexFunc{
		ByServiceProviderAppName:     byServiceProviderAppName,
		ByServiceProviderServiceName: byServiceProviderServiceName,
	})
}

func byServiceProviderAppName(obj interface{}) ([]string, error) {
	metadata, ok := obj.(*meshresource.ServiceProviderMetadataResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.ServiceProviderMetadataKind, reflect.TypeOf(obj).Name())
	}
	if metadata.Spec == nil {
		return []string{}, nil
	}
	return []string{metadata.Spec.ProviderAppName}, nil
}

func byServiceProviderServiceName(obj interface{}) ([]string, error) {
	metadata, ok := obj.(*meshresource.ServiceProviderMetadataResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.ServiceProviderMetadataKind, reflect.TypeOf(obj).Name())
	}
	if metadata.Spec == nil {
		return []string{}, nil
	}
	return []string{metadata.Spec.ServiceName}, nil
}
