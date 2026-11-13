/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */

package service

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	logcfg "github.com/apache/dubbo-admin/pkg/config/log"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store"
	memoryst "github.com/apache/dubbo-admin/pkg/store/memory"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionalpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dubbo-admin-service-test-logs-")
	if err != nil {
		panic(err)
	}
	// Init replaces the gRPC logger; run it once before any package test starts a server.
	cfg := logcfg.DefaultLogConfig()
	cfg.OutputPath = filepath.Join(dir, "dubbo-admin.log")
	logger.Init(cfg)
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

func idlTestCatalog(t *testing.T, port int64) (*testContext, model.ServiceGenericInvokeReq) {
	t.Helper()
	stores := map[coremodel.ResourceKind]store.ResourceStore{}
	for _, kind := range []coremodel.ResourceKind{meshresource.InstanceKind, meshresource.RPCInstanceKind, meshresource.ServiceProviderMetadataKind} {
		s := memoryst.NewMemoryResourceStore(kind)
		require.NoError(t, s.Init(nil))
		stores[kind] = s
	}
	instance := meshresource.NewInstanceResourceWithAttributes("instance", "mesh")
	instance.Spec.AppName = "app"
	instance.Spec.Ip = "127.0.0.1"
	instance.Spec.RpcPort = port
	require.NoError(t, stores[meshresource.InstanceKind].Add(instance))
	rpc := meshresource.NewRPCInstanceResourceWithAttributes(meshresource.BuildInstanceResName("app", "127.0.0.1", port), "mesh")
	rpc.Spec.AppName = "app"
	rpc.Spec.Ip = "127.0.0.1"
	rpc.Spec.Endpoints = []*meshproto.Endpoint{{Protocol: "tri", Port: port}}
	require.NoError(t, stores[meshresource.RPCInstanceKind].Add(rpc))
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("provider", "mesh")
	provider.Spec.ServiceName = "grpc.health.v1.Health"
	provider.Spec.ProviderAppName = "app"
	provider.Spec.Parameters = map[string]string{"IDL-mode": "IDL"}
	provider.Spec.Methods = []*meshproto.Method{{Name: "Check", ParameterTypes: []string{"grpc.health.v1.HealthCheckRequest"}, ReturnType: "grpc.health.v1.HealthCheckResponse"}}
	require.NoError(t, stores[meshresource.ServiceProviderMetadataKind].Add(provider))
	req := model.ServiceGenericInvokeReq{BaseServiceReq: model.BaseServiceReq{Mesh: "mesh", ServiceName: "grpc.health.v1.Health"}, InstanceName: "instance", MethodName: "Check", Signature: "grpc.health.v1.HealthCheckRequest->grpc.health.v1.HealthCheckResponse", Args: []json.RawMessage{json.RawMessage(`{}`)}, TimeoutMs: 3000}
	return &testContext{rm: manager.NewResourceManager(&testRouter{stores: stores}, nil), stores: stores}, req
}

