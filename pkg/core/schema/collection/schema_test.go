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

package collection_test

import (
	"testing"

	"github.com/apache/dubbo-admin/pkg/core/schema/collection"
	"github.com/apache/dubbo-admin/pkg/core/schema/resource"

	. "github.com/onsi/gomega"
)

func TestSchema_NewSchema(t *testing.T) {
	g := NewWithT(t)

	s, err := collection.Builder{
		Name:     "foo",
		Resource: emptyResource,
	}.Build()
	g.Expect(err).To(BeNil())
	g.Expect(s.Name()).To(Equal(collection.NewName("foo")))
	g.Expect(s.Resource().Proto()).To(Equal("google.protobuf.Empty"))
}

func TestSchema_NewSchema_Error(t *testing.T) {
	g := NewWithT(t)

	_, err := collection.Builder{
		Name:     "$",
		Resource: emptyResource,
	}.Build()
	g.Expect(err).NotTo(BeNil())
}

func TestSchema_MustNewSchema(t *testing.T) {
	g := NewWithT(t)
	defer func() {
		r := recover()
		g.Expect(r).To(BeNil())
	}()

	s := collection.Builder{
		Name:     "foo",
		Resource: emptyResource,
	}.MustBuild()
	g.Expect(s.Name()).To(Equal(collection.NewName("foo")))
	g.Expect(s.Resource().Proto()).To(Equal("google.protobuf.Empty"))
}

func TestSchema_MustNewSchema_Error(t *testing.T) {
	g := NewWithT(t)
	defer func() {
		r := recover()
		g.Expect(r).NotTo(BeNil())
	}()

	collection.Builder{
		Name: "$",
		Resource: resource.Builder{
			Proto: "google.protobuf.Empty",
		}.MustBuild(),
	}.MustBuild()
}

func TestSchema_String(t *testing.T) {
	g := NewWithT(t)

	s := collection.Builder{
		Name: "foo",
		Resource: resource.Builder{
			Kind:   "Empty",
			Plural: "empties",
			Proto:  "google.protobuf.Empty",
		}.MustBuild(),
	}.MustBuild()

	g.Expect(s.String()).To(Equal(`[Schema](foo, google.protobuf.Empty)`))
}
