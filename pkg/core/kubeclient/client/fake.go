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

package client

import (
	fake2 "github.com/apache/dubbo-admin/pkg/core/gen/generated/clientset/versioned/fake"
	dubboinformer "github.com/apache/dubbo-admin/pkg/core/gen/generated/informers/externalversions"
	"go.uber.org/atomic"
	extfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/apimachinery/pkg/watch"
	clienttesting "k8s.io/client-go/testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes/fake"

	"k8s.io/apimachinery/pkg/runtime"
)

const resyncInterval = 0

func NewFakeClient(objects ...runtime.Object) *KubeClient {
	c := KubeClient{
		informerWatchesPending: atomic.NewInt32(0),
	}
	fakeClient := fake.NewSimpleClientset(objects...)
	c.Interface = fakeClient
	c.kubernetesClientSet = c.Interface
	c.kubeInformer = informers.NewSharedInformerFactory(c.Interface, resyncInterval)

	s := runtime.NewScheme()
	if err := metav1.AddMetaToScheme(s); err != nil {
		panic(err.Error())
	}

	dubboFake := fake2.NewSimpleClientset()
	c.dubboClientSet = dubboFake
	c.dubboInformer = dubboinformer.NewSharedInformerFactory(c.dubboClientSet, resyncInterval)
	c.extSet = extfake.NewSimpleClientset()

	listReactor := func(action clienttesting.Action) (handled bool, ret runtime.Object, err error) {
		c.informerWatchesPending.Inc()
		return false, nil, nil
	}
	watchReactor := func(tracker clienttesting.ObjectTracker) func(action clienttesting.Action) (handled bool, ret watch.Interface, err error) {
		return func(action clienttesting.Action) (handled bool, ret watch.Interface, err error) {
			gvr := action.GetResource()
			ns := action.GetNamespace()
			watch, err := tracker.Watch(gvr, ns)
			if err != nil {
				return false, nil, err
			}
			c.informerWatchesPending.Dec()
			return true, watch, nil
		}
	}
	fakeClient.PrependReactor("list", "*", listReactor)
	fakeClient.PrependWatchReactor("*", watchReactor(fakeClient.Tracker()))
	dubboFake.PrependReactor("list", "*", listReactor)
	dubboFake.PrependWatchReactor("*", watchReactor(dubboFake.Tracker()))
	c.fastSync = true

	return &c
}
