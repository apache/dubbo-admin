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
	"errors"
	"fmt"
	"strings"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"

	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
)

const genericInvokeInstanceName = "dubbo-admin-generic-invoke"

type genericInvocation struct {
	URL            string
	Protocol       string
	Serialization  string
	ServiceName    string
	Group          string
	Version        string
	MethodName     string
	ParameterTypes []string
	Args           []hessian.Object
}

type genericInvokeTarget struct {
	instance      *meshresource.RPCInstanceResource
	protocol      string
	port          int64
	serialization string
}

var invokeGenericServiceRPC = func(callCtx context.Context, invocation genericInvocation) (any, error) {
	// TODO: Cache generic invoke clients to avoid recreating the Dubbo instance/client on every call.
	ins, err := dubbo.NewInstance(dubbo.WithName(genericInvokeInstanceName))
	if err != nil {
		return nil, err
	}

	clientOpts := []client.ClientOption{
		client.WithClientSerialization(invocation.Serialization),
	}
	switch invocation.Protocol {
	case dubboconstant.TriProtocol:
		clientOpts = append(clientOpts, client.WithClientProtocolTriple())
	case dubboconstant.DubboProtocol:
		clientOpts = append(clientOpts, client.WithClientProtocolDubbo())
	default:
		return nil, fmt.Errorf("unsupported invoke protocol %s", invocation.Protocol)
	}

	cli, err := ins.NewClient(clientOpts...)
	if err != nil {
		return nil, err
	}

	svc, err := cli.NewGenericService(
		invocation.ServiceName,
		client.WithURL(invocation.URL),
		client.WithVersion(invocation.Version),
		client.WithGroup(invocation.Group),
		client.WithProtocol(invocation.Protocol),
		client.WithSerialization(invocation.Serialization),
	)
	if err != nil {
		return nil, err
	}

	return svc.Invoke(callCtx, invocation.MethodName, invocation.ParameterTypes, invocation.Args)
}

func InvokeServiceGeneric(ctx consolectx.Context, req model.ServiceGenericInvokeReq) (*model.ServiceGenericInvokeResp, error) {
	if err := req.Validate(); err != nil {
		return nil, bizerror.New(bizerror.InvalidArgument, err.Error())
	}

	instanceRes, err := getGenericInvokeInstance(ctx, req.Mesh, req.InstanceName)
	if err != nil {
		return nil, err
	}

	metadataList, err := listServiceProviderMetadata(ctx, model.ServiceMethodsReq{
		ServiceName:     req.ServiceName,
		Group:           req.Group,
		Version:         req.Version,
		Mesh:            req.Mesh,
		ProviderAppName: instanceRes.Spec.AppName,
	})
	if err != nil {
		logger.Errorf("list service provider metadata failed, service=%s, mesh=%s, instance=%s, cause: %v",
			req.ServiceName, req.Mesh, req.InstanceName, err)
		return nil, err
	}

	parameterTypes, err := resolveServiceMethodParameterTypes(metadataList, req, instanceRes.Spec.AppName)
	if err != nil {
		return nil, err
	}
	if len(parameterTypes) != len(req.Args) {
		return nil, bizerror.New(bizerror.InvalidArgument, "resolved method parameter count does not match args length")
	}

	decodedArgs, err := decodeGenericInvokeArgs(parameterTypes, req.Args)
	if err != nil {
		return nil, bizerror.New(bizerror.InvalidArgument, err.Error())
	}

	rpcInstanceRes, err := findRPCInstanceByInstance(ctx, instanceRes)
	if err != nil {
		return nil, err
	}

	targets, err := buildGenericInvokeTargets(rpcInstanceRes)
	if err != nil {
		return nil, err
	}

	parentCtx := context.Background()
	if ctx != nil && ctx.AppContext() != nil {
		parentCtx = ctx.AppContext()
	}
	callCtx, cancel := context.WithTimeout(parentCtx, time.Duration(req.TimeoutMs)*time.Millisecond)
	defer cancel()

	if len(req.Attachments) > 0 {
		callCtx = context.WithValue(callCtx, dubboconstant.AttachmentKey, toAttachmentValues(req.Attachments))
	}

	startedAt := time.Now()
	result, err := invokeGenericServiceWithTargets(callCtx, req, targets, genericInvocation{
		ServiceName:    req.ServiceName,
		Group:          req.Group,
		Version:        req.Version,
		MethodName:     req.MethodName,
		ParameterTypes: append([]string{}, parameterTypes...),
		Args:           toHessianObjects(decodedArgs),
	})
	elapsedMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		logger.Errorf("generic invoke failed, service=%s, method=%s, instance=%s, cause: %v",
			req.ServiceName, req.MethodName, req.InstanceName, err)
		return nil, bizerror.New(bizerror.InternalError, "generic invoke failed, please check server logs")
	}

	return &model.ServiceGenericInvokeResp{
		ElapsedMs: elapsedMs,
		RawResult: result,
	}, nil
}

