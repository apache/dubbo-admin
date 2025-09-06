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
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
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
	ResourceName             string
	ResourceType             string
	ProtoType                string
	Selectors                []string
	SkipRegistration         bool
	SkipKubernetesWrappers   bool
	ScopeNamespace           bool
	Global                   bool
	DubboctlSingular         string
	DubboctlPlural           string
	WsReadOnly               bool
	WsAdminOnly              bool
	WsPath                   string
	DdsDirection             string
	AllowToInspect           bool
	StorageVersion           bool
	IsPolicy                 bool
	SingularDisplayName      string
	PluralDisplayName        string
	IsExperimental           bool
	AdditionalPrinterColumns []string
	HasInsights              bool
}

func ToResourceInfo(desc protoreflect.MessageDescriptor) ResourceInfo {
	r := DubboResourceForMessage(desc)

	out := ResourceInfo{
		ResourceType:             r.Type,
		ResourceName:             r.Name,
		ProtoType:                string(desc.Name()),
		Selectors:                SelectorsForMessage(desc),
		SkipRegistration:         r.SkipRegistration,
		SkipKubernetesWrappers:   r.SkipKubernetesWrappers,
		Global:                   r.Global,
		ScopeNamespace:           r.ScopeNamespace,
		AllowToInspect:           r.AllowToInspect,
		StorageVersion:           r.StorageVersion,
		SingularDisplayName:      coremodel.DisplayName(r.Type),
		PluralDisplayName:        r.PluralDisplayName,
		IsExperimental:           r.IsExperimental,
		AdditionalPrinterColumns: r.AdditionalPrinterColumns,
		HasInsights:              r.HasInsights,
	}
	if r.Ws != nil {
		pluralResourceName := r.Ws.Plural
		if pluralResourceName == "" {
			pluralResourceName = r.Ws.Name + "s"
		}
		out.WsReadOnly = r.Ws.ReadOnly
		out.WsAdminOnly = r.Ws.AdminOnly
		out.WsPath = pluralResourceName
		if !r.Ws.ReadOnly {
			out.DubboctlSingular = r.Ws.Name
			out.DubboctlPlural = pluralResourceName
			// Keep the typo to preserve backward compatibility
			if out.DubboctlSingular == "health-check" {
				out.DubboctlSingular = "healthcheck"
				out.DubboctlPlural = "healthchecks"
			}
		}
	}
	if out.PluralDisplayName == "" {
		out.PluralDisplayName = coremodel.PluralType(coremodel.DisplayName(r.Type))
	}
	// Working around the fact we don't really differentiate policies from the rest of resources:
	// Anything global can't be a policy as it need to be on a mesh. Anything with locked Ws config is something internal and therefore not a policy
	out.IsPolicy = !out.SkipRegistration && !out.Global && !out.WsAdminOnly && !out.WsReadOnly && out.ResourceType != "Dataplane" && out.ResourceType != "ExternalService"
	switch {
	case r.Dds == nil || (!r.Dds.SendToZone && !r.Dds.SendToGlobal):
		out.DdsDirection = ""
	case r.Dds.SendToGlobal && r.Dds.SendToZone:
		out.DdsDirection = "model.ZoneToGlobalFlag | model.GlobalToAllButOriginalZoneFlag"
	case r.Dds.SendToGlobal:
		out.DdsDirection = "model.ZoneToGlobalFlag"
	case r.Dds.SendToZone:
		out.DdsDirection = "model.GlobalToAllZonesFlag"
	}

	if out.ResourceType == "MeshGateway" {
		out.DdsDirection = "model.ZoneToGlobalFlag | model.GlobalToAllZonesFlag"
	}

	if p := desc.Parent(); p != nil {
		if _, ok := p.(protoreflect.MessageDescriptor); ok {
			out.ProtoType = fmt.Sprintf("%s_%s", p.Name(), desc.Name())
		}
	}
	return out
}
