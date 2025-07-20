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

package schema

import (
	"fmt"
	"reflect"

	"github.com/apache/dubbo-admin/operator/pkg/config"
	"github.com/hashicorp/go-multierror"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type schemaImpl struct {
	gvk            config.GroupVersionKind
	plural         string
	clusterScoped  bool
	goPackage      string
	proto          string
	versionAliases []string
	apiVersion     string
	reflectType    reflect.Type
	statusType     reflect.Type
}

// Schema for a resource.
type Schema interface {
	fmt.Stringer
	GroupVersionResource() schema.GroupVersionResource
	GroupVersionKind() config.GroupVersionKind
	GroupVersionAliasKinds() []config.GroupVersionKind
	Validate() error
	IsClusterScoped() bool
}

func (s *schemaImpl) GroupVersionAliasKinds() []config.GroupVersionKind {
	gvks := make([]config.GroupVersionKind, len(s.versionAliases))
	for i, va := range s.versionAliases {
		gvks[i] = s.gvk
		gvks[i].Version = va
	}
	gvks = append(gvks, s.GroupVersionKind())
	return gvks
}

func (s *schemaImpl) String() string {
	return fmt.Sprintf("[Schema](%s, %q, %s)", s.Kind(), s.goPackage, s.proto)
}

func (s *schemaImpl) APIVersion() string {
	return s.apiVersion
}

func (s *schemaImpl) Kind() string {
	return s.gvk.Kind
}

func (s *schemaImpl) Group() string {
	return s.gvk.Group
}

func (s *schemaImpl) Version() string {
	return s.gvk.Version
}

func (s *schemaImpl) Plural() string {
	return s.plural
}

func (s *schemaImpl) IsClusterScoped() bool {
	return s.clusterScoped
}

func (s *schemaImpl) GroupVersionKind() config.GroupVersionKind {
	return s.gvk
}

func (s *schemaImpl) GroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    s.Group(),
		Version:  s.Version(),
		Resource: s.Plural(),
	}
}

func (s *schemaImpl) Validate() (err error) {
	if s.reflectType == nil && getProtoMessageType(s.proto) == nil {
		err = multierror.Append(err, fmt.Errorf("proto message or reflect type not found: %v", s.proto))
	}
	return
}

// Builder for a Schema.
type Builder struct {
	Identifier    string
	Plural        string
	ClusterScoped bool
	ProtoPackage  string
	Proto         string
	Kind          string
	Group         string
	Version       string
	ReflectType   reflect.Type
	StatusType    reflect.Type
	Builtin       bool
	Synthetic     bool
}

// BuildNoValidate builds the Schema without checking the fields.
func (b Builder) BuildNoValidate() Schema {
	return &schemaImpl{
		gvk: config.GroupVersionKind{
			Group:   b.Group,
			Version: b.Version,
			Kind:    b.Kind,
		},
		plural:        b.Plural,
		clusterScoped: b.ClusterScoped,
		goPackage:     b.ProtoPackage,
		proto:         b.Proto,
		apiVersion:    b.Group + "/" + b.Version,
		reflectType:   b.ReflectType,
		statusType:    b.StatusType,
	}
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

var protoMessageType = protoregistry.GlobalTypes.FindMessageByName

// getProtoMessageType returns the Go lang type of the proto with the specified name.
func getProtoMessageType(protoMessageName string) reflect.Type {
	t, err := protoMessageType(protoreflect.FullName(protoMessageName))
	if err != nil || t == nil {
		return nil
	}
	return reflect.TypeOf(t.Zero().Interface())
}