func TestIDLUnaryCatalogInvoke(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	healthpb.RegisterHealthServer(server, health.NewServer())
	reflection.Register(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	ctx, req := idlTestCatalog(t, int64(listener.Addr().(*net.TCPAddr).Port))
	result, err := InvokeServiceGeneric(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "SERVING", result.RawResult.(map[string]any)["status"])
}

// The imported message exercises enum, oneof, int64 and bytes using real protojson rules.
func idlFixtureDescriptors(t *testing.T) (*protoregistry.Files, protoreflect.MessageDescriptor, *descriptorpb.FileDescriptorProto) {
	t.Helper()
	optional := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	str := descriptorpb.FieldDescriptorProto_TYPE_STRING
	enum := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	i64 := descriptorpb.FieldDescriptorProto_TYPE_INT64
	bytesType := descriptorpb.FieldDescriptorProto_TYPE_BYTES
	msg := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	dep := &descriptorpb.FileDescriptorProto{Name: proto.String("detail.proto"), Package: proto.String("fixture"), Syntax: proto.String("proto3"), MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Detail"), EnumType: []*descriptorpb.EnumDescriptorProto{{Name: proto.String("Kind"), Value: []*descriptorpb.EnumValueDescriptorProto{{Name: proto.String("UNKNOWN"), Number: proto.Int32(0)}, {Name: proto.String("READY"), Number: proto.Int32(1)}}}}, OneofDecl: []*descriptorpb.OneofDescriptorProto{{Name: proto.String("choice")}}, Field: []*descriptorpb.FieldDescriptorProto{
		{Name: proto.String("name"), Number: proto.Int32(1), Label: &optional, Type: &str, OneofIndex: proto.Int32(0)},
		{Name: proto.String("kind"), Number: proto.Int32(2), Label: &optional, Type: &enum, TypeName: proto.String(".fixture.Detail.Kind")},
		{Name: proto.String("count"), Number: proto.Int32(3), Label: &optional, Type: &i64},
		{Name: proto.String("payload"), Number: proto.Int32(4), Label: &optional, Type: &bytesType},
	}}}}
	root := &descriptorpb.FileDescriptorProto{Name: proto.String("service.proto"), Package: proto.String("fixture"), Syntax: proto.String("proto3"), Dependency: []string{"detail.proto"}, MessageType: []*descriptorpb.DescriptorProto{{Name: proto.String("Request"), Field: []*descriptorpb.FieldDescriptorProto{{Name: proto.String("detail"), Number: proto.Int32(1), Label: &optional, Type: &msg, TypeName: proto.String(".fixture.Detail")}}}}, Service: []*descriptorpb.ServiceDescriptorProto{{Name: proto.String("Echo"), Method: []*descriptorpb.MethodDescriptorProto{
		{Name: proto.String("Call"), InputType: proto.String(".fixture.Request"), OutputType: proto.String(".fixture.Request")},
		{Name: proto.String("Watch"), InputType: proto.String(".fixture.Request"), OutputType: proto.String(".fixture.Request"), ServerStreaming: proto.Bool(true)},
	}}}}
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{root, dep}})
	require.NoError(t, err)
	d, err := files.FindDescriptorByName("fixture.Request")
	require.NoError(t, err)
	return files, d.(protoreflect.MessageDescriptor), root
}

type idlFixtureService interface{}

func idlFixtureServer(t *testing.T, version string) int64 {
	t.Helper()
	files, message, _ := idlFixtureDescriptors(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "fixture.Echo", HandlerType: (*idlFixtureService)(nil), Methods: []grpc.MethodDesc{{MethodName: "Call", Handler: func(_ any, ctx context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
		request := dynamicpb.NewMessage(message)
		if err := decode(request); err != nil {
			return nil, err
		}
		return request, nil
	}}}}, struct{}{})
	options := reflection.ServerOptions{Services: server, DescriptorResolver: files}
	switch version {
	case "v1":
		reflectionv1.RegisterServerReflectionServer(server, reflection.NewServerV1(options))
	case "v1alpha":
		reflectionalpha.RegisterServerReflectionServer(server, reflection.NewServer(options))
	}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	return int64(listener.Addr().(*net.TCPAddr).Port)
}

func idlFixtureCatalog(t *testing.T, port int64) (*testContext, model.ServiceGenericInvokeReq) {
	t.Helper()
	ctx, req := idlTestCatalog(t, port)
	old, err := listProviderMeta(ctx, req.BaseServiceReq)
	require.NoError(t, err)
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Delete(old[0]))
	provider := meshresource.NewServiceProviderMetadataResourceWithAttributes("fixture", "mesh")
	provider.Spec.ServiceName = "fixture.Echo"
	provider.Spec.ProviderAppName = "app"
	provider.Spec.Parameters = map[string]string{"IDL-mode": "IDL"}
	for _, name := range []string{"Call", "Watch", "Absent"} {
		provider.Spec.Methods = append(provider.Spec.Methods, &meshproto.Method{Name: name, ParameterTypes: []string{"fixture.Request"}, ReturnType: "fixture.Request"})
	}
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(provider))
	req.ServiceName = "fixture.Echo"
	req.MethodName = "Call"
	req.Signature = "fixture.Request->fixture.Request"
	req.Args = []json.RawMessage{json.RawMessage(`{"detail":{"name":"Ada","kind":"READY","count":"9223372036854775807","payload":"AQI="}}`)}
	return ctx, req
}

