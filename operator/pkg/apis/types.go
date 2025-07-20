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

package apis

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DubboOperator defines the custom API.
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubetype-gen
// +kubetype-gen:groupVersion=install.dubbo.io/v1alpha1
// +k8s:deepcopy-gen=true
type DubboOperator struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Spec defines the implementation of this definition.
	// +optional
	Spec DubboOperatorSpec `json:"spec,omitempty"`
}

// DubboOperatorSpec defines the desired installed state of Dubbo components.
// This specification is used to customize the default profile values provided with each Dubbo release.
// Since this specification is a customization API, specifying an empty DubboOperatorSpec results in the default Dubbo component values.
type DubboOperatorSpec struct {
	// Path or name for the profile.
	// default profile is used if this field is unset.
	Profile string `json:"profile,omitempty"`
	// For admin dashboard.
	Dashboard *DubboDashboardSpec `json:"dashboard,omitempty"`
	// enablement and component-specific settings that are not internal to the component.
	Components *DubboComponentSpec `json:"components,omitempty"`
	// Overrides for default `values.yaml`. This is a validated pass-through to Helm templates.
	Values json.RawMessage `json:"values,omitempty"`
}

type DubboComponentSpec struct {
	// Used for Dubbo resources.
	Base *BaseComponentSpec `json:"base,omitempty"`
	// Using Zookeeper and Nacos as the registration plane.
	Register *RegisterSpec `json:"register,omitempty"`
}

type DubboDashboardSpec struct {
	Admin *DashboardComponentSpec `json:"admin,omitempty"`
}

type RegisterSpec struct {
	// Nacos component.
	Nacos *RegisterComponentSpec `json:"nacos,omitempty"`
	// Zookeeper component.
	Zookeeper *RegisterComponentSpec `json:"zookeeper,omitempty"`
}

type BaseComponentSpec struct {
	// Selects whether this component is installed.
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type DashboardComponentSpec struct {
	// Selects whether this component is installed.
	Enabled *BoolValue `json:"enabled,omitempty"`
}

type RegisterComponentSpec struct {
	// Selects whether this component is installed.
	Enabled *BoolValue `json:"enabled,omitempty"`
	// Namespace for the component.
	Namespace string `json:"namespace,omitempty"`
	// Raw is the raw inputs. This allows distinguishing unset vs zero-values for KubernetesResources
	Raw map[string]any `json:"-"`
}

type DefaultCompSpec struct {
	RegisterComponentSpec
}

type BoolValue struct {
	bool
}

func (b *BoolValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.GetValueOrFalse())
}

func (b *BoolValue) UnmarshalJSON(bytes []byte) error {
	bb := false
	if err := json.Unmarshal(bytes, &bb); err != nil {
		return err
	}
	*b = BoolValue{bb}
	return nil
}

func (b *BoolValue) GetValueOrFalse() bool {
	if b == nil {
		return false
	}
	return b.bool
}

func (b *BoolValue) GetValueOrTrue() bool {
	if b == nil {
		return true
	}
	return b.bool
}

var (
	_ json.Unmarshaler = &BoolValue{}
	_ json.Marshaler   = &BoolValue{}
)
