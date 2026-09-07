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
	"fmt"
	"regexp"
	"strings"
	"time"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/store"
)

var registryEventNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

// RegistryEventInput carries the data needed to record a registry-side event
// as a unified LifecycleEventResource.
type RegistryEventInput struct {
	Mesh         string
	Source       string // human-readable source name (e.g. "Nacos", "Zookeeper")
	SourceType   string // machine identifier (e.g. "nacos", "zookeeper")
	Category     string // "registry" | "config" | "metadata"
	Action       string // "registered" | "deregistered" | "updated" | "added" | "deleted"
	Message      string
	Type         string // "normal" | "warning"
	AppName      string
	InstanceName string
	InstanceIP   string
	ServiceName  string
}

// RecordRegistryEvent writes a registry-side lifecycle event as a unified
// LifecycleEventResource. The InvolvedObjName encodes a hierarchical identifier
// (appName/ip:port for instances, appName/serviceName for config/metadata)
// so that Console API queries can use a single HasPrefix lookup.
func RecordRegistryEvent(storeRouter store.Router, input RegistryEventInput) {
	if input.Mesh == "" || input.Message == "" {
		logger.Warnf("RecordRegistryEvent: dropping event due to empty Mesh or Message, source=%s, category=%s, action=%s, appName=%s",
			input.SourceType, input.Category, input.Action, input.AppName)
		return
	}
	eventStore, err := storeRouter.ResourceKindRoute(meshresource.LifecycleEventKind)
	if err != nil {
		logger.Errorf("route LifecycleEvent store failed, cause: %v", err)
		return
	}

	now := time.Now()

	// Build InvolvedObjName with hierarchical encoding:
	// - instance events:  {appName}/{ip}:{port}
	// - config events:    {appName}/{serviceName}:{ruleType}
	// - metadata events:  {appName}/{serviceName}
	var involvedObjName string
	switch {
	case input.InstanceIP != "":
		involvedObjName = input.AppName + "/" + input.InstanceIP
	case input.ServiceName != "":
		involvedObjName = input.AppName + "/" + input.ServiceName
	default:
		involvedObjName = input.AppName
	}

	nameSeed := strings.Join([]string{
		input.SourceType,
		input.Category,
		input.Action,
		input.AppName,
		input.ServiceName,
		input.InstanceName,
		input.InstanceIP,
	}, "-")
	sanitized := registryEventNameSanitizer.ReplaceAllString(nameSeed, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		sanitized = "event"
	}
	eventName := fmt.Sprintf("%d-%s", now.UnixNano(), sanitized)

	eventType := "normal"
	if strings.EqualFold(input.Type, "warning") {
		eventType = "warning"
	}

	source := input.Source
	if source == "" {
		source = input.SourceType
	}

	res := meshresource.NewLifecycleEventResourceWithAttributes(eventName, input.Mesh)
	res.Spec = &meshproto.LifecycleEvent{
		InvolvedObjKind: input.Category,
		InvolvedObjName: involvedObjName,
		Reason:          input.Action,
		Message:         input.Message,
		Type:            eventType,
		SourceComponent: input.SourceType,
		SourceHost:      input.InstanceIP,
		EventSource:     "REGISTRY",
		LastTimestamp:   now.Format(constants.TimeFormatStr),
		FirstTimestamp:  now.Format(constants.TimeFormatStr),
	}

	if err := eventStore.Add(res); err != nil {
		logger.Errorf("record registry event failed, key: %s, cause: %v", res.ResourceKey(), err)
	}
}
