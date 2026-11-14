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
	"fmt"
	"io"
	"net"
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
	files, err := idlDescriptors(ctx, cc, req.ServiceName)
	if err != nil {
		return nil, idlCallError(ctx, "IDL reflection failed", err)
	}
	desc, err := files.FindDescriptorByName(protoreflect.FullName(req.ServiceName))
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
	input := dynamicpb.NewMessage(method.Input())
	// Each args element is the protobuf JSON document, not a Hessian class wrapper.
	if err = (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(req.Args[0], input); err != nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "invalid protobuf request JSON: "+err.Error())
	}
	output := dynamicpb.NewMessage(method.Output())
	if err = cc.Invoke(invokeCtx, "/"+string(svc.FullName())+"/"+string(method.Name()), input, output); err != nil {
		return nil, idlCallError(ctx, "IDL unary invocation failed", err)
	}
	encoded, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(output)
	if err != nil {
		return nil, idlCallError(ctx, "IDL response encoding failed", err)
	}
	var result any
	if err = json.Unmarshal(encoded, &result); err != nil {
		return nil, idlCallError(ctx, "IDL response encoding failed", err)
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

func idlDescriptors(ctx context.Context, cc *grpc.ClientConn, service string) (*protoregistry.Files, error) {
	const maxFiles = 256
	const maxBytes = 8 << 20
	const maxRounds = 256
	response, err := idlReflectionQuery(ctx, cc, false, true, service)
	alpha := false
	if status.Code(err) == codes.Unimplemented {
		alpha = true
		response, err = idlReflectionQuery(ctx, cc, true, true, service)
	}
	if err != nil {
		if status.Code(err) == codes.Unimplemented {
			return nil, fmt.Errorf("reflection unsupported: %w", err)
		}
		return nil, err
	}
	files := make(map[string]*descriptorpb.FileDescriptorProto)
	fileCount, byteCount := 0, 0
	add := func(response *reflectionv1.ServerReflectionResponse) error {
		data := response.GetFileDescriptorResponse()
		if data == nil || len(data.FileDescriptorProto) == 0 {
			return fmt.Errorf("reflection returned no descriptors")
		}
		for _, raw := range data.FileDescriptorProto {
			fileCount++
			byteCount += len(raw)
			if fileCount > maxFiles || byteCount > maxBytes {
				return fmt.Errorf("reflection descriptor limit exceeded")
			}
			file := new(descriptorpb.FileDescriptorProto)
			if err := proto.Unmarshal(raw, file); err != nil {
				return fmt.Errorf("invalid descriptor: %w", err)
			}
			if file.GetName() == "" {
				return fmt.Errorf("descriptor has no filename")
			}
			if old, ok := files[file.GetName()]; ok && !proto.Equal(old, file) {
				return fmt.Errorf("conflicting descriptor %s", file.GetName())
			}
			files[file.GetName()] = file
		}
		return nil
	}
	if err = add(response); err != nil {
		return nil, err
	}
	for rounds := 1; ; rounds++ {
		missing := ""
		for _, file := range files {
			for _, dep := range file.Dependency {
				if _, ok := files[dep]; !ok {
					missing = dep
					break
				}
			}
			if missing != "" {
				break
			}
		}
		if missing == "" {
			break
		}
		if rounds >= maxRounds {
			return nil, fmt.Errorf("reflection fetch round limit exceeded")
		}
		response, err = idlReflectionQuery(ctx, cc, alpha, false, missing)
		if err != nil {
			return nil, fmt.Errorf("missing dependency %s: %w", missing, err)
		}
		if err = add(response); err != nil {
			return nil, fmt.Errorf("dependency %s: %w", missing, err)
		}
		if _, ok := files[missing]; !ok {
			return nil, fmt.Errorf("reflection did not return dependency %s", missing)
		}
	}
	set := &descriptorpb.FileDescriptorSet{}
	for _, file := range files {
		set.File = append(set.File, file)
	}
	registry, err := protodesc.NewFiles(set)
	if err != nil {
		return nil, fmt.Errorf("incomplete or invalid descriptors: %w", err)
	}
	return registry, nil
}
