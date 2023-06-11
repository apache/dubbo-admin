//
// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.6
// source: v1alpha1/ca.proto

package v1alpha1

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	structpb "google.golang.org/protobuf/types/known/structpb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type IdentityRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Csr      string           `protobuf:"bytes,1,opt,name=csr,proto3" json:"csr,omitempty"`
	Type     string           `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Metadata *structpb.Struct `protobuf:"bytes,3,opt,name=metadata,proto3" json:"metadata,omitempty"`
}

func (x *IdentityRequest) Reset() {
	*x = IdentityRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1alpha1_ca_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IdentityRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IdentityRequest) ProtoMessage() {}

func (x *IdentityRequest) ProtoReflect() protoreflect.Message {
	mi := &file_v1alpha1_ca_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IdentityRequest.ProtoReflect.Descriptor instead.
func (*IdentityRequest) Descriptor() ([]byte, []int) {
	return file_v1alpha1_ca_proto_rawDescGZIP(), []int{0}
}

func (x *IdentityRequest) GetCsr() string {
	if x != nil {
		return x.Csr
	}
	return ""
}

func (x *IdentityRequest) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *IdentityRequest) GetMetadata() *structpb.Struct {
	if x != nil {
		return x.Metadata
	}
	return nil
}

type IdentityResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Success                bool     `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
	CertPem                string   `protobuf:"bytes,2,opt,name=cert_pem,json=certPem,proto3" json:"cert_pem,omitempty"`
	TrustCerts             []string `protobuf:"bytes,3,rep,name=trust_certs,json=trustCerts,proto3" json:"trust_certs,omitempty"`
	Token                  string   `protobuf:"bytes,4,opt,name=token,proto3" json:"token,omitempty"`
	TrustedTokenPublicKeys []string `protobuf:"bytes,5,rep,name=trusted_token_public_keys,json=trustedTokenPublicKeys,proto3" json:"trusted_token_public_keys,omitempty"`
	RefreshTime            int64    `protobuf:"varint,6,opt,name=refresh_time,json=refreshTime,proto3" json:"refresh_time,omitempty"`
	ExpireTime             int64    `protobuf:"varint,7,opt,name=expire_time,json=expireTime,proto3" json:"expire_time,omitempty"`
	Message                string   `protobuf:"bytes,8,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *IdentityResponse) Reset() {
	*x = IdentityResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1alpha1_ca_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IdentityResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IdentityResponse) ProtoMessage() {}

func (x *IdentityResponse) ProtoReflect() protoreflect.Message {
	mi := &file_v1alpha1_ca_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use IdentityResponse.ProtoReflect.Descriptor instead.
func (*IdentityResponse) Descriptor() ([]byte, []int) {
	return file_v1alpha1_ca_proto_rawDescGZIP(), []int{1}
}

func (x *IdentityResponse) GetSuccess() bool {
	if x != nil {
		return x.Success
	}
	return false
}

func (x *IdentityResponse) GetCertPem() string {
	if x != nil {
		return x.CertPem
	}
	return ""
}

func (x *IdentityResponse) GetTrustCerts() []string {
	if x != nil {
		return x.TrustCerts
	}
	return nil
}

func (x *IdentityResponse) GetToken() string {
	if x != nil {
		return x.Token
	}
	return ""
}

func (x *IdentityResponse) GetTrustedTokenPublicKeys() []string {
	if x != nil {
		return x.TrustedTokenPublicKeys
	}
	return nil
}

func (x *IdentityResponse) GetRefreshTime() int64 {
	if x != nil {
		return x.RefreshTime
	}
	return 0
}

func (x *IdentityResponse) GetExpireTime() int64 {
	if x != nil {
		return x.ExpireTime
	}
	return 0
}

func (x *IdentityResponse) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type ObserveRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce string `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Type  string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
}

func (x *ObserveRequest) Reset() {
	*x = ObserveRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1alpha1_ca_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ObserveRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObserveRequest) ProtoMessage() {}

func (x *ObserveRequest) ProtoReflect() protoreflect.Message {
	mi := &file_v1alpha1_ca_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObserveRequest.ProtoReflect.Descriptor instead.
func (*ObserveRequest) Descriptor() ([]byte, []int) {
	return file_v1alpha1_ca_proto_rawDescGZIP(), []int{2}
}

func (x *ObserveRequest) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *ObserveRequest) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

type ObserveResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Nonce    string `protobuf:"bytes,1,opt,name=nonce,proto3" json:"nonce,omitempty"`
	Type     string `protobuf:"bytes,2,opt,name=type,proto3" json:"type,omitempty"`
	Revision int64  `protobuf:"varint,3,opt,name=revision,proto3" json:"revision,omitempty"`
	Data     string `protobuf:"bytes,4,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ObserveResponse) Reset() {
	*x = ObserveResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_v1alpha1_ca_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ObserveResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ObserveResponse) ProtoMessage() {}

func (x *ObserveResponse) ProtoReflect() protoreflect.Message {
	mi := &file_v1alpha1_ca_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ObserveResponse.ProtoReflect.Descriptor instead.
func (*ObserveResponse) Descriptor() ([]byte, []int) {
	return file_v1alpha1_ca_proto_rawDescGZIP(), []int{3}
}

func (x *ObserveResponse) GetNonce() string {
	if x != nil {
		return x.Nonce
	}
	return ""
}

func (x *ObserveResponse) GetType() string {
	if x != nil {
		return x.Type
	}
	return ""
}

func (x *ObserveResponse) GetRevision() int64 {
	if x != nil {
		return x.Revision
	}
	return 0
}

func (x *ObserveResponse) GetData() string {
	if x != nil {
		return x.Data
	}
	return ""
}

var File_v1alpha1_ca_proto protoreflect.FileDescriptor

var file_v1alpha1_ca_proto_rawDesc = []byte{
	0x0a, 0x11, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2f, 0x63, 0x61, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x12, 0x1e, 0x6f, 0x72, 0x67, 0x2e, 0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2e,
	0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x61, 0x75, 0x74, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70,
	0x68, 0x61, 0x31, 0x1a, 0x1c, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2f, 0x73, 0x74, 0x72, 0x75, 0x63, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x6c, 0x0a, 0x0f, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x73, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28,
	0x09, 0x52, 0x03, 0x63, 0x73, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79, 0x70, 0x65, 0x12, 0x33, 0x0a, 0x08, 0x6d, 0x65,
	0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x67,
	0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x53,
	0x74, 0x72, 0x75, 0x63, 0x74, 0x52, 0x08, 0x6d, 0x65, 0x74, 0x61, 0x64, 0x61, 0x74, 0x61, 0x22,
	0x97, 0x02, 0x0a, 0x10, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x12, 0x18, 0x0a, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x07, 0x73, 0x75, 0x63, 0x63, 0x65, 0x73, 0x73, 0x12, 0x19,
	0x0a, 0x08, 0x63, 0x65, 0x72, 0x74, 0x5f, 0x70, 0x65, 0x6d, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x63, 0x65, 0x72, 0x74, 0x50, 0x65, 0x6d, 0x12, 0x1f, 0x0a, 0x0b, 0x74, 0x72, 0x75,
	0x73, 0x74, 0x5f, 0x63, 0x65, 0x72, 0x74, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28, 0x09, 0x52, 0x0a,
	0x74, 0x72, 0x75, 0x73, 0x74, 0x43, 0x65, 0x72, 0x74, 0x73, 0x12, 0x14, 0x0a, 0x05, 0x74, 0x6f,
	0x6b, 0x65, 0x6e, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x74, 0x6f, 0x6b, 0x65, 0x6e,
	0x12, 0x39, 0x0a, 0x19, 0x74, 0x72, 0x75, 0x73, 0x74, 0x65, 0x64, 0x5f, 0x74, 0x6f, 0x6b, 0x65,
	0x6e, 0x5f, 0x70, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x5f, 0x6b, 0x65, 0x79, 0x73, 0x18, 0x05, 0x20,
	0x03, 0x28, 0x09, 0x52, 0x16, 0x74, 0x72, 0x75, 0x73, 0x74, 0x65, 0x64, 0x54, 0x6f, 0x6b, 0x65,
	0x6e, 0x50, 0x75, 0x62, 0x6c, 0x69, 0x63, 0x4b, 0x65, 0x79, 0x73, 0x12, 0x21, 0x0a, 0x0c, 0x72,
	0x65, 0x66, 0x72, 0x65, 0x73, 0x68, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x06, 0x20, 0x01, 0x28,
	0x03, 0x52, 0x0b, 0x72, 0x65, 0x66, 0x72, 0x65, 0x73, 0x68, 0x54, 0x69, 0x6d, 0x65, 0x12, 0x1f,
	0x0a, 0x0b, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x5f, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x07, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x0a, 0x65, 0x78, 0x70, 0x69, 0x72, 0x65, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x18, 0x0a, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x07, 0x6d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x22, 0x3a, 0x0a, 0x0e, 0x4f, 0x62, 0x73,
	0x65, 0x72, 0x76, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14, 0x0a, 0x05, 0x6e,
	0x6f, 0x6e, 0x63, 0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63,
	0x65, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x74, 0x79, 0x70, 0x65, 0x22, 0x6b, 0x0a, 0x0f, 0x4f, 0x62, 0x73, 0x65, 0x72, 0x76, 0x65,
	0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x6f, 0x6e, 0x63,
	0x65, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x6f, 0x6e, 0x63, 0x65, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x79, 0x70, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x79,
	0x70, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x72, 0x65, 0x76, 0x69, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x12,
	0x0a, 0x04, 0x64, 0x61, 0x74, 0x61, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x64, 0x61,
	0x74, 0x61, 0x32, 0x89, 0x01, 0x0a, 0x10, 0x41, 0x75, 0x74, 0x68, 0x6f, 0x72, 0x69, 0x74, 0x79,
	0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x75, 0x0a, 0x0e, 0x43, 0x72, 0x65, 0x61, 0x74,
	0x65, 0x49, 0x64, 0x65, 0x6e, 0x74, 0x69, 0x74, 0x79, 0x12, 0x2f, 0x2e, 0x6f, 0x72, 0x67, 0x2e,
	0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x61, 0x75, 0x74,
	0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x49, 0x64, 0x65, 0x6e, 0x74,
	0x69, 0x74, 0x79, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x30, 0x2e, 0x6f, 0x72, 0x67,
	0x2e, 0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x61, 0x75,
	0x74, 0x68, 0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x49, 0x64, 0x65, 0x6e,
	0x74, 0x69, 0x74, 0x79, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x32, 0x7f,
	0x0a, 0x0b, 0x52, 0x75, 0x6c, 0x65, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x70, 0x0a,
	0x07, 0x4f, 0x62, 0x73, 0x65, 0x72, 0x76, 0x65, 0x12, 0x2e, 0x2e, 0x6f, 0x72, 0x67, 0x2e, 0x61,
	0x70, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x61, 0x75, 0x74, 0x68,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4f, 0x62, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x2f, 0x2e, 0x6f, 0x72, 0x67, 0x2e, 0x61,
	0x70, 0x61, 0x63, 0x68, 0x65, 0x2e, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2e, 0x61, 0x75, 0x74, 0x68,
	0x2e, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x2e, 0x4f, 0x62, 0x73, 0x65, 0x72, 0x76,
	0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x42,
	0x2d, 0x50, 0x01, 0x5a, 0x29, 0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f,
	0x61, 0x70, 0x61, 0x63, 0x68, 0x65, 0x2f, 0x64, 0x75, 0x62, 0x62, 0x6f, 0x2d, 0x61, 0x64, 0x6d,
	0x69, 0x6e, 0x2f, 0x63, 0x61, 0x2f, 0x76, 0x31, 0x61, 0x6c, 0x70, 0x68, 0x61, 0x31, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_v1alpha1_ca_proto_rawDescOnce sync.Once
	file_v1alpha1_ca_proto_rawDescData = file_v1alpha1_ca_proto_rawDesc
)

func file_v1alpha1_ca_proto_rawDescGZIP() []byte {
	file_v1alpha1_ca_proto_rawDescOnce.Do(func() {
		file_v1alpha1_ca_proto_rawDescData = protoimpl.X.CompressGZIP(file_v1alpha1_ca_proto_rawDescData)
	})
	return file_v1alpha1_ca_proto_rawDescData
}

var file_v1alpha1_ca_proto_msgTypes = make([]protoimpl.MessageInfo, 4)
var file_v1alpha1_ca_proto_goTypes = []interface{}{
	(*IdentityRequest)(nil),  // 0: org.apache.dubbo.auth.v1alpha1.IdentityRequest
	(*IdentityResponse)(nil), // 1: org.apache.dubbo.auth.v1alpha1.IdentityResponse
	(*ObserveRequest)(nil),   // 2: org.apache.dubbo.auth.v1alpha1.ObserveRequest
	(*ObserveResponse)(nil),  // 3: org.apache.dubbo.auth.v1alpha1.ObserveResponse
	(*structpb.Struct)(nil),  // 4: google.protobuf.Struct
}
var file_v1alpha1_ca_proto_depIdxs = []int32{
	4, // 0: org.apache.dubbo.auth.v1alpha1.IdentityRequest.metadata:type_name -> google.protobuf.Struct
	0, // 1: org.apache.dubbo.auth.v1alpha1.AuthorityService.CreateIdentity:input_type -> org.apache.dubbo.auth.v1alpha1.IdentityRequest
	2, // 2: org.apache.dubbo.auth.v1alpha1.RuleService.Observe:input_type -> org.apache.dubbo.auth.v1alpha1.ObserveRequest
	1, // 3: org.apache.dubbo.auth.v1alpha1.AuthorityService.CreateIdentity:output_type -> org.apache.dubbo.auth.v1alpha1.IdentityResponse
	3, // 4: org.apache.dubbo.auth.v1alpha1.RuleService.Observe:output_type -> org.apache.dubbo.auth.v1alpha1.ObserveResponse
	3, // [3:5] is the sub-list for method output_type
	1, // [1:3] is the sub-list for method input_type
	1, // [1:1] is the sub-list for extension type_name
	1, // [1:1] is the sub-list for extension extendee
	0, // [0:1] is the sub-list for field type_name
}

func init() { file_v1alpha1_ca_proto_init() }
func file_v1alpha1_ca_proto_init() {
	if File_v1alpha1_ca_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_v1alpha1_ca_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IdentityRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_v1alpha1_ca_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*IdentityResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_v1alpha1_ca_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ObserveRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_v1alpha1_ca_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ObserveResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_v1alpha1_ca_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   4,
			NumExtensions: 0,
			NumServices:   2,
		},
		GoTypes:           file_v1alpha1_ca_proto_goTypes,
		DependencyIndexes: file_v1alpha1_ca_proto_depIdxs,
		MessageInfos:      file_v1alpha1_ca_proto_msgTypes,
	}.Build()
	File_v1alpha1_ca_proto = out.File
	file_v1alpha1_ca_proto_rawDesc = nil
	file_v1alpha1_ca_proto_goTypes = nil
	file_v1alpha1_ca_proto_depIdxs = nil
}
