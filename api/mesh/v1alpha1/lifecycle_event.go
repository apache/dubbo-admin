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

package v1alpha1

// LifecycleEvent is the spec for a K8s event resource, capturing both K8s native events
// and registry-side lifecycle events in a unified format.
type LifecycleEvent struct {
	Namespace       string `json:"namespace,omitempty"`
	Reason          string `json:"reason,omitempty"`
	Message         string `json:"message,omitempty"`
	Type            string `json:"type,omitempty"`
	InvolvedObjKind string `json:"involvedObjKind,omitempty"`
	InvolvedObjName string `json:"involvedObjName,omitempty"`
	SourceComponent string `json:"sourceComponent,omitempty"`
	SourceHost      string `json:"sourceHost,omitempty"`
	FirstTimestamp  string `json:"firstTimestamp,omitempty"`
	LastTimestamp   string `json:"lastTimestamp,omitempty"`
	Count           int32  `json:"count,omitempty"`
	EventSource     string `json:"eventSource,omitempty"`
}

func (e *LifecycleEvent) Clone() *LifecycleEvent {
	if e == nil {
		return nil
	}
	return &LifecycleEvent{
		Namespace:       e.Namespace,
		Reason:          e.Reason,
		Message:         e.Message,
		Type:            e.Type,
		InvolvedObjKind: e.InvolvedObjKind,
		InvolvedObjName: e.InvolvedObjName,
		SourceComponent: e.SourceComponent,
		SourceHost:      e.SourceHost,
		FirstTimestamp:  e.FirstTimestamp,
		LastTimestamp:   e.LastTimestamp,
		Count:           e.Count,
		EventSource:     e.EventSource,
	}
}
