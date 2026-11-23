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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"slices"
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	"github.com/apache/dubbo-admin/pkg/common/constants"
	"github.com/apache/dubbo-admin/pkg/console/model"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	reflectionv1 "google.golang.org/grpc/reflection/grpc_reflection_v1"
	reflectionalpha "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// Only service-scoped provider parameters establish the mode; instance tags can
// describe other services exported by the same process.
func genericInvokeIDLMode(selected []*meshresource.ServiceProviderMetadataResource) (bool, error) {
	hasIDL, hasNonIDL := false, false
	for _, provider := range selected {
		if provider == nil || provider.Spec == nil {
			continue
		}
		value := strings.TrimSpace(provider.Spec.Parameters[constant.IDLMode])
		if value == "" {
			continue
		}
		switch {
		case strings.EqualFold(value, constant.IDL):
			hasIDL = true
		case strings.EqualFold(value, constant.NONIDL):
			hasNonIDL = true
		default:
			return false, bizerror.New(bizerror.InvalidArgument, "unknown IDL-mode in provider metadata")
		}
	}
	if hasIDL && hasNonIDL {
		return false, bizerror.New(bizerror.InvalidArgument, "conflicting IDL-mode for service providers")
	}
	return hasIDL, nil
}

// Known SDK markers require credentials this path cannot configure. Unknown marker
// names are not covered. Reject malformed catalog URL parameters rather than dial.
func idlSecurityRequired(values map[string]string) bool {
	if raw, ok := values[constants.URLParamsKey]; ok {
		var params map[string]string
		if err := json.Unmarshal([]byte(raw), &params); err != nil || params == nil {
			return true
		}
		if idlSecurityMarkers(params) {
			return true
		}
	}
	return idlSecurityMarkers(values)
}

func idlSecurityMarkers(values map[string]string) bool {
	for key, value := range values {
		key = strings.TrimSpace(key)
		value = strings.ToLower(strings.TrimSpace(value))
		switch key {
		case constant.SslEnabledKey, constant.ServiceAuthKey:
			if value != "" && value != "false" && value != "0" && value != "none" {
				return true
			}
		case constant.TokenKey, constant.AuthenticatorKey, constant.TLSConfigKey, constant.TLSKey, constant.TLSCert, constant.CACert, constant.TLSServerNAME, constant.AccessKeyIDKey, constant.SecretAccessKeyKey:
			if value != "" {
				return true
			}
		}
	}
	return false
}

func idlCallError(ctx context.Context, operation string, err error) error {
	if ctx.Err() != nil {
		return bizerror.New(bizerror.InternalError, operation+": "+ctx.Err().Error())
	}
	switch status.Code(err) {
	case codes.DeadlineExceeded:
		return bizerror.New(bizerror.InternalError, operation+": "+context.DeadlineExceeded.Error())
	case codes.Canceled:
		return bizerror.New(bizerror.InternalError, operation+": "+context.Canceled.Error())
	}
	return bizerror.New(bizerror.InternalError, fmt.Sprintf("%s: %v", operation, err))
}

func idlFullName(name string) protoreflect.FullName {
	return protoreflect.FullName(strings.TrimPrefix(strings.TrimSpace(name), "."))
}

func idlInvokeContext(ctx context.Context, req model.ServiceGenericInvokeReq) (context.Context, error) {
	var pairs []string
	for key, value := range req.Attachments {
		key = strings.ToLower(strings.TrimSpace(key))
		if key == "" {
			return nil, bizerror.New(bizerror.InvalidArgument, "attachment key must not be empty")
		}
		if triple_protocol.IsReservedHeader(key) || key == triple_protocol.TripleServiceGroup || key == triple_protocol.TripleServiceVersion {
			return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("reserved attachment key %q", key))
		}
		for _, c := range key {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '.' && c != '_' && c != '-' {
				return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("invalid attachment key %q", key))
			}
		}
		pairs = append(pairs, key, value)
	}
	if req.Group != "" {
		pairs = append(pairs, triple_protocol.TripleServiceGroup, req.Group)
	}
	if req.Version != "" {
		pairs = append(pairs, triple_protocol.TripleServiceVersion, req.Version)
	}
	return metadata.AppendToOutgoingContext(ctx, pairs...), nil
}