type idlCountingListener struct {
	net.Listener
	accepted atomic.Int32
	closed   atomic.Int32
}

type idlCountingConn struct {
	net.Conn
	once   sync.Once
	closed *atomic.Int32
}

func (l *idlCountingListener) Accept() (net.Conn, error) {
	conn, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	l.accepted.Add(1)
	return &idlCountingConn{Conn: conn, closed: &l.closed}, nil
}

func (c *idlCountingConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { c.closed.Add(1) })
	return err
}

func idlTrackedServer(t *testing.T, handler func(context.Context)) (int64, *idlCountingListener, *atomic.Int32) {
	t.Helper()
	files, message, _ := idlFixtureDescriptors(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	tracked := &idlCountingListener{Listener: listener}
	calls := new(atomic.Int32)
	server := grpc.NewServer(grpc.StreamInterceptor(func(srv any, stream grpc.ServerStream, info *grpc.StreamServerInfo, next grpc.StreamHandler) error {
		md, _ := metadata.FromIncomingContext(stream.Context())
		for _, key := range []string{triple_protocol.TripleServiceGroup, triple_protocol.TripleServiceVersion, "x-e2e-echo"} {
			if len(md.Get(key)) != 0 {
				return status.Error(codes.InvalidArgument, "business metadata reached reflection")
			}
		}
		return next(srv, stream)
	}))
	server.RegisterService(&grpc.ServiceDesc{ServiceName: "fixture.Echo", HandlerType: (*idlFixtureService)(nil), Methods: []grpc.MethodDesc{{MethodName: "Call", Handler: func(_ any, ctx context.Context, decode func(any) error, _ grpc.UnaryServerInterceptor) (any, error) {
		calls.Add(1)
		request := dynamicpb.NewMessage(message)
		if err := decode(request); err != nil {
			return nil, err
		}
		if handler != nil {
			handler(ctx)
		}
		return request, nil
	}}}}, struct{}{})
	reflectionv1.RegisterServerReflectionServer(server, reflection.NewServerV1(reflection.ServerOptions{Services: server, DescriptorResolver: files}))
	go func() { _ = server.Serve(tracked) }()
	t.Cleanup(func() { server.Stop(); _ = tracked.Close() })
	return int64(listener.Addr().(*net.TCPAddr).Port), tracked, calls
}

func TestIDLCatalogReflectionAgreement(t *testing.T) {
	tests := []struct {
		name, input, output, canonical string
		valid                          bool
	}{
		{name: "matching", input: "fixture.Request", output: "fixture.Request", valid: true},
		{name: "leading dots", input: " .fixture.Request ", output: ".fixture.Request", canonical: ".fixture.Echo", valid: true},
		{name: "stale input", input: "fixture.OldRequest", output: "fixture.Request"},
		{name: "stale output", input: "fixture.Request", output: "fixture.OldResponse"},
		{name: "array input", input: "fixture.Request[]", output: "fixture.Request"},
		{name: "generic input", input: "List<fixture.Request>", output: "fixture.Request"},
		{name: "case mismatch", input: "fixture.request", output: "fixture.Request"},
		{name: "canonical conflict", input: "fixture.Request", output: "fixture.Request", canonical: "fixture.Other"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			port, _, calls := idlTrackedServer(t, nil)
			ctx, req := idlFixtureCatalog(t, port)
			providers, err := listProviderMeta(ctx, req.BaseServiceReq)
			require.NoError(t, err)
			provider := providers[0]
			provider.Spec.CanonicalName = tc.canonical
			provider.Spec.Methods[0].ParameterTypes = []string{tc.input}
			provider.Spec.Methods[0].ReturnType = tc.output
			req.Signature = tc.input + "->" + tc.output
			require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(provider))
			_, err = InvokeServiceGeneric(ctx, req)
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, int32(1), calls.Load())
			} else {
				message := "catalog signature"
				if tc.canonical != "" {
					message = "canonical name"
				}
				assertIDLBusinessError(t, err, bizerror.InvalidArgument, message)
				require.Zero(t, calls.Load())
			}
		})
	}
}

