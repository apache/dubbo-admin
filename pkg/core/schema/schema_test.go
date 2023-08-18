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
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/gomega"
)

var Authentication = collections.DubboCAV1Alpha1Authentication.Resource()

// ExpectEqual calls CheckEqual and fails the test if it returns an error.
func ExpectEqual(t *testing.T, o1 interface{}, o2 interface{}) {
	t.Helper()
	if err := CheckEqual(o1, o2); err != nil {
		t.Fatal(err)
	}
}

// CheckEqual checks that o1 and o2 are equal. If not, returns an error with the diff.
func CheckEqual(o1 interface{}, o2 interface{}) error {
	if diff := cmp.Diff(o1, o2); diff != "" {
		return fmt.Errorf(diff)
	}
	return nil
}

func TestSchema_ParseAndBuild(t *testing.T) {
	cases := []struct {
		Input    string
		Expected *Metadata
	}{
		{
			Input: `
collections:
  - name: "dubbo/apache/org/v1alpha1/AuthenticationPolicy"
    kind: "AuthenticationPolicy"
    group: "dubbo.apache.org"
    dds: true

# Configuration for resource types
resources:
  - kind: "AuthenticationPolicy"
    plural: "authenticationpolicies"
    group: "dubbo.apache.org"
    version: "v1alpha1"
    validate: "EmptyValidate"
    proto: "dubbo.apache.org.v1alpha1.AuthenticationPolicy"
`,
			Expected: &Metadata{
				collections: func() collection.Schemas {
					b := collection.NewSchemasBuilder()
					b.MustAdd(
						collection.Builder{
							Name:     "dubbo/apache/org/v1alpha1/AuthenticationPolicy",
							Resource: Authentication,
						}.MustBuild(),
					)
					return b.Build()
				}(),
			},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			g := NewWithT(t)

			_, err := ParseAndBuild(c.Input)
			g.Expect(err).To(BeNil())
		})
	}
}
