/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kube

import (
	"fmt"
	"github.com/apache/dubbo-admin/pkg/logger"
	crdV1alpha1 "github.com/apache/dubbo-admin/pkg/mapping/apis/dubbo.apache.org/v1alpha1"
	informerV1alpha1 "github.com/apache/dubbo-admin/pkg/mapping/generated/informers/externalversions/dubbo.apache.org/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"time"
)

type Controller struct {
	informer       informerV1alpha1.ServiceNameMappingInformer
	workqueue      workqueue.RateLimitingInterface
	informerSynced cache.InformerSynced
}

func NewController(spInformer informerV1alpha1.ServiceNameMappingInformer) *Controller {
	controller := &Controller{
		informer:  spInformer,
		workqueue: workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "ServiceMappings"),
	}
	logger.Sugar().Info("Setting up service mappings event handlers")
	_, err := spInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueServiceMapping,
		UpdateFunc: func(old, new interface{}) {
			oldObj := old.(*crdV1alpha1.ServiceNameMapping)
			newObj := new.(*crdV1alpha1.ServiceNameMapping)
			if oldObj.ResourceVersion == newObj.ResourceVersion {
				return
			}
			controller.enqueueServiceMapping(new)
		},
		DeleteFunc: controller.enqueueServiceMappingForDelete,
	})
	if err != nil {
		return nil
	}
	return controller
}

func (c *Controller) Process() bool {
	obj, shutdown := c.workqueue.Get()
	if shutdown {
		return false
	}
	err := func(obj interface{}) error {
		defer c.workqueue.Done(obj)
		var key string
		var ok bool
		if key, ok = obj.(string); !ok {
			c.workqueue.Forget(obj)
			runtime.HandleError(fmt.Errorf("expected string in workqueue but got %#v", obj))
			return nil
		}
		if err := c.syncHandler(key); err != nil {
			logger.Sugar().Error("error syncing '%s': %s", key, err.Error())
			return nil
		}
		c.workqueue.Forget(obj)
		logger.Sugar().Infof("Successfully synced %s", key)
		return nil
	}(obj)

	if err != nil {
		runtime.HandleError(err)
		return true
	}
	return true
}
func (c *Controller) PreRun(thread int, stopCh <-chan struct{}) error {
	defer runtime.HandleCrash()
	defer c.workqueue.ShutDown()

	logger.Sugar().Info("Starting Service Mapping control loop")
	logger.Sugar().Info("Waiting for informer caches to sync")
	logger.Sugar().Info("Starting sync")
	for i := 0; i < thread; i++ {
		go wait.Until(c.Run, time.Second, stopCh)
	}
	logger.Sugar().Info("Started sync")
	<-stopCh
	logger.Sugar().Info("Shutting down")
	return nil
}

func (c *Controller) Run() {
	for c.Process() {
	}
}

func (c *Controller) syncHandler(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		runtime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	sp, err := c.informer.Lister().ServiceNameMappings(namespace).Get(name)

	if err != nil {
		if errors.IsNotFound(err) {
			logger.Sugar().Warnf("[ServiceMappingsCRD] %s/%s does not exist in local cache, will delete it from service mapping ...",
				namespace, name)
			logger.Sugar().Infof("[ServiceMappingsCRD] deleting service mapping: %s/%s ...", namespace, name)
			return nil
		}
		runtime.HandleError(fmt.Errorf("failed to get service mapping by: %s/%s", namespace, name))
		return err
	}
	logger.Sugar().Infof("[ServiceMappingsCRD] Trying to handle service mapping: %#v ...", sp)
	return nil
}

func (c *Controller) enqueueServiceMapping(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}

func (c *Controller) enqueueServiceMappingForDelete(obj interface{}) {
	var key string
	var err error
	key, err = cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		runtime.HandleError(err)
		return
	}
	c.workqueue.AddRateLimited(key)
}
