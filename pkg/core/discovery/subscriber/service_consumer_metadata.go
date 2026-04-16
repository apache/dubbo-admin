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

package subscriber

import (
	"reflect"

	"github.com/duke-git/lancet/v2/strutil"
	"k8s.io/client-go/tools/cache"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

type ServiceConsumerMetadataEventSubscriber struct {
	appStore store.ResourceStore
	emitter  events.Emitter
}

func NewServiceConsumerMetadataEventSubscriber(
	appStore store.ResourceStore,
	emitter events.Emitter) *ServiceConsumerMetadataEventSubscriber {
	return &ServiceConsumerMetadataEventSubscriber{
		appStore: appStore,
		emitter:  emitter,
	}
}

func (s *ServiceConsumerMetadataEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceConsumerMetadataKind
}

func (s *ServiceConsumerMetadataEventSubscriber) Name() string {
	return "Discovery-" + s.ResourceKind().ToString()
}

func (s *ServiceConsumerMetadataEventSubscriber) AsyncEnabled() bool {
	return true
}

func (s *ServiceConsumerMetadataEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.ServiceConsumerMetadataResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(newObj), event.NewObj())
	}
	var processErr error
	switch event.Type() {
	case cache.Added, cache.Updated, cache.Replaced, cache.Sync:
		if newObj == nil {
			errStr := "process consumer metadata resource upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processUpsert(newObj)
	case cache.Deleted:
		logger.Warnf("ignored consumer metadata resource deleted event in ServiceConsumerMetadataEventSubscriber")
	}
	if processErr != nil {
		logger.Errorf("process consumer metadata resource event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process consumer metadata resource event successfully, event: %s", event.String())
	return nil
}

func (s *ServiceConsumerMetadataEventSubscriber) processUpsert(r *meshresource.ServiceConsumerMetadataResource) error {
	if r.Spec == nil {
		return bizerror.New(bizerror.UnknownError, "consumer metadata resource spec is nil")
	}
	if strutil.IsBlank(r.Spec.ConsumerAppName) {
		logger.Warnf("skip processing service consumer metadata event because spec.consumerAppName is blank, res:%s", r.String())
		return nil
	}
	_, exists, err := s.appStore.GetByKey(coremodel.BuildResourceKey(r.Mesh, r.Spec.ConsumerAppName))
	if err != nil {
		logger.Errorf("get application resource failed, appName: %s, mesh: %s, cause: %s",
			r.Spec.ConsumerAppName, r.Mesh, err.Error())
		return err
	}
	if exists {
		logger.Infof("application resource already exists, appName: %s, mesh: %s", r.Spec.ConsumerAppName, r.Mesh)
		return nil
	}
	appRes := meshresource.NewApplicationResourceWithAttributes(r.Spec.ConsumerAppName, r.Mesh)
	appRes.Spec.Name = r.Spec.ConsumerAppName
	if err := s.appStore.Add(appRes); err != nil {
		logger.Errorf("add application resource failed, appName: %s, mesh: %s, cause: %s",
			r.Spec.ConsumerAppName, r.Mesh, err.Error())
		return err
	}
	s.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, appRes))
	return nil
}
