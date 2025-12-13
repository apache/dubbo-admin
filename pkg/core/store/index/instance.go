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

	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

const (
	ByInstanceAppNameIndex = "idx_instance_app_name"
	ByInstanceIpIndex      = "idx_instance_ip"
	ByInstanceNameIndex    = "idx_instance_name"
)

func init() {
	RegisterIndexers(meshresource.InstanceKind, map[string]cache.IndexFunc{
		ByInstanceAppNameIndex: byInstanceAppName,
		ByInstanceIpIndex:      byInstanceIp,
		ByInstanceNameIndex:    byInstanceName,
	})
}

func byInstanceAppName(obj interface{}) ([]string, error) {
	instance, ok := obj.(*meshresource.InstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.InstanceKind, reflect.TypeOf(obj).Name())
	}
	if instance.Spec == nil || strutil.IsBlank(instance.Spec.AppName) {
		return []string{}, nil
	}
	return []string{instance.Spec.AppName}, nil
}

func byInstanceIp(obj interface{}) ([]string, error) {
	instance, ok := obj.(*meshresource.InstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.InstanceKind, reflect.TypeOf(obj).Name())
	}
	if instance.Spec == nil || strutil.IsBlank(instance.Spec.Ip) {
		return []string{}, nil
	}
	return []string{instance.Spec.Ip}, nil
}

func byInstanceName(obj interface{}) ([]string, error) {
	instance, ok := obj.(*meshresource.InstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.InstanceKind, reflect.TypeOf(obj).Name())
	}
	if instance.Spec == nil || strutil.IsBlank(instance.Spec.Name) {
		return []string{}, nil
	}
	return []string{instance.Spec.Name}, nil
}
