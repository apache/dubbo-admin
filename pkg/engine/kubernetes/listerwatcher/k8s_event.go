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

package listerwatcher

import (
	"reflect"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	enginecfg "github.com/apache/dubbo-admin/pkg/config/engine"
	"github.com/apache/dubbo-admin/pkg/core/controller"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

type K8sEventListerWatcher struct {
	cfg *enginecfg.Config
	lw  cache.ListerWatcher
}

var _ controller.ResourceListerWatcher = &K8sEventListerWatcher{}

func NewK8sEventListWatcher(clientset *kubernetes.Clientset, cfg *enginecfg.Config) (*K8sEventListerWatcher, error) {
	// Only watch events related to Pods to avoid collecting cluster-wide noise
	// (Node events, Namespace events, etc.). The LifecycleEventSubscriber on the
	// EventBus performs further filtering using DubboAppIdentifier.
	lw := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"events",
		metav1.NamespaceAll,
		fields.ParseSelectorOrDie("involvedObject.kind=Pod"),
	)
	return &K8sEventListerWatcher{cfg: cfg, lw: lw}, nil
}

func (k *K8sEventListerWatcher) List(options metav1.ListOptions) (k8sruntime.Object, error) {
	return k.lw.List(options)
}

func (k *K8sEventListerWatcher) Watch(options metav1.ListOptions) (watch.Interface, error) {
	return k.lw.Watch(options)
}

func (k *K8sEventListerWatcher) ResourceKind() coremodel.ResourceKind {
	return meshresource.LifecycleEventKind
}

func (k *K8sEventListerWatcher) TransformFunc() cache.TransformFunc {
	return func(obj interface{}) (interface{}, error) {
		k8sEvent, ok := obj.(*v1.Event)
		if !ok {
			return nil, bizerror.NewAssertionError("v1.Event", reflect.TypeOf(obj).Name())
		}

		firstTs := ""
		if !k8sEvent.FirstTimestamp.IsZero() {
			firstTs = k8sEvent.FirstTimestamp.Format(constants.TimeFormatStr)
		}
		lastTs := ""
		if !k8sEvent.LastTimestamp.IsZero() {
			lastTs = k8sEvent.LastTimestamp.Format(constants.TimeFormatStr)
		}

		res := meshresource.NewLifecycleEventResourceWithAttributes(k8sEvent.Namespace+"/"+k8sEvent.Name, k.cfg.ID)
		res.Spec = &meshproto.LifecycleEvent{
			Namespace:       k8sEvent.Namespace,
			Reason:          k8sEvent.Reason,
			Message:         k8sEvent.Message,
			Type:            k8sEvent.Type,
			InvolvedObjKind: k8sEvent.InvolvedObject.Kind,
			InvolvedObjName: k8sEvent.InvolvedObject.Name,
			SourceComponent: k8sEvent.Source.Component,
			SourceHost:      k8sEvent.Source.Host,
			FirstTimestamp:  firstTs,
			LastTimestamp:   lastTs,
			Count:           k8sEvent.Count,
			EventSource:     "KUBERNETES",
		}
		logger.Debugf("transformed k8s event %s/%s", k8sEvent.Namespace, k8sEvent.Name)
		return res, nil
	}
}
