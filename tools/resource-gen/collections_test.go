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
	"strings"
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/schema/ast"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
)

func TestStaticCollections(t *testing.T) {
	cases := []struct {
		packageName string
		m           *ast.Metadata
		err         string
		output      string
	}{
		{
			packageName: "pkg",
			m: &ast.Metadata{
				Collections: []*ast.Collection{
					{
						Name:         "foo",
						VariableName: "Foo",
						Group:        "foo.group",
						Kind:         "fookind",
					},
					{
						Name:         "bar",
						VariableName: "Bar",
						Group:        "bar.group",
						Kind:         "barkind",
					},
				},
				Resources: []*ast.Resource{
					{
						Group:         "foo.group",
						Version:       "v1",
						Kind:          "fookind",
						Plural:        "fookinds",
						ClusterScoped: true,
						Proto:         "google.protobuf.Struct",
						Validate:      "EmptyValidate",
					},
					{
						Group:         "bar.group",
						Version:       "v1",
						Kind:          "barkind",
						Plural:        "barkinds",
						ClusterScoped: false,
						Proto:         "google.protobuf.Struct",
						Validate:      "EmptyValidate",
					},
				},
			},
			output: `
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

package collections

import (
	"reflect"

	api "github.com/apache/dubbo-admin/api/resource/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"
	"github.com/apache/dubbo-admin/pkg/core/validation"
)

var (

	Bar = collection.Builder {
		Name: "bar",
		VariableName: "Bar",
		Resource: resource.Builder {
			Group: "bar.group",
			Kind: "barkind",
			Plural: "barkinds",
			Version: "v1",
			Proto: "google.protobuf.Struct",
			ReflectType: reflect.TypeOf(&api.Struct{}).Elem(),
			ClusterScoped: false,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()

	Foo = collection.Builder {
		Name: "foo",
		VariableName: "Foo",
		Resource: resource.Builder {
			Group: "foo.group",
			Kind: "fookind",
			Plural: "fookinds",
			Version: "v1",
			Proto: "google.protobuf.Struct",
			ReflectType: reflect.TypeOf(&api.Struct{}).Elem(),
			ClusterScoped: true,
			ValidateProto: validation.EmptyValidate,
		}.MustBuild(),
	}.MustBuild()


	Rule = collection.NewSchemasBuilder().
		Build()
)

`,
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			g := NewWithT(t)

			s, err := StaticCollections(c.m)
			if c.err != "" {
				g.Expect(err).NotTo(BeNil())
				g.Expect(err.Error()).To(Equal(s))
			} else {
				g.Expect(err).To(BeNil())
				if diff := cmp.Diff(strings.TrimSpace(s), strings.TrimSpace(c.output)); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
