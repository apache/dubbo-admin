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
	ByRuntimeInstanceIPIndex   = "idx_rt_instance_ip"
	ByRuntimeInstanceNameIndex = "idx_rt_instance_name"
)

func init() {
	RegisterIndexers(meshresource.RuntimeInstanceKind, map[string]cache.IndexFunc{
		ByRuntimeInstanceIPIndex:   byRuntimeInstanceIp,
		ByRuntimeInstanceNameIndex: byRuntimeInstanceName,
	})
}

func byRuntimeInstanceIp(obj interface{}) ([]string, error) {
	rtInstance, ok := obj.(*meshresource.RuntimeInstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(obj).Name())
	}
	if rtInstance.Spec == nil || strutil.IsBlank(rtInstance.Spec.Ip) {
		return []string{}, nil
	}
	return []string{rtInstance.Spec.Ip}, nil
}

func byRuntimeInstanceName(obj interface{}) ([]string, error) {
	rtInstance, ok := obj.(*meshresource.RuntimeInstanceResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.RuntimeInstanceKind, reflect.TypeOf(obj).Name())
	}
	if rtInstance.Spec == nil || rtInstance.Spec.Name == "" {
		return []string{}, nil
	}
	return []string{rtInstance.Spec.Name}, nil
}