func TestIDLConnectionsAndMetadata(t *testing.T) {
	entered := make(chan metadata.MD, 10)
	block := make(chan struct{}, 1)
	port, listener, calls := idlTrackedServer(t, func(ctx context.Context) {
		md, _ := metadata.FromIncomingContext(ctx)
		entered <- md.Copy()
		select {
		case <-block:
			<-ctx.Done()
		default:
		}
	})
	ctx, req := idlFixtureCatalog(t, port)
	req.Attachments = map[string]string{" X-E2E-Echo ": "ok"}
	assertClosed := func(want int32) {
		t.Helper()
		require.Eventually(t, func() bool { return listener.accepted.Load() == want && listener.closed.Load() == want }, 3*time.Second, 10*time.Millisecond, "accepted=%d closed=%d", listener.accepted.Load(), listener.closed.Load())
	}
	for i := int32(1); i <= 3; i++ {
		_, err := InvokeServiceGeneric(ctx, req)
		require.NoError(t, err)
		md := <-entered
		require.Equal(t, []string{"ok"}, md.Get("x-e2e-echo"))
		require.Empty(t, md.Get(triple_protocol.TripleServiceGroup))
		require.Empty(t, md.Get(triple_protocol.TripleServiceVersion))
		assertClosed(i)
	}
	providers, err := listProviderMeta(ctx, req.BaseServiceReq)
	require.NoError(t, err)
	provider := providers[0]
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Delete(provider))
	provider.Spec.Group, provider.Spec.Version = "group-a", "version-b"
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(provider))
	req.Group, req.Version = provider.Spec.Group, provider.Spec.Version
	_, err = InvokeServiceGeneric(ctx, req)
	require.NoError(t, err)
	md := <-entered
	require.Equal(t, []string{"ok"}, md.Get("x-e2e-echo"))
	require.Equal(t, []string{"group-a"}, md.Get(triple_protocol.TripleServiceGroup))
	require.Equal(t, []string{"version-b"}, md.Get(triple_protocol.TripleServiceVersion))
	assertClosed(4)
	invalid := req
	invalid.Args = []json.RawMessage{json.RawMessage(`{`)}
	_, err = InvokeServiceGeneric(ctx, invalid)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "invalid protobuf request JSON")
	assertClosed(5)
	require.Equal(t, int32(4), calls.Load())
	block <- struct{}{}
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		_, err := InvokeServiceGeneric(&idlContext{testContext: ctx, parent: parent}, req)
		done <- err
	}()
	select {
	case <-entered:
	case <-time.After(3 * time.Second):
		cancel()
		t.Fatal("handler was not entered")
	}
	cancel()
	select {
	case err := <-done:
		assertIDLBusinessError(t, err, bizerror.InternalError, "context canceled")
	case <-time.After(3 * time.Second):
		t.Fatal("canceled call did not return")
	}
	require.Equal(t, int32(5), calls.Load())
	assertClosed(6)
}

