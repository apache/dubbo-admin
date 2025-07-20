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

/*
 CODE Copied and modified from https://github.com/golang/protobuf
 more specifically: https://github.com/protocolbuffers/protobuf-go/blob/master/proto/merge.go
 because of: https://github.com/golang/protobuf/issues/1359

Copyright 2010 The Go Authors.  All rights reserved.

Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are
met:

    * Redistributions of source code must retain the above copyright
notice, this list of conditions and the following disclaimer.
    * Redistributions in binary form must reproduce the above
copyright notice, this list of conditions and the following disclaimer
in the documentation and/or other materials provided with the
distribution.
    * Neither the name of Google Inc. nor the names of its
contributors may be used to endorse or promote products derived from
this software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
"AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

*/

package proto

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/durationpb"
)

type (
	MergeFunction func(dst, src protoreflect.Message)
	mergeOptions  struct {
		customMergeFn map[protoreflect.FullName]MergeFunction
	}
)
type OptionFn func(options mergeOptions) mergeOptions

func MergeFunctionOptionFn(name protoreflect.FullName, function MergeFunction) OptionFn {
	return func(options mergeOptions) mergeOptions {
		options.customMergeFn[name] = function
		return options
	}
}

// ReplaceMergeFn instead of merging all subfields one by one, takes src and set it to dest
var ReplaceMergeFn MergeFunction = func(dst, src protoreflect.Message) {
	dst.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dst.Clear(fd)
		return true
	})
	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		dst.Set(fd, v)
		return true
	})
}

func Replace(dst, src proto.Message) {
	ReplaceMergeFn(dst.ProtoReflect(), src.ProtoReflect())
}

func Merge(dst, src proto.Message) {
	duration := &durationpb.Duration{}
	merge(dst, src, MergeFunctionOptionFn(duration.ProtoReflect().Descriptor().FullName(), ReplaceMergeFn))
}

// Merge Code of proto.Merge with modifications to support custom types
func merge(dst, src proto.Message, opts ...OptionFn) {
	mo := mergeOptions{customMergeFn: map[protoreflect.FullName]MergeFunction{}}
	for _, opt := range opts {
		mo = opt(mo)
	}
	mo.mergeMessage(dst.ProtoReflect(), src.ProtoReflect())
}

func (o mergeOptions) mergeMessage(dst, src protoreflect.Message) {
	// The regular proto.mergeMessage would have a fast path method option here.
	// As we want to have exceptions we always use the slow path.
	if !dst.IsValid() {
		panic(fmt.Sprintf("cannot merge into invalid %v message", dst.Descriptor().FullName()))
	}

	src.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		switch {
		case fd.IsList():
			o.mergeList(dst.Mutable(fd).List(), v.List(), fd)
		case fd.IsMap():
			o.mergeMap(dst.Mutable(fd).Map(), v.Map(), fd.MapValue())
		case fd.Message() != nil:
			mergeFn, exists := o.customMergeFn[fd.Message().FullName()]
			if exists {
				mergeFn(dst.Mutable(fd).Message(), v.Message())
			} else {
				o.mergeMessage(dst.Mutable(fd).Message(), v.Message())
			}
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(fd, o.cloneBytes(v))
		default:
			dst.Set(fd, v)
		}
		return true
	})

	if len(src.GetUnknown()) > 0 {
		dst.SetUnknown(append(dst.GetUnknown(), src.GetUnknown()...))
	}
}

func (o mergeOptions) mergeList(dst, src protoreflect.List, fd protoreflect.FieldDescriptor) {
	// Merge semantics appends to the end of the existing list.
	for i, n := 0, src.Len(); i < n; i++ {
		switch v := src.Get(i); {
		case fd.Message() != nil:
			dstv := dst.NewElement()
			o.mergeMessage(dstv.Message(), v.Message())
			dst.Append(dstv)
		case fd.Kind() == protoreflect.BytesKind:
			dst.Append(o.cloneBytes(v))
		default:
			dst.Append(v)
		}
	}
}

func (o mergeOptions) mergeMap(dst, src protoreflect.Map, fd protoreflect.FieldDescriptor) {
	// Merge semantics replaces, rather than merges into existing entries.
	src.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		switch {
		case fd.Message() != nil:
			dstv := dst.NewValue()
			o.mergeMessage(dstv.Message(), v.Message())
			dst.Set(k, dstv)
		case fd.Kind() == protoreflect.BytesKind:
			dst.Set(k, o.cloneBytes(v))
		default:
			dst.Set(k, v)
		}
		return true
	})
}

func (o mergeOptions) cloneBytes(v protoreflect.Value) protoreflect.Value {
	return protoreflect.ValueOfBytes(append([]byte{}, v.Bytes()...))
}
