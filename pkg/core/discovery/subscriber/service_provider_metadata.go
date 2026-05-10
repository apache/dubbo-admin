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
	"sort"
	"strings"

	"k8s.io/client-go/tools/cache"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/core/events"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

type ServiceProviderMetadataEventSubscriber struct {
	appStore      store.ResourceStore
	serviceStore  store.ResourceStore
	providerStore store.ResourceStore
	emitter       events.Emitter
}

func NewServiceProviderMetadataEventSubscriber(
	appStore store.ResourceStore,
	serviceStore store.ResourceStore,
	providerStore store.ResourceStore,
	emitter events.Emitter) *ServiceProviderMetadataEventSubscriber {
	return &ServiceProviderMetadataEventSubscriber{
		appStore:      appStore,
		serviceStore:  serviceStore,
		providerStore: providerStore,
		emitter:       emitter,
	}
}

func (s *ServiceProviderMetadataEventSubscriber) ResourceKind() coremodel.ResourceKind {
	return meshresource.ServiceProviderMetadataKind
}

func (s *ServiceProviderMetadataEventSubscriber) Name() string {
	return "Discovery-" + s.ResourceKind().ToString()
}

func (s *ServiceProviderMetadataEventSubscriber) AsyncEnabled() bool {
	return true
}

func (s *ServiceProviderMetadataEventSubscriber) ProcessEvent(event events.Event) error {
	newObj, ok := event.NewObj().(*meshresource.ServiceProviderMetadataResource)
	if !ok && event.NewObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(newObj), event.NewObj())
	}
	oldObj, ok := event.OldObj().(*meshresource.ServiceProviderMetadataResource)
	if !ok && event.OldObj() != nil {
		return bizerror.NewAssertionError(reflect.TypeOf(oldObj), event.OldObj())
	}

	var processErr error
	switch event.Type() {
	case cache.Added, cache.Replaced, cache.Sync:
		if newObj == nil {
			errStr := "process provider metadata resource upsert event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processUpsert(newObj)
	case cache.Updated:
		if newObj == nil {
			errStr := "process provider metadata resource update event, but new obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processUpdate(newObj)
	case cache.Deleted:
		if oldObj == nil {
			errStr := "process provider metadata resource delete event, but old obj is nil, skipped processing"
			logger.Errorf(errStr)
			return bizerror.New(bizerror.EventError, errStr)
		}
		processErr = s.processDelete(oldObj)
	}
	if processErr != nil {
		logger.Errorf("process provider metadata resource event failed, cause: %s, event: %s", processErr.Error(), event.String())
		return processErr
	}
	logger.Infof("process provider metadata resource event successfully, event: %s", event.String())
	return nil
}

func (s *ServiceProviderMetadataEventSubscriber) processUpsert(r *meshresource.ServiceProviderMetadataResource) error {
	if r.Spec == nil {
		return bizerror.New(bizerror.UnknownError, "provider metadata resource spec is nil")
	}
	if err := s.ensureApplication(r); err != nil {
		return err
	}
	return s.syncService(r.Mesh, r.Spec.ServiceName, r.Spec.Version, r.Spec.Group)
}

func (s *ServiceProviderMetadataEventSubscriber) processDelete(r *meshresource.ServiceProviderMetadataResource) error {
	if r.Spec == nil {
		return bizerror.New(bizerror.UnknownError, "provider metadata resource spec is nil")
	}
	return s.syncService(r.Mesh, r.Spec.ServiceName, r.Spec.Version, r.Spec.Group)
}

func (s *ServiceProviderMetadataEventSubscriber) processUpdate(newRes *meshresource.ServiceProviderMetadataResource) error {
	if newRes.Spec == nil {
		return bizerror.New(bizerror.UnknownError, "provider metadata resource spec is nil")
	}
	if err := s.ensureApplication(newRes); err != nil {
		return err
	}
	return s.syncService(newRes.Mesh, newRes.Spec.ServiceName, newRes.Spec.Version, newRes.Spec.Group)
}

func (s *ServiceProviderMetadataEventSubscriber) ensureApplication(r *meshresource.ServiceProviderMetadataResource) error {
	if r.Spec.ProviderAppName == "" {
		logger.Warnf("skip processing application sync because spec.providerAppName is blank, res:%s", r.String())
		return nil
	}
	_, exists, err := s.appStore.GetByKey(coremodel.BuildResourceKey(r.Mesh, r.Spec.ProviderAppName))
	if err != nil {
		logger.Errorf("get application resource failed, appName: %s, mesh: %s, cause: %s",
			r.Spec.ProviderAppName, r.Mesh, err.Error())
		return err
	}
	if exists {
		return nil
	}

	appRes := meshresource.NewApplicationResourceWithAttributes(r.Spec.ProviderAppName, r.Mesh)
	appRes.Spec.Name = r.Spec.ProviderAppName
	if err := s.appStore.Add(appRes); err != nil {
		logger.Errorf("add application resource failed, appName: %s, mesh: %s, cause: %s",
			r.Spec.ProviderAppName, r.Mesh, err.Error())
		return err
	}
	s.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, appRes))
	return nil
}