func TestIDLRejectsInvalidAttachmentKeys(t *testing.T) {
	port, listener, calls := idlTrackedServer(t, nil)
	ctx, req := idlFixtureCatalog(t, port)
	providers, err := listProviderMeta(ctx, req.BaseServiceReq)
	require.NoError(t, err)
	provider := providers[0]
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Delete(provider))
	provider.Spec.Group, provider.Spec.Version = "group-a", "version-b"
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(provider))
	req.Group, req.Version = provider.Spec.Group, provider.Spec.Version

	for _, tc := range []struct{ key, value, message string }{
		{"content-type", "secret-value", "content-type"},
		{" :AUTHORITY ", "secret-value", ":authority"},
		{"grpc-status", "secret-value", "grpc-status"},
		{"tri-service-group", "group-a", "tri-service-group"},
		{"tri-service-version", "version-b", "tri-service-version"},
		{"  ", "secret-value", "key must not be empty"},
		{"invalid/key", "secret-value", "invalid/key"},
	} {
		t.Run(tc.message, func(t *testing.T) {
			r := req
			r.Attachments = map[string]string{tc.key: tc.value}
			_, err := InvokeServiceGeneric(ctx, r)
			assertIDLBusinessError(t, err, bizerror.InvalidArgument, tc.message)
			require.NotContains(t, err.Error(), tc.value)
			require.Zero(t, calls.Load())
			require.Zero(t, listener.accepted.Load())
		})
	}
}

func TestIDLInvokeContext(t *testing.T) {
	req := model.ServiceGenericInvokeReq{
		BaseServiceReq: model.BaseServiceReq{Group: "group-a", Version: "version-b"},
		Attachments:    map[string]string{" X-E2E-Echo ": "ok", "payload-bin": "\x00\x01"},
	}
	ctx, err := idlInvokeContext(context.Background(), req)
	require.NoError(t, err)
	md, ok := metadata.FromOutgoingContext(ctx)
	require.True(t, ok)
	require.Equal(t, []string{"ok"}, md.Get("x-e2e-echo"))
	require.Equal(t, []string{"\x00\x01"}, md.Get("payload-bin"))
	require.Equal(t, []string{"group-a"}, md.Get(triple_protocol.TripleServiceGroup))
	require.Equal(t, []string{"version-b"}, md.Get(triple_protocol.TripleServiceVersion))

	for _, key := range []string{"content-type", ":authority", "grpc-status", "tri-service-group", "tri-service-version", "  ", "invalid/key"} {
		t.Run(key, func(t *testing.T) {
			r := req
			r.Attachments = map[string]string{key: "secret-value"}
			_, err := idlInvokeContext(context.Background(), r)
			assertIDLBusinessError(t, err, bizerror.InvalidArgument, "attachment key")
			require.NotContains(t, err.Error(), "secret-value")
		})
	}
}

func assertIDLBusinessError(t *testing.T, err error, code bizerror.ErrorCode, contains string) {
	t.Helper()
	require.Error(t, err)
	var business bizerror.Error
	require.ErrorAs(t, err, &business)
	require.Equal(t, code, business.Code())
	require.Contains(t, business.Message(), contains)
	envelope := model.NewBizErrorResp(business)
	require.Equal(t, string(code), envelope.Code)
	require.Nil(t, envelope.Data)
}

func TestIDLDescriptorsAndUnary(t *testing.T) {
	for _, version := range []string{"v1", "v1alpha"} {
		t.Run(version, func(t *testing.T) {
			ctx, req := idlFixtureCatalog(t, idlFixtureServer(t, version))
			result, err := InvokeServiceGeneric(ctx, req)
			require.NoError(t, err)
			envelope := model.NewSuccessResp(result)
			require.Equal(t, "Success", envelope.Code)
			detail := result.RawResult.(map[string]any)["detail"].(map[string]any)
			require.Equal(t, "Ada", detail["name"])
			require.Equal(t, "READY", detail["kind"])
			require.Equal(t, "9223372036854775807", detail["count"])
			require.Equal(t, "AQI=", detail["payload"])
		})
	}
}

