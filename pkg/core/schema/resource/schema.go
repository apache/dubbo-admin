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

package resource

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/apache/dubbo-admin/pkg/core/labels"
	"github.com/apache/dubbo-admin/pkg/core/model"
	"github.com/apache/dubbo-admin/pkg/core/validation"
	"github.com/gogo/protobuf/proto"
	"github.com/hashicorp/go-multierror"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Schema for a resource
type Schema interface {
	fmt.Stringer

	// GroupVersionKind of the resource. This is the only way to uniquely identify a resource.
	GroupVersionKind() model.GroupVersionKind

	// GroupVersionResource of the resource.
	GroupVersionResource() schema.GroupVersionResource

	// Kind for this resource.
	Kind() string

	// Plural returns the plural form of the Kind.
	Plural() string

	IsClusterScoped() bool

	// Group for this resource.
	Group() string

	// Version of this resource.
	Version() string

	// Proto returns the protocol buffer type name for this resource.
	Proto() string

	// NewInstance returns a new instance of the protocol buffer message for this resource.
	NewInstance() (model.Spec, error)

	// MustNewInstance calls NewInstance and panics if an error occurs.
	MustNewInstance() model.Spec

	// Validate this schema.
	Validate() error
}

type Builder struct {
	// ClusterScoped is true for resource in cluster-level.
	ClusterScoped bool

	// Kind is the config proto type.
	Kind string

	// Plural is the type in plural.
	Plural string

	// Group is the config proto group.
	Group string

	// Version is the config proto version.
	Version string

	// Proto refers to the protobuf message type name corresponding to the type
	Proto string

	// ReflectType is the type of the go struct
	ReflectType reflect.Type

	// ValidateProto performs validation on protobuf messages based on this schema.
	ValidateProto validation.ValidateFunc
}

type schemaImpl struct {
	clusterScoped  bool
	gvk            model.GroupVersionKind
	plural         string
	proto          string
	validateConfig validation.ValidateFunc
	reflectType    reflect.Type
}

// Build a Schema instance.
func (b Builder) Build() (Schema, error) {
	s := b.BuildNoValidate()

	// Validate the schema.
	if err := s.Validate(); err != nil {
		return nil, err
	}

	return s, nil
}

// MustBuild calls Build and panics if it fails.
func (b Builder) MustBuild() Schema {
	s, err := b.Build()
	if err != nil {
		panic(fmt.Sprintf("MustBuild: %v", err))
	}
	return s
}

func (s *schemaImpl) MustNewInstance() model.Spec {
	p, err := s.NewInstance()
	if err != nil {
		panic(err)
	}
	return p
}

func (s *schemaImpl) GroupVersionKind() model.GroupVersionKind {
	return s.gvk
}

func (s *schemaImpl) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    s.Group(),
		Version:  s.Version(),
		Resource: s.Plural(),
	}
}

func (s *schemaImpl) IsClusterScoped() bool {
	return s.clusterScoped
}

func (s *schemaImpl) Kind() string {
	return s.gvk.Kind
}

func (s *schemaImpl) Plural() string {
	return s.plural
}

func (s *schemaImpl) Group() string {
	return s.gvk.Group
}

func (s *schemaImpl) Version() string {
	return s.gvk.Version
}

func (s *schemaImpl) Proto() string {
	return s.proto
}

func (s *schemaImpl) Validate() (err error) {
	if !labels.IsDNS1123Label(s.Kind()) {
		err = multierror.Append(err, fmt.Errorf("invalid kind: %s", s.Kind()))
	}
	if !labels.IsDNS1123Label(s.plural) {
		err = multierror.Append(err, fmt.Errorf("invalid plural for kind %s: %s", s.Kind(), s.plural))
	}
	if s.reflectType == nil && getProtoMessageType(s.proto) == nil {
		err = multierror.Append(err, fmt.Errorf("proto message or reflect type not found: %v", s.proto))
	}
	return
}

func (s *schemaImpl) String() string {
	return fmt.Sprintf("[Schema](%s, %s)", s.Kind(), s.proto)
}

func (s *schemaImpl) NewInstance() (model.Spec, error) {
	rt := s.reflectType
	if rt == nil {
		rt = getProtoMessageType(s.proto)
	}
	if rt == nil {
		return nil, errors.New("failed to find reflect type")
	}
	instance := reflect.New(rt).Interface()

	p, ok := instance.(model.Spec)
	if !ok {
		return nil, fmt.Errorf(
			"newInstance: message is not an instance of config.Spec. kind:%s, type:%v, value:%v",
			s.Kind(), rt, instance)
	}
	return p, nil
}

func (s *schemaImpl) ValidateConfig(cfg model.Config) (validation.Warning, error) {
	return s.validateConfig(cfg)
}

// BuildNoValidate builds the Schema without checking the fields.
func (b Builder) BuildNoValidate() Schema {
	if b.ValidateProto == nil {
		b.ValidateProto = validation.EmptyValidate
	}

	return &schemaImpl{
		clusterScoped: b.ClusterScoped,
		gvk: model.GroupVersionKind{
			Group:   b.Group,
			Version: b.Version,
			Kind:    b.Kind,
		},
		plural:         b.Plural,
		proto:          b.Proto,
		reflectType:    b.ReflectType,
		validateConfig: b.ValidateProto,
	}
}

// getProtoMessageType returns the Go lang type of the proto with the specified name.
func getProtoMessageType(protoMessageName string) reflect.Type {
	t := protoMessageType(protoMessageName)
	if t == nil {
		return nil
	}
	return t.Elem()
}

var protoMessageType = proto.MessageType