func invokeIDLService(ctx context.Context, req model.ServiceGenericInvokeReq, targets []*genericInvokeTarget, providers []*meshresource.ServiceProviderMetadataResource, catalogMethod *meshproto.Method) (any, error) {
	if len(req.Args) != 1 {
		return nil, bizerror.New(bizerror.InvalidArgument, "IDL unary methods require exactly one request message")
	}
	for _, p := range providers {
		if p != nil && p.Spec != nil && idlSecurityRequired(p.Spec.Parameters) {
			return nil, bizerror.New(bizerror.InvalidArgument, "secure or authenticated IDL instance is unsupported without configured credentials")
		}
	}
	var target *genericInvokeTarget
	for _, t := range targets {
		if idlSecurityRequired(t.instance.Spec.GetMetadata()) || idlSecurityRequired(t.instance.Spec.GetTags()) {
			return nil, bizerror.New(bizerror.InvalidArgument, "secure or authenticated IDL instance is unsupported without configured credentials")
		}
		if target == nil && t.protocol == constant.TriProtocol && t.port > 0 && t.port <= 65535 {
			target = t
		}
	}
	if target == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "IDL invocation requires a Triple endpoint")
	}
	invokeCtx, err := idlInvokeContext(ctx, req)
	if err != nil {
		return nil, err
	}
	addr := net.JoinHostPort(target.instance.Spec.GetIp(), strconv.FormatInt(target.port, 10))
	cc, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, idlCallError(ctx, "reflection connection failed", err)
	}
	defer func() { _ = cc.Close() }()
	// Reflection is process-global and cannot distinguish divergent descriptors
	// for versions of the same service. Routing headers apply only to Invoke.
	reflector, err := newIDLReflector(ctx, cc, req.ServiceName)
	if err != nil {
		return nil, idlCallError(ctx, "IDL reflection failed", err)
	}
	desc, err := reflector.registry.FindDescriptorByName(protoreflect.FullName(req.ServiceName))
	if err != nil {
		return nil, bizerror.New(bizerror.NotFoundError, "service not found in reflection descriptor")
	}
	svc, ok := desc.(protoreflect.ServiceDescriptor)
	if !ok {
		return nil, bizerror.New(bizerror.InvalidArgument, "reflected symbol is not a service")
	}
	method := svc.Methods().ByName(protoreflect.Name(req.MethodName))
	if method == nil {
		return nil, bizerror.New(bizerror.NotFoundError, "method not found in reflection descriptor")
	}
	if method.IsStreamingClient() || method.IsStreamingServer() {
		return nil, bizerror.New(bizerror.InvalidArgument, "streaming is unsupported; IDL invocation is unary only")
	}
	for _, provider := range providers {
		if provider == nil || provider.Spec == nil || strings.TrimSpace(provider.Spec.GetCanonicalName()) == "" {
			continue
		}
		name := idlFullName(provider.Spec.GetCanonicalName())
		if !name.IsValid() || name != svc.FullName() || name != idlFullName(req.ServiceName) {
			return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("catalog provider canonical name %q conflicts with requested service %q and reflected service %q", provider.Spec.GetCanonicalName(), req.ServiceName, svc.FullName()))
		}
	}
	params := catalogMethod.GetParameterTypes()
	if len(params) != 1 || !idlFullName(params[0]).IsValid() || !idlFullName(catalogMethod.GetReturnType()).IsValid() || idlFullName(params[0]) != method.Input().FullName() || idlFullName(catalogMethod.GetReturnType()) != method.Output().FullName() {
		return nil, bizerror.New(bizerror.InvalidArgument, fmt.Sprintf("catalog signature %q conflicts with reflected signature %s->%s", methodSig(catalogMethod), method.Input().FullName(), method.Output().FullName()))
	}
	types := &idlTypes{Types: dynamicpb.NewTypes(reflector.registry), reflector: reflector}
	input := dynamicpb.NewMessage(method.Input())
	// Each args element is the protobuf JSON document, not a Hessian class wrapper.
	if err = (protojson.UnmarshalOptions{DiscardUnknown: false, Resolver: types}).Unmarshal(req.Args[0], input); err != nil {
		if types.lookupErr != nil {
			return nil, idlCallError(ctx, "IDL reflection failed", types.lookupErr)
		}
		return nil, bizerror.New(bizerror.InvalidArgument, "invalid protobuf request JSON: "+err.Error())
	}
	output := dynamicpb.NewMessage(method.Output())
	if err = cc.Invoke(invokeCtx, "/"+string(svc.FullName())+"/"+string(method.Name()), input, output); err != nil {
		return nil, idlCallError(ctx, "IDL unary invocation failed", err)
	}
	// The method has already run, so the error says so to discourage a retry.
	const encodingFailed = "IDL invocation succeeded but its response cannot be encoded"
	encoded, err := (protojson.MarshalOptions{EmitUnpopulated: true, Resolver: types}).Marshal(output)
	if err != nil {
		return nil, idlCallError(ctx, encodingFailed, err)
	}
	var result any
	if err = json.Unmarshal(encoded, &result); err != nil {
		return nil, idlCallError(ctx, encodingFailed, err)
	}
	return result, nil
}