func TestIDLInvokeErrors(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, idlFixtureServer(t, "v1"))
	tests := []struct {
		name, method, signature, body string
		args                          []json.RawMessage
		code                          bizerror.ErrorCode
		message                       string
	}{
		{name: "invalid JSON", body: `{`, code: bizerror.InvalidArgument, message: "invalid protobuf request JSON"},
		{name: "unknown field wrapper", body: `{"class":"fixture.Request"}`, code: bizerror.InvalidArgument, message: "unknown field"},
		{name: "bare string", body: `"Ada"`, code: bizerror.InvalidArgument, message: "invalid protobuf request JSON"},
		{name: "arity", args: []json.RawMessage{}, code: bizerror.InvalidArgument, message: "parameter count"},
		{name: "unknown catalog method", method: "Unknown", code: bizerror.NotFoundError, message: "method Unknown not found"},
		{name: "unknown descriptor method", method: "Absent", code: bizerror.NotFoundError, message: "method not found in reflection"},
		{name: "streaming", method: "Watch", code: bizerror.InvalidArgument, message: "unary only"},
		{name: "signature mismatch", signature: "bad", code: bizerror.NotFoundError, message: "method Call not found"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := req
			if tc.method != "" {
				r.MethodName = tc.method
			}
			if tc.signature != "" {
				r.Signature = tc.signature
			}
			if tc.body != "" {
				r.Args = []json.RawMessage{json.RawMessage(tc.body)}
			}
			if tc.args != nil {
				r.Args = tc.args
			}
			_, err := InvokeServiceGeneric(ctx, r)
			assertIDLBusinessError(t, err, tc.code, tc.message)
		})
	}
}

func TestIDLReflectionUnsupported(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, idlFixtureServer(t, "none"))
	_, err := InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "reflection unsupported")
}

type idlContext struct {
	*testContext
	parent context.Context
}

func (c *idlContext) AppContext() context.Context { return c.parent }
func TestIDLContextErrors(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, idlFixtureServer(t, "v1"))
	req.Attachments = map[string]string{"x-e2e-echo": "ok"}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := InvokeServiceGeneric(&idlContext{testContext: ctx, parent: canceled}, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "context canceled")
	req.TimeoutMs = -1
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "context deadline exceeded")
}

func TestIDLProviderBindingAndSecurity(t *testing.T) {
	port, listener, calls := idlTrackedServer(t, nil)
	ctx, req := idlFixtureCatalog(t, port)
	missing := req
	missing.InstanceName = "unregistered"
	_, err := InvokeServiceGeneric(ctx, missing)
	assertIDLBusinessError(t, err, bizerror.NotFoundError, "instance unregistered not found")
	p, err := listProviderMeta(ctx, req.BaseServiceReq)
	require.NoError(t, err)
	other := meshresource.NewServiceProviderMetadataResourceWithAttributes("other", "mesh")
	other.Spec.ServiceName = req.ServiceName
	other.Spec.ProviderAppName = "other-app"
	other.Spec.Parameters = map[string]string{"IDL-mode": "non-IDL"}
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(other))
	_, err = InvokeServiceGeneric(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int32(1), calls.Load())
	calls.Store(0)
	other.Spec.ProviderAppName = "app"
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(other))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "conflicting IDL-mode")
	other.Spec.Parameters["IDL-mode"] = "unknown"
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(other))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "unknown IDL-mode")
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Delete(other))
	p[0].Spec.Parameters["ssl-enabled"] = "true"
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(p[0]))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "secure or authenticated")
	accepted := listener.accepted.Load()
	for _, app := range []string{"other-app", ""} {
		p[0].Spec.ProviderAppName = app
		require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(p[0]))
		_, err = InvokeServiceGeneric(ctx, req)
		assertIDLBusinessError(t, err, bizerror.InvalidArgument, "IDL provider metadata is not bound to the selected instance")
		require.Equal(t, int32(0), calls.Load())
		require.Equal(t, accepted, listener.accepted.Load())
	}
}

func TestIDLRejectsMismatchedRPCInstance(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, 1)
	instance, err := getGenericInvokeInstance(ctx, "mesh", "instance")
	require.NoError(t, err)
	rpc, err := findRPCInstanceByInstance(ctx, instance)
	require.NoError(t, err)
	rpc.Spec.AppName = "other-app"
	require.NoError(t, ctx.stores[meshresource.RPCInstanceKind].Update(rpc))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "does not match selected catalog instance")
}