func getGenericInvokeInstance(
	ctx consolectx.Context,
	mesh string,
	instanceName string,
) (*meshresource.InstanceResource, error) {
	instanceRes, exists, err := manager.GetByKey[*meshresource.InstanceResource](
		ctx.ResourceManager(),
		meshresource.InstanceKind,
		coremodel.BuildResourceKey(mesh, instanceName),
	)
	if err != nil {
		logger.Errorf("get instance failed, mesh=%s, instance=%s, cause: %v", mesh, instanceName, err)
		return nil, err
	}
	if !exists || instanceRes == nil || instanceRes.Spec == nil {
		return nil, bizerror.New(bizerror.NotFoundError, fmt.Sprintf("instance %s not found", instanceName))
	}
	if instanceRes.Spec.AppName == "" || instanceRes.Spec.Ip == "" || instanceRes.Spec.RpcPort <= 0 {
		return nil, bizerror.New(
			bizerror.InvalidArgument,
			fmt.Sprintf("instance %s is not a valid rpc invoke target", instanceName),
		)
	}
	return instanceRes, nil
}

func resolveServiceMethodParameterTypes(
	metadataList []*meshresource.ServiceProviderMetadataResource,
	req model.ServiceGenericInvokeReq,
	providerAppName string,
) ([]string, error) {
	candidate, err := resolveStructuredServiceMethodCandidate(buildServiceMethodCandidates(metadataList), model.ServiceMethodDetailReq{
		ServiceMethodsReq: model.ServiceMethodsReq{
			ServiceName:     req.ServiceName,
			Group:           req.Group,
			Version:         req.Version,
			Mesh:            req.Mesh,
			ProviderAppName: providerAppName,
		},
		MethodName: req.MethodName,
		Signature:  req.Signature,
	})
	if err != nil {
		return nil, err
	}
	return append([]string{}, candidate.detail.ParameterTypes...), nil
}

func buildGenericInvokeTargets(rpcInstanceRes *meshresource.RPCInstanceResource) ([]*genericInvokeTarget, error) {
	if rpcInstanceRes == nil || rpcInstanceRes.Spec == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "rpc instance is empty")
	}
	if len(rpcInstanceRes.Spec.GetEndpoints()) == 0 {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("rpc instance %s has no available rpc endpoints", rpcInstanceRes.Spec.GetName()),
		)
	}

	serializations := buildGenericInvokeSerializations(rpcInstanceRes)
	targets := make([]*genericInvokeTarget, 0, len(rpcInstanceRes.Spec.GetEndpoints())*len(serializations))
	for _, endpoint := range rpcInstanceRes.Spec.GetEndpoints() {
		if endpoint == nil || endpoint.GetPort() <= 0 {
			continue
		}

		protocol := normalizeGenericInvokeProtocol(endpoint.GetProtocol())
		if protocol == "" {
			continue
		}

		// Try every serialization on the current endpoint before moving to the next endpoint.
		for _, serialization := range serializations {
			targets = append(targets, &genericInvokeTarget{
				instance:      rpcInstanceRes,
				protocol:      protocol,
				port:          endpoint.GetPort(),
				serialization: serialization,
			})
		}
	}
	if len(targets) == 0 {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("rpc instance %s has no supported rpc endpoints", rpcInstanceRes.Spec.GetName()),
		)
	}
	return targets, nil
}

func buildGenericInvokeSerializations(rpcInstanceRes *meshresource.RPCInstanceResource) []string {
	candidates := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)

	appendCandidate := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		candidates = append(candidates, value)
	}

	appendCandidate(rpcInstanceRes.Spec.GetSerialization())
	for _, item := range strings.Split(rpcInstanceRes.Spec.GetPreferSerialization(), ",") {
		appendCandidate(item)
	}
	if len(candidates) == 0 {
		appendCandidate(dubboconstant.Hessian2Serialization)
	}
	return candidates
}