// A fresh stream per query makes each version probe independent.
func idlReflectionQuery(ctx context.Context, cc *grpc.ClientConn, alpha bool, symbol bool, value string) (*reflectionv1.ServerReflectionResponse, error) {
	request := &reflectionv1.ServerReflectionRequest{}
	if symbol {
		request.MessageRequest = &reflectionv1.ServerReflectionRequest_FileContainingSymbol{FileContainingSymbol: value}
	} else {
		request.MessageRequest = &reflectionv1.ServerReflectionRequest_FileByFilename{FileByFilename: value}
	}
	var response *reflectionv1.ServerReflectionResponse
	if !alpha {
		stream, err := reflectionv1.NewServerReflectionClient(cc).ServerReflectionInfo(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = stream.CloseSend() }()
		if err = stream.Send(request); err != nil && err != io.EOF {
			return nil, err
		}
		response, err = stream.Recv()
		if err != nil {
			return nil, err
		}
	} else {
		data, err := proto.Marshal(request)
		if err != nil {
			return nil, err
		}
		// Fallback uses the v1alpha wire type; the proto is deprecated but still required.
		ar := new(reflectionalpha.ServerReflectionRequest) //nolint:staticcheck
		if err = proto.Unmarshal(data, ar); err != nil {
			return nil, err
		}
		stream, err := reflectionalpha.NewServerReflectionClient(cc).ServerReflectionInfo(ctx)
		if err != nil {
			return nil, err
		}
		defer func() { _ = stream.CloseSend() }()
		if err = stream.Send(ar); err != nil && err != io.EOF {
			return nil, err
		}
		reply, err := stream.Recv()
		if err != nil {
			return nil, err
		}
		data, err = proto.Marshal(reply)
		if err != nil {
			return nil, err
		}
		response = new(reflectionv1.ServerReflectionResponse)
		if err = proto.Unmarshal(data, response); err != nil {
			return nil, err
		}
	}
	if e := response.GetErrorResponse(); e != nil {
		return nil, status.Error(codes.Code(e.GetErrorCode()), e.GetErrorMessage())
	}
	return response, nil
}

// idlReflector fetches descriptors over one reflection connection into one
// registry. All fetches of an invocation share the descriptor limits.
type idlReflector struct {
	// ctx bounds every fetch of the invocation; protojson's resolver, which
	// fetches Any payload types, takes no context.
	ctx      context.Context
	cc       *grpc.ClientConn
	alpha    bool
	protos   map[string]*descriptorpb.FileDescriptorProto
	registry *protoregistry.Files
	files    int
	bytes    int
	rounds   int
}

const (
	idlMaxDescriptorFiles  = 256
	idlMaxDescriptorBytes  = 8 << 20
	idlMaxReflectionRounds = 256
)

// newIDLReflector fetches the file defining service and its dependencies.
func newIDLReflector(ctx context.Context, cc *grpc.ClientConn, service string) (*idlReflector, error) {
	r := &idlReflector{ctx: ctx, cc: cc, protos: map[string]*descriptorpb.FileDescriptorProto{}, registry: new(protoregistry.Files)}
	response, err := idlReflectionQuery(ctx, cc, false, true, service)
	if status.Code(err) == codes.Unimplemented {
		r.alpha = true
		response, err = idlReflectionQuery(ctx, cc, true, true, service)
	}
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return nil, fmt.Errorf("reflection unsupported: %w", err)
		}
		return nil, err
	}
	if err = r.load(response); err != nil {
		return nil, err
	}
	return r, nil
}