func TestIDLRejectsInstanceSecurityMarkers(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, 1)
	instance, err := getGenericInvokeInstance(ctx, "mesh", "instance")
	require.NoError(t, err)
	rpc, err := findRPCInstanceByInstance(ctx, instance)
	require.NoError(t, err)
	rpc.Spec.Metadata = map[string]string{constants.URLParamsKey: `{"ssl-enabled":"true"}`}
	require.NoError(t, ctx.stores[meshresource.RPCInstanceKind].Update(rpc))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InvalidArgument, "secure or authenticated")
}

func TestIDLSecurityKnownMarkers(t *testing.T) {
	require.True(t, idlSecurityRequired(map[string]string{constant.CACert: "ca.pem"}))
	require.True(t, idlSecurityRequired(map[string]string{constant.AccessKeyIDKey: "key"}))
	require.True(t, idlSecurityRequired(map[string]string{constants.URLParamsKey: "{"}))
	require.False(t, idlSecurityRequired(map[string]string{constant.SslEnabledKey: "false", "tls-enabled": "true"}))
}

func TestNonIDLUsesExistingGenericPath(t *testing.T) {
	ctx, req := idlFixtureCatalog(t, 1)
	p, err := listProviderMeta(ctx, req.BaseServiceReq)
	require.NoError(t, err)
	delete(p[0].Spec.Parameters, "IDL-mode")
	require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(p[0]))
	previous := invokeGenericServiceRPC
	t.Cleanup(func() { invokeGenericServiceRPC = previous })
	called := false
	invokeGenericServiceRPC = func(_ context.Context, inv genericInvocation) (any, error) {
		called = true
		require.Equal(t, "tri://127.0.0.1:1", inv.URL)
		return map[string]any{"ok": true}, nil
	}
	result, err := InvokeServiceGeneric(ctx, req)
	require.NoError(t, err)
	require.True(t, called)
	require.Equal(t, true, result.RawResult.(map[string]any)["ok"])
	invokeGenericServiceRPC = func(_ context.Context, _ genericInvocation) (any, error) {
		return nil, errors.New("underlying failure")
	}
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "generic invoke failed, please check server logs")
}

func TestNonIDLMetadataBinding(t *testing.T) {
	previous := invokeGenericServiceRPC
	t.Cleanup(func() { invokeGenericServiceRPC = previous })
	for _, tc := range []struct {
		name, app, mode string
		otherProviders  bool
		mismatchedRPC   bool
	}{
		{name: "legacy metadata without app"},
		{name: "bound empty mode", app: "app", otherProviders: true},
		{name: "bound non-IDL mode", app: "app", mode: "non-IDL", otherProviders: true, mismatchedRPC: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, req := idlFixtureCatalog(t, 1)
			providers, err := listProviderMeta(ctx, req.BaseServiceReq)
			require.NoError(t, err)
			providers[0].Spec.ProviderAppName = tc.app
			if tc.mode == "" {
				delete(providers[0].Spec.Parameters, "IDL-mode")
			} else {
				providers[0].Spec.Parameters["IDL-mode"] = tc.mode
			}
			require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(providers[0]))
			if tc.otherProviders {
				for _, mode := range []string{"IDL", "unknown"} {
					other := meshresource.NewServiceProviderMetadataResourceWithAttributes(mode, "mesh")
					other.Spec.ServiceName = req.ServiceName
					other.Spec.ProviderAppName = "other-app"
					other.Spec.Parameters = map[string]string{"IDL-mode": mode}
					other.Spec.Methods = providers[0].Spec.Methods
					require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(other))
				}
			}
			if tc.mismatchedRPC {
				instance, err := getGenericInvokeInstance(ctx, "mesh", "instance")
				require.NoError(t, err)
				rpc, err := findRPCInstanceByInstance(ctx, instance)
				require.NoError(t, err)
				rpc.Spec.AppName = "other-app"
				require.NoError(t, ctx.stores[meshresource.RPCInstanceKind].Update(rpc))
			}
			called := false
			invokeGenericServiceRPC = func(_ context.Context, _ genericInvocation) (any, error) {
				called = true
				return "ok", nil
			}
			result, err := InvokeServiceGeneric(ctx, req)
			require.NoError(t, err)
			require.True(t, called)
			require.Equal(t, "ok", result.RawResult)
		})
	}
}

