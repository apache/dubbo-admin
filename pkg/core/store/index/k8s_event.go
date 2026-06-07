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
	ByK8sEventInvolvedObjKind = "idx_k8s_event_involved_obj_kind"
	ByK8sEventInvolvedObjName = "idx_k8s_event_involved_obj_name"
	ByK8sEventType            = "idx_k8s_event_type"
	ByK8sEventSource          = "idx_k8s_event_source"
)

func init() {
	RegisterIndexers(meshresource.K8sEventKind, map[string]cache.IndexFunc{
		ByK8sEventInvolvedObjKind: byK8sEventInvolvedObjKind,
		ByK8sEventInvolvedObjName: byK8sEventInvolvedObjName,
		ByK8sEventType:            byK8sEventType,
		ByK8sEventSource:          byK8sEventSource,
	})
}

func byK8sEventInvolvedObjKind(obj interface{}) ([]string, error) {
	event, ok := obj.(*meshresource.K8sEventResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.K8sEventKind, reflect.TypeOf(obj).Name())
	}
	if event.Spec == nil || event.Spec.InvolvedObjKind == "" {
		return []string{}, nil
	}
	return []string{event.Spec.InvolvedObjKind}, nil
}

func byK8sEventInvolvedObjName(obj interface{}) ([]string, error) {
	event, ok := obj.(*meshresource.K8sEventResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.K8sEventKind, reflect.TypeOf(obj).Name())
	}
	if event.Spec == nil || event.Spec.InvolvedObjName == "" {
		return []string{}, nil
	}
	return []string{event.Spec.InvolvedObjName}, nil
}

func byK8sEventType(obj interface{}) ([]string, error) {
	event, ok := obj.(*meshresource.K8sEventResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.K8sEventKind, reflect.TypeOf(obj).Name())
	}
	if event.Spec == nil || event.Spec.Type == "" {
		return []string{}, nil
	}
	return []string{event.Spec.Type}, nil
}

func byK8sEventSource(obj interface{}) ([]string, error) {
	event, ok := obj.(*meshresource.K8sEventResource)
	if !ok {
		return nil, bizerror.NewAssertionError(meshresource.K8sEventKind, reflect.TypeOf(obj).Name())
	}
	if event.Spec == nil || event.Spec.EventSource == "" {
		return []string{}, nil
	}
	return []string{event.Spec.EventSource}, nil
}
