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

package service

import (
	"reflect"
	"testing"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/console/model"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
)

func TestBuildServiceMethodSummariesIncludesOverloadsAndFallback(t *testing.T) {
	metadataList := []*meshresource.ServiceProviderMetadataResource{
		newTestServiceProviderMetadata(
			[]*meshproto.Method{
				{
					Name:           "foo",
					ParameterTypes: []string{" java.lang.String "},
					ReturnType:     " java.lang.String ",
				},
				{
					Name:           "foo",
					ParameterTypes: []string{"java.lang.Integer"},
					ReturnType:     "java.lang.String",
				},
			},
			"",
		),
		newTestServiceProviderMetadata(nil, "bar, foo"),
	}

	got := buildServiceMethodSummaries(metadataList)
	want := []model.ServiceMethodSummaryResp{
		{
			MethodName:     "bar",
			ParameterTypes: []string{},
			Signature:      "",
		},
		{
			MethodName:     "foo",
			ParameterTypes: []string{"java.lang.Integer"},
			Signature:      "java.lang.Integer->java.lang.String",
		},
		{
			MethodName:     "foo",
			ParameterTypes: []string{"java.lang.String"},
			Signature:      "java.lang.String->java.lang.String",
		},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected method summaries: got %#v, want %#v", got, want)
	}
}

func TestFindServiceMethodDetailSupportsSignatureAndFallbackStub(t *testing.T) {
	metadataList := []*meshresource.ServiceProviderMetadataResource{
		newTestServiceProviderMetadata(
			[]*meshproto.Method{
				{
					Name:           "foo",
					ParameterTypes: []string{"java.lang.String"},
					Parameters: []*meshproto.Parameter{
						{Name: "name", Type: "java.lang.String"},
					},
					ReturnType: "java.lang.String",
				},
				{
					Name:           "foo",
					ParameterTypes: []string{"java.lang.Integer"},
					ReturnType:     "java.lang.String",
				},
			},
			"",
		),
		newTestServiceProviderMetadata(nil, "bar"),
	}
	candidates := buildServiceMethodCandidates(metadataList)

	detail, ambiguous := findServiceMethodDetail(candidates, model.ServiceMethodDetailReq{
		MethodName: "foo",
	})
	if !ambiguous || detail != nil {
		t.Fatalf("expected overloaded method lookup without signature to be ambiguous, got detail=%#v ambiguous=%v", detail, ambiguous)
	}

	detail, ambiguous = findServiceMethodDetail(candidates, model.ServiceMethodDetailReq{
		MethodName: "foo",
		Signature:  "java.lang.String->java.lang.String",
	})
	if ambiguous {
		t.Fatalf("expected signature lookup to resolve overload, got ambiguous result")
	}
	if detail == nil {
		t.Fatalf("expected signature lookup to return detail")
	}
	if !reflect.DeepEqual(detail.ParameterTypes, []string{"java.lang.String"}) {
		t.Fatalf("unexpected parameter types: got %#v", detail.ParameterTypes)
	}
	if detail.ReturnType != "java.lang.String" {
		t.Fatalf("unexpected return type: got %q", detail.ReturnType)
	}
	if !reflect.DeepEqual(detail.Parameters, []model.ServiceMethodParameter{{Name: "name", Type: "java.lang.String"}}) {
		t.Fatalf("unexpected parameters: got %#v", detail.Parameters)
	}

	detail, ambiguous = findServiceMethodDetail(candidates, model.ServiceMethodDetailReq{
		MethodName: "bar",
	})
	if ambiguous {
		t.Fatalf("expected fallback-only method lookup to be unambiguous")
	}
	if detail == nil {
		t.Fatalf("expected fallback-only method lookup to return stub detail")
	}
	if len(detail.ParameterTypes) != 0 || len(detail.Parameters) != 0 || detail.ReturnType != "" {
		t.Fatalf("unexpected fallback detail: %#v", detail)
	}
}

func newTestServiceProviderMetadata(methods []*meshproto.Method, methodsParam string) *meshresource.ServiceProviderMetadataResource {
	parameters := map[string]string{}
	if methodsParam != "" {
		parameters["methods"] = methodsParam
	}
	return &meshresource.ServiceProviderMetadataResource{
		Spec: &meshproto.ServiceProviderMetadata{
			Methods:    methods,
			Parameters: parameters,
		},
	}
}