func normalizeGenericInvokeProtocol(protocol string) string {
	switch strings.TrimSpace(strings.ToLower(protocol)) {
	case dubboconstant.TriProtocol:
		return dubboconstant.TriProtocol
	case dubboconstant.DubboProtocol:
		return dubboconstant.DubboProtocol
	default:
		return ""
	}
}

func invokeGenericServiceWithTargets(
	callCtx context.Context,
	req model.ServiceGenericInvokeReq,
	targets []*genericInvokeTarget,
	invocation genericInvocation,
) (any, error) {
	var lastErr error
	for index, target := range targets {
		if callCtx.Err() != nil {
			return nil, callCtx.Err()
		}

		// Copy the base invocation so each attempt can override transport-specific fields safely.
		attemptInvocation := invocation
		attemptInvocation.Protocol = target.protocol
		attemptInvocation.Serialization = target.serialization
		attemptInvocation.URL = buildGenericInvokeURL(target.protocol, target.instance.Spec.GetIp(), target.port)

		result, err := invokeGenericServiceRPC(callCtx, attemptInvocation)
		if err == nil {
			return result, nil
		}

		lastErr = err
		logger.Warnf(
			"generic invoke attempt failed, service=%s, method=%s, instance=%s, target=%s, protocol=%s, serialization=%s, attempt=%d/%d, cause: %v",
			req.ServiceName,
			req.MethodName,
			req.InstanceName,
			attemptInvocation.URL,
			target.protocol,
			target.serialization,
			index+1,
			len(targets),
			err,
		)
		if !isRetryableGenericInvokeError(err) {
			return nil, err
		}
	}

	if lastErr == nil {
		return nil, bizerror.New(bizerror.NotFoundError, "no generic invoke target available")
	}
	return nil, lastErr
}

func buildGenericInvokeURL(protocol string, ip string, port int64) string {
	return fmt.Sprintf("%s://%s:%d", protocol, ip, port)
}

func isRetryableGenericInvokeError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, protocolbase.ErrClientClosed) ||
		errors.Is(err, protocolbase.ErrDestroyedInvoker) ||
		errors.Is(err, protocolbase.ErrNoReply) {
		return true
	}

	message := strings.ToLower(err.Error())
	if message == "" {
		return false
	}
	for _, keyword := range []string{
		"connection refused",
		"connection reset",
		"broken pipe",
		"no such host",
		"network is unreachable",
		"i/o timeout",
		"unexpected eof",
		"eof",
		"transport",
		"codec",
		"serialization",
		"unsupported serialization",
		"no codec configured",
		"unknown compression",
		"header buffer too short",
		"body buffer too short",
		"illegal package",
		"stream error",
		"protocol error",
	} {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	// Treat all other errors as provider-side or business failures to avoid duplicate invocation.
	return false
}

func findRPCInstanceByInstance(
	ctx consolectx.Context,
	instanceRes *meshresource.InstanceResource,
) (*meshresource.RPCInstanceResource, error) {
	if instanceRes == nil || instanceRes.Spec == nil {
		return nil, bizerror.New(bizerror.InvalidArgument, "instance is empty")
	}

	rpcInstanceName := meshresource.BuildInstanceResName(
		instanceRes.Spec.AppName,
		instanceRes.Spec.Ip,
		instanceRes.Spec.RpcPort,
	)
	rpcInstanceRes, exists, err := manager.GetByKey[*meshresource.RPCInstanceResource](
		ctx.ResourceManager(),
		meshresource.RPCInstanceKind,
		coremodel.BuildResourceKey(instanceRes.Mesh, rpcInstanceName),
	)
	if err != nil {
		logger.Errorf("get rpc instance failed, mesh=%s, instance=%s, cause: %v",
			instanceRes.Mesh, instanceRes.Spec.Name, err)
		return nil, err
	}
	if exists && rpcInstanceRes != nil && rpcInstanceRes.Spec != nil {
		return rpcInstanceRes, nil
	}
	return nil, bizerror.New(
		bizerror.NotFoundError,
		fmt.Sprintf("rpc instance not found for instance %s", instanceRes.Spec.Name),
	)
}

func toAttachmentValues(attachments map[string]string) map[string]any {
	values := make(map[string]any, len(attachments))
	for key, value := range attachments {
		values[key] = value
	}
	return values
}

func toHessianObjects(args []any) []hessian.Object {
	objects := make([]hessian.Object, len(args))
	for i, arg := range args {
		objects[i] = arg
	}
	return objects
}
