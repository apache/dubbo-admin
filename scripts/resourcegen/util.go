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

package main

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/apache/dubbo-admin/api/mesh"
)

// DubboResourceForMessage fetches the Dubbo resource option out of a message.
func DubboResourceForMessage(desc protoreflect.MessageDescriptor) *mesh.DubboResourceOptions {
	ext := proto.GetExtension(desc.Options(), mesh.E_Resource)
	var resOption *mesh.DubboResourceOptions
	if r, ok := ext.(*mesh.DubboResourceOptions); ok {
		resOption = r
	}

	return resOption
}

// SelectorsForMessage finds all the top-level fields in the message are
// repeated selectors. We want to generate convenience accessors for these.
func SelectorsForMessage(m protoreflect.MessageDescriptor) []string {
	var selectors []string
	fields := m.Fields()

	for i := 0; i < fields.Len(); i++ {
		field := fields.Get(i)
		m := field.Message()
		if m != nil && m.FullName() == "dubbo.mesh.v1alpha1.Selector" {
			fieldName := string(field.Name())
			caser := cases.Title(language.English)
			selectors = append(selectors, caser.String(fieldName))
		}
	}

	return selectors
}

type ResourceInfo struct {
	Name           string
	PluralName     string
	ProtoType      string
	Selectors      []string
	IsExperimental bool
}

func ToResourceInfo(desc protoreflect.MessageDescriptor) ResourceInfo {
	r := DubboResourceForMessage(desc)

	out := ResourceInfo{
		Name:           r.Name,
		PluralName:     r.PluralName,
		ProtoType:      string(desc.Name()),
		Selectors:      SelectorsForMessage(desc),
		IsExperimental: r.IsExperimental,
	}

	if p := desc.Parent(); p != nil {
		if _, ok := p.(protoreflect.MessageDescriptor); ok {
			out.ProtoType = fmt.Sprintf("%s_%s", p.Name(), desc.Name())
		}
	}
	return out
}