func (s *ServiceProviderMetadataEventSubscriber) syncService(mesh, serviceName, version, group string) error {
	serviceKey := meshresource.BuildServiceIdentityKey(serviceName, version, group)
	resources, err := s.providerStore.ListByIndexes(
		[]index.IndexCondition{{IndexName: index.ByMeshIndex, Value: mesh, Operator: index.Equals},
			{IndexName: index.ByServiceProviderServiceKey, Value: serviceKey, Operator: index.Equals},
		})
	if err != nil {
		return err
	}

	providers := make([]*meshresource.ServiceProviderMetadataResource, 0, len(resources))
	for _, item := range resources {
		res, ok := item.(*meshresource.ServiceProviderMetadataResource)
		if !ok {
			return bizerror.NewAssertionError(meshresource.ServiceProviderMetadataKind, reflect.TypeOf(item).Name())
		}
		providers = append(providers, res)
	}

	rawOldRes, exists, err := s.serviceStore.GetByKey(coremodel.BuildResourceKey(mesh, serviceKey))
	if err != nil {
		return err
	}

	var oldRes *meshresource.ServiceResource
	if exists {
		var ok bool
		oldRes, ok = rawOldRes.(*meshresource.ServiceResource)
		if !ok {
			return bizerror.NewAssertionError(meshresource.ServiceKind, reflect.TypeOf(rawOldRes).Name())
		}
	}

	if len(providers) == 0 {
		if !exists {
			return nil
		}
		if err := s.serviceStore.Delete(oldRes); err != nil {
			return err
		}
		s.emitter.Send(events.NewResourceChangedEvent(cache.Deleted, oldRes, nil))
		return nil
	}

	newRes := meshresource.NewServiceResourceWithAttributes(serviceKey, mesh)
	newRes.Spec = buildServiceSpec(serviceName, version, group, providers)
	if !exists {
		if err := s.serviceStore.Add(newRes); err != nil {
			return err
		}
		s.emitter.Send(events.NewResourceChangedEvent(cache.Added, nil, newRes))
		return nil
	}

	if err := s.serviceStore.Update(newRes); err != nil {
		return err
	}
	s.emitter.Send(events.NewResourceChangedEvent(cache.Updated, oldRes, newRes))
	return nil
}

func buildServiceSpec(serviceName, version, group string, providers []*meshresource.ServiceProviderMetadataResource) *meshproto.Service {
	methodSet := make(map[string]struct{})
	language := ""

	for _, provider := range providers {
		if provider.Spec == nil {
			continue
		}
		if language == "" {
			language = inferProviderLanguage(provider.Spec)
		}
		for _, method := range provider.Spec.Methods {
			if method == nil || method.Name == "" {
				continue
			}
			methodSet[method.Name] = struct{}{}
		}
	}

	methods := make([]string, 0, len(methodSet))
	for methodName := range methodSet {
		methods = append(methods, methodName)
	}
	sort.Strings(methods)

	return &meshproto.Service{
		Name:     serviceName,
		Group:    group,
		Version:  version,
		Language: language,
		Methods:  methods,
	}
}

func inferProviderLanguage(spec *meshproto.ServiceProviderMetadata) string {
	if spec == nil {
		return ""
	}
	if spec.Parameters != nil && spec.Parameters["language"] != "" {
		return spec.Parameters["language"]
	}
	// Current SDKs do not reliably publish an explicit language field, so we
	// fall back to SDK-specific metadata fingerprints when the source field is absent.
	if looksLikeDubboGo(spec) {
		return "golang"
	}
	if looksLikeDubboJava(spec) {
		return "java"
	}
	return ""
}

func looksLikeDubboGo(spec *meshproto.ServiceProviderMetadata) bool {
	if spec == nil || spec.Parameters == nil {
		return false
	}
	// dubbo-go writes a release like "dubbo-golang-<version>", which is the
	// most stable discriminator we can use without requiring provider changes.
	release := strings.ToLower(spec.Parameters["release"])
	return strings.HasPrefix(release, "dubbo-golang-") || strings.HasPrefix(release, "dubbo-go-")
}

func looksLikeDubboJava(spec *meshproto.ServiceProviderMetadata) bool {
	if spec == nil {
		return false
	}
	// Java providers usually expose Java type names in method signatures and
	// exported type definitions, even when "language" itself is missing.
	for _, method := range spec.Methods {
		if methodHasJavaTypeHint(method) {
			return true
		}
	}
	for _, typ := range spec.Types {
		if metadataTypeHasJavaHint(typ) {
			return true
		}
	}
	return false
}

func methodHasJavaTypeHint(method *meshproto.Method) bool {
	if method == nil {
		return false
	}
	for _, parameterType := range method.ParameterTypes {
		if isJavaTypeHint(parameterType) {
			return true
		}
	}
	if isJavaTypeHint(method.ReturnType) {
		return true
	}
	for _, parameter := range method.Parameters {
		if parameter != nil && isJavaTypeHint(parameter.Type) {
			return true
		}
	}
	return false
}

func metadataTypeHasJavaHint(typ *meshproto.Type) bool {
	if typ == nil {
		return false
	}
	if isJavaTypeHint(typ.Type) {
		return true
	}
	for _, item := range typ.Items {
		if isJavaTypeHint(item) {
			return true
		}
	}
	for _, propertyType := range typ.Properties {
		if isJavaTypeHint(propertyType) {
			return true
		}
	}
	return false
}

func isJavaTypeHint(typeName string) bool {
	typeName = strings.ToLower(typeName)
	return strings.HasPrefix(typeName, "java.") || strings.Contains(typeName, ".java.")
}
