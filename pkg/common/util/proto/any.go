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
	"strings"

	protov1 "github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/known/anypb"
)

const googleApis = "type.googleapis.com/"

// When saving Snapshot in SnapshotCache we generate version based on proto.Equal()
// Therefore we need deterministic way of marshaling Any which is part of the Protobuf on which we execute Equal()
//
// Based on proto.MarshalAny
func MarshalAnyDeterministic(pb proto.Message) (*anypb.Any, error) {
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(pb)
	if err != nil {
		return nil, err
	}
	name := string(protov1.MessageV2(pb).ProtoReflect().Descriptor().FullName())
	return &anypb.Any{TypeUrl: googleApis + name, Value: bytes}, nil
}

func MustMarshalAny(pb proto.Message) *anypb.Any {
	msg, err := MarshalAnyDeterministic(pb)
	if err != nil {
		panic(err.Error())
	}
	return msg
}

func UnmarshalAnyTo(src *anypb.Any, dst proto.Message) error {
	return anypb.UnmarshalTo(src, dst, proto.UnmarshalOptions{})
}

// MergeAnys merges two Any messages of the same type. We cannot just use proto#Merge on Any directly because values are encoded in byte slices.
// Instead we have to unmarshal types, merge them and marshal again.
func MergeAnys(dst *anypb.Any, src *anypb.Any) (*anypb.Any, error) {
	if src == nil {
		return dst, nil
	}
	if dst == nil {
		return src, nil
	}
	if src.TypeUrl != dst.TypeUrl {
		return nil, errors.Errorf("type URL of dst %q is different than src %q", dst.TypeUrl, src.TypeUrl)
	}

	msgType, err := FindMessageType(dst.TypeUrl)
	if err != nil {
		return nil, err
	}

	dstMsg := msgType.New().Interface()
	if err := proto.Unmarshal(dst.Value, dstMsg); err != nil {
		return nil, err
	}

	srcMsg := msgType.New().Interface()
	if err := proto.Unmarshal(src.Value, srcMsg); err != nil {
		return nil, err
	}

	Merge(dstMsg, srcMsg)
	return MarshalAnyDeterministic(dstMsg)
}

func FindMessageType(typeUrl string) (protoreflect.MessageType, error) {
	// TypeURL in Any contains type.googleapis.com/ prefix, but in Proto
	// registry it does not have this prefix.
	msgTypeName := strings.ReplaceAll(typeUrl, googleApis, "")
	fullName := protoreflect.FullName(msgTypeName)

	return protoregistry.GlobalTypes.FindMessageByName(fullName)
}