// query sends a reflection request after the initial one. Each request counts
// toward the round limit.
func (r *idlReflector) query(symbol bool, value string) (*reflectionv1.ServerReflectionResponse, error) {
	r.rounds++
	if r.rounds >= idlMaxReflectionRounds {
		return nil, fmt.Errorf("reflection fetch round limit exceeded")
	}
	return idlReflectionQuery(r.ctx, r.cc, r.alpha, symbol, value)
}

// load adds the files of response, fetches their missing dependencies and
// registers them all.
func (r *idlReflector) load(response *reflectionv1.ServerReflectionResponse) error {
	if err := r.add(response); err != nil {
		return err
	}
	for {
		missing, found := r.missingDependency()
		if !found {
			break
		}
		response, err := r.query(false, missing)
		if err != nil {
			return fmt.Errorf("missing dependency %s: %w", missing, err)
		}
		if err = r.add(response); err != nil {
			return fmt.Errorf("dependency %s: %w", missing, err)
		}
		if _, ok := r.protos[missing]; !ok {
			return fmt.Errorf("reflection did not return dependency %s", missing)
		}
	}
	visiting := map[string]bool{}
	for name := range r.protos {
		if err := r.register(name, visiting); err != nil {
			return fmt.Errorf("incomplete or invalid descriptors: %w", err)
		}
	}
	return nil
}

func (r *idlReflector) missingDependency() (string, bool) {
	for _, file := range r.protos {
		for _, dep := range file.Dependency {
			if _, ok := r.protos[dep]; !ok {
				return dep, true
			}
		}
	}
	return "", false
}

func (r *idlReflector) add(response *reflectionv1.ServerReflectionResponse) error {
	data := response.GetFileDescriptorResponse()
	if data == nil || len(data.FileDescriptorProto) == 0 {
		return fmt.Errorf("reflection returned no descriptors")
	}
	for _, raw := range data.FileDescriptorProto {
		r.files++
		r.bytes += len(raw)
		if r.files > idlMaxDescriptorFiles || r.bytes > idlMaxDescriptorBytes {
			return fmt.Errorf("reflection descriptor limit exceeded")
		}
		file := new(descriptorpb.FileDescriptorProto)
		if err := proto.Unmarshal(raw, file); err != nil {
			return fmt.Errorf("invalid descriptor: %w", err)
		}
		if file.GetName() == "" {
			return fmt.Errorf("descriptor has no filename")
		}
		if slices.Contains(file.Dependency, "") {
			return fmt.Errorf("descriptor %s has an empty import", file.GetName())
		}
		if old, ok := r.protos[file.GetName()]; ok && !proto.Equal(old, file) {
			return fmt.Errorf("conflicting descriptor %s", file.GetName())
		}
		r.protos[file.GetName()] = file
	}
	return nil
}

// register registers a fetched file after its dependencies and rejects import
// cycles.
func (r *idlReflector) register(name string, visiting map[string]bool) error {
	if _, err := r.registry.FindFileByPath(name); err == nil {
		return nil
	}
	if visiting[name] {
		return fmt.Errorf("import cycle through %s", name)
	}
	visiting[name] = true
	file, ok := r.protos[name]
	if !ok {
		return fmt.Errorf("could not resolve import %s", name)
	}
	for _, dep := range file.Dependency {
		if err := r.register(dep, visiting); err != nil {
			return err
		}
	}
	desc, err := protodesc.NewFile(file, r.registry)
	if err != nil {
		return err
	}
	return r.registry.RegisterFile(desc)
}

// idlTypes resolves Any payloads from the reflected descriptors. A payload type
// need not be imported by the service's files, so a missing one is fetched by
// name over reflection.
type idlTypes struct {
	*dynamicpb.Types
	reflector *idlReflector
	// lookupErr is a reflection failure other than an unknown payload type.
	// protojson reports it as invalid input, which it is not.
	lookupErr error
}

func (t *idlTypes) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	mt, err := t.Types.FindMessageByURL(url)
	if !errors.Is(err, protoregistry.NotFound) {
		return mt, err
	}
	name := url[strings.LastIndexByte(url, '/')+1:]
	response, err := t.reflector.query(true, name)
	if status.Code(err) == codes.NotFound {
		return nil, fmt.Errorf("resolve %s over reflection: %w", name, err)
	}
	if err == nil {
		err = t.reflector.load(response)
	}
	if err != nil {
		t.lookupErr = err
		return nil, fmt.Errorf("resolve %s over reflection: %w", name, err)
	}
	return t.Types.FindMessageByURL(url)
}
