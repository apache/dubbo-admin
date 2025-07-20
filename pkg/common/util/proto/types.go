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

package proto

import (
	"time"

	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func Now() *timestamppb.Timestamp {
	return timestamppb.Now()
}

func MustTimestampProto(t time.Time) *timestamppb.Timestamp {
	ts := timestamppb.New(t)

	if err := ts.CheckValid(); err != nil {
		panic(err.Error())
	}

	return ts
}

func MustTimestampFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}

	if err := ts.CheckValid(); err != nil {
		panic(err.Error())
	}

	t := ts.AsTime()
	return &t
}

func Bool(b bool) *wrapperspb.BoolValue {
	return &wrapperspb.BoolValue{Value: b}
}

func Bytes(b []byte) *wrapperspb.BytesValue {
	return &wrapperspb.BytesValue{Value: b}
}

func String(s string) *wrapperspb.StringValue {
	return &wrapperspb.StringValue{Value: s}
}

func UInt32(u uint32) *wrapperspb.UInt32Value {
	return &wrapperspb.UInt32Value{Value: u}
}

func UInt64(u uint64) *wrapperspb.UInt64Value {
	return &wrapperspb.UInt64Value{Value: u}
}

func Double(f float64) *wrapperspb.DoubleValue {
	return &wrapperspb.DoubleValue{Value: f}
}

func Duration(d time.Duration) *durationpb.Duration {
	return durationpb.New(d)
}

func Struct(in map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(in)
}

func MustStruct(in map[string]interface{}) *structpb.Struct {
	r, err := Struct(in)
	if err != nil {
		panic(err.Error())
	}
	return r
}

func NewValueForStruct(in interface{}) (*structpb.Value, error) {
	return structpb.NewValue(in)
}

func MustNewValueForStruct(in interface{}) *structpb.Value {
	r, err := NewValueForStruct(in)
	if err != nil {
		panic(err.Error())
	}
	return r
}