type incompleteReflection struct {
	reflectionv1.UnimplementedServerReflectionServer
	root []byte
}

func (s *incompleteReflection) ServerReflectionInfo(stream grpc.BidiStreamingServer[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]) error {
	request, err := stream.Recv()
	if err != nil {
		return err
	}
	if request.GetFileContainingSymbol() != "" {
		return stream.Send(&reflectionv1.ServerReflectionResponse{MessageResponse: &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{FileDescriptorProto: [][]byte{s.root}}}})
	}
	return stream.Send(&reflectionv1.ServerReflectionResponse{MessageResponse: &reflectionv1.ServerReflectionResponse_ErrorResponse{ErrorResponse: &reflectionv1.ErrorResponse{ErrorCode: int32(codes.NotFound), ErrorMessage: "file not found"}}})
}
func TestIDLMissingDependency(t *testing.T) {
	_, _, root := idlFixtureDescriptors(t)
	data, err := proto.Marshal(root)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	reflectionv1.RegisterServerReflectionServer(server, &incompleteReflection{root: data})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	ctx, req := idlFixtureCatalog(t, int64(listener.Addr().(*net.TCPAddr).Port))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "missing dependency detail.proto")
}

type excessiveReflection struct {
	reflectionv1.UnimplementedServerReflectionServer
	raw []byte
}

func (s *excessiveReflection) ServerReflectionInfo(stream grpc.BidiStreamingServer[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]) error {
	if _, err := stream.Recv(); err != nil {
		return err
	}
	files := make([][]byte, 257)
	for i := range files {
		files[i] = s.raw
	}
	return stream.Send(&reflectionv1.ServerReflectionResponse{MessageResponse: &reflectionv1.ServerReflectionResponse_FileDescriptorResponse{FileDescriptorResponse: &reflectionv1.FileDescriptorResponse{FileDescriptorProto: files}}})
}

func TestIDLDescriptorLimit(t *testing.T) {
	_, _, root := idlFixtureDescriptors(t)
	raw, err := proto.Marshal(root)
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	reflectionv1.RegisterServerReflectionServer(server, &excessiveReflection{raw: raw})
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	ctx, req := idlFixtureCatalog(t, int64(listener.Addr().(*net.TCPAddr).Port))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "reflection descriptor limit exceeded")
}

type deniedReflection struct {
	reflectionv1.UnimplementedServerReflectionServer
}

func (*deniedReflection) ServerReflectionInfo(grpc.BidiStreamingServer[reflectionv1.ServerReflectionRequest, reflectionv1.ServerReflectionResponse]) error {
	return status.Error(codes.PermissionDenied, "denied")
}

type countedAlphaReflection struct {
	reflectionalpha.UnimplementedServerReflectionServer
	calls atomic.Int32
}

//nolint:staticcheck // v1alpha types are deprecated but required to assert no fallback.
func (s *countedAlphaReflection) ServerReflectionInfo(grpc.BidiStreamingServer[reflectionalpha.ServerReflectionRequest, reflectionalpha.ServerReflectionResponse]) error {
	s.calls.Add(1)
	return status.Error(codes.Unimplemented, "unsupported")
}
func TestIDLReflectionDoesNotDowngradePermissionDenied(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := grpc.NewServer()
	alpha := new(countedAlphaReflection)
	reflectionv1.RegisterServerReflectionServer(server, &deniedReflection{})
	reflectionalpha.RegisterServerReflectionServer(server, alpha)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); _ = listener.Close() })
	ctx, req := idlFixtureCatalog(t, int64(listener.Addr().(*net.TCPAddr).Port))
	_, err = InvokeServiceGeneric(ctx, req)
	assertIDLBusinessError(t, err, bizerror.InternalError, "PermissionDenied")
	require.Equal(t, int32(0), alpha.calls.Load())
}
