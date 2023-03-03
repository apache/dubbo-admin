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

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.21.6
// source: v1alpha1/ca.proto

package v1alpha1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// DubboCertificateServiceClient is the client API for DubboCertificateService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type DubboCertificateServiceClient interface {
	CreateCertificate(ctx context.Context, in *DubboCertificateRequest, opts ...grpc.CallOption) (*DubboCertificateResponse, error)
}

type dubboCertificateServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewDubboCertificateServiceClient(cc grpc.ClientConnInterface) DubboCertificateServiceClient {
	return &dubboCertificateServiceClient{cc}
}

func (c *dubboCertificateServiceClient) CreateCertificate(ctx context.Context, in *DubboCertificateRequest, opts ...grpc.CallOption) (*DubboCertificateResponse, error) {
	out := new(DubboCertificateResponse)
	err := c.cc.Invoke(ctx, "/org.apache.dubbo.auth.v1alpha1.DubboCertificateService/CreateCertificate", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// DubboCertificateServiceServer is the server API for DubboCertificateService service.
// All implementations must embed UnimplementedDubboCertificateServiceServer
// for forward compatibility
type DubboCertificateServiceServer interface {
	CreateCertificate(context.Context, *DubboCertificateRequest) (*DubboCertificateResponse, error)
	mustEmbedUnimplementedDubboCertificateServiceServer()
}

// UnimplementedDubboCertificateServiceServer must be embedded to have forward compatible implementations.
type UnimplementedDubboCertificateServiceServer struct {
}

func (UnimplementedDubboCertificateServiceServer) CreateCertificate(context.Context, *DubboCertificateRequest) (*DubboCertificateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateCertificate not implemented")
}
func (UnimplementedDubboCertificateServiceServer) mustEmbedUnimplementedDubboCertificateServiceServer() {
}

// UnsafeDubboCertificateServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to DubboCertificateServiceServer will
// result in compilation errors.
type UnsafeDubboCertificateServiceServer interface {
	mustEmbedUnimplementedDubboCertificateServiceServer()
}

func RegisterDubboCertificateServiceServer(s grpc.ServiceRegistrar, srv DubboCertificateServiceServer) {
	s.RegisterService(&DubboCertificateService_ServiceDesc, srv)
}

func _DubboCertificateService_CreateCertificate_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DubboCertificateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(DubboCertificateServiceServer).CreateCertificate(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/org.apache.dubbo.auth.v1alpha1.DubboCertificateService/CreateCertificate",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(DubboCertificateServiceServer).CreateCertificate(ctx, req.(*DubboCertificateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// DubboCertificateService_ServiceDesc is the grpc.ServiceDesc for DubboCertificateService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var DubboCertificateService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "org.apache.dubbo.auth.v1alpha1.DubboCertificateService",
	HandlerType: (*DubboCertificateServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "CreateCertificate",
			Handler:    _DubboCertificateService_CreateCertificate_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "v1alpha1/ca.proto",
}

// ObserveServiceClient is the client API for ObserveService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ObserveServiceClient interface {
	Observe(ctx context.Context, opts ...grpc.CallOption) (ObserveService_ObserveClient, error)
}

type observeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewObserveServiceClient(cc grpc.ClientConnInterface) ObserveServiceClient {
	return &observeServiceClient{cc}
}

func (c *observeServiceClient) Observe(ctx context.Context, opts ...grpc.CallOption) (ObserveService_ObserveClient, error) {
	stream, err := c.cc.NewStream(ctx, &ObserveService_ServiceDesc.Streams[0], "/org.apache.dubbo.auth.v1alpha1.ObserveService/Observe", opts...)
	if err != nil {
		return nil, err
	}
	x := &observeServiceObserveClient{stream}
	return x, nil
}

type ObserveService_ObserveClient interface {
	Send(*ObserveRequest) error
	Recv() (*ObserveResponse, error)
	grpc.ClientStream
}

type observeServiceObserveClient struct {
	grpc.ClientStream
}

func (x *observeServiceObserveClient) Send(m *ObserveRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *observeServiceObserveClient) Recv() (*ObserveResponse, error) {
	m := new(ObserveResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ObserveServiceServer is the server API for ObserveService service.
// All implementations must embed UnimplementedObserveServiceServer
// for forward compatibility
type ObserveServiceServer interface {
	Observe(ObserveService_ObserveServer) error
	mustEmbedUnimplementedObserveServiceServer()
}

// UnimplementedObserveServiceServer must be embedded to have forward compatible implementations.
type UnimplementedObserveServiceServer struct {
}

func (UnimplementedObserveServiceServer) Observe(ObserveService_ObserveServer) error {
	return status.Errorf(codes.Unimplemented, "method Observe not implemented")
}
func (UnimplementedObserveServiceServer) mustEmbedUnimplementedObserveServiceServer() {}

// UnsafeObserveServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ObserveServiceServer will
// result in compilation errors.
type UnsafeObserveServiceServer interface {
	mustEmbedUnimplementedObserveServiceServer()
}

func RegisterObserveServiceServer(s grpc.ServiceRegistrar, srv ObserveServiceServer) {
	s.RegisterService(&ObserveService_ServiceDesc, srv)
}

func _ObserveService_Observe_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ObserveServiceServer).Observe(&observeServiceObserveServer{stream})
}

type ObserveService_ObserveServer interface {
	Send(*ObserveResponse) error
	Recv() (*ObserveRequest, error)
	grpc.ServerStream
}

type observeServiceObserveServer struct {
	grpc.ServerStream
}

func (x *observeServiceObserveServer) Send(m *ObserveResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *observeServiceObserveServer) Recv() (*ObserveRequest, error) {
	m := new(ObserveRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ObserveService_ServiceDesc is the grpc.ServiceDesc for ObserveService service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var ObserveService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "org.apache.dubbo.auth.v1alpha1.ObserveService",
	HandlerType: (*ObserveServiceServer)(nil),
	Methods:     []grpc.MethodDesc{},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "Observe",
			Handler:       _ObserveService_Observe_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
	},
	Metadata: "v1alpha1/ca.proto",
}
