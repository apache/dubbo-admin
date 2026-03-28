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
	"reflect"
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
	instanceRes, err := getGenericInvokeInstance(ctx, req.Mesh, req.InstanceName)
	if err != nil {
		return nil, err
	}

	metadataList, err := listProviderMeta(ctx, req.BaseServiceReq)
	if err != nil {
		return nil, err
	}
	resolvedMethod := findMethod(metadataList, req.MethodName, req.Signature)
	if resolvedMethod == nil {
		err := bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("method %s not found for service %s", req.MethodName, req.ServiceName),
		)
		logger.Errorf("resolve service method failed, service=%s, mesh=%s, instance=%s, method=%s, cause: %v",
			req.ServiceName, req.Mesh, req.InstanceName, req.MethodName, err)
		return nil, err
	}

	parameterTypes := resolvedMethod.GetParameterTypes()
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
		attachmentValues := make(map[string]any, len(req.Attachments))
		for key, value := range req.Attachments {
			attachmentValues[key] = value
		}
		callCtx = context.WithValue(callCtx, dubboconstant.AttachmentKey, attachmentValues)
	}

	hessianArgs := make([]hessian.Object, len(decodedArgs))
	for i, arg := range decodedArgs {
		hessianArgs[i] = arg
	}

	startedAt := time.Now()
	result, err := invokeGenericServiceWithTargets(callCtx, req, targets, genericInvocation{
		ServiceName:    req.ServiceName,
		Group:          req.Group,
		Version:        req.Version,
		MethodName:     req.MethodName,
		ParameterTypes: parameterTypes,
		Args:           hessianArgs,
	})
	elapsedMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		logger.Errorf("generic invoke failed, service=%s, method=%s, instance=%s, cause: %v",
			req.ServiceName, req.MethodName, req.InstanceName, err)
		return nil, bizerror.New(bizerror.InternalError, "generic invoke failed, please check server logs")
	}

	return &model.ServiceGenericInvokeResp{
		ElapsedMs: elapsedMs,
		RawResult: normalizeJSONValue(reflect.ValueOf(result)),
	}, nil
}

func normalizeJSONValue(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}

	for value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if value.CanInterface() {
		if _, ok := value.Interface().(json.Marshaler); ok {
			return value.Interface()
		}
	}

	switch value.Kind() {
	case reflect.Map:
		normalized := make(map[string]any, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			normalized[fmt.Sprint(normalizeJSONValue(iter.Key()))] = normalizeJSONValue(iter.Value())
		}
		return normalized
	case reflect.Slice, reflect.Array:
		normalized := make([]any, value.Len())
		for index := 0; index < value.Len(); index++ {
			normalized[index] = normalizeJSONValue(value.Index(index))
		}
		return normalized
	case reflect.Struct:
		return normalizeJSONStruct(value)
	default:
		if value.CanInterface() {
			return value.Interface()
		}
		return fmt.Sprint(value)
	}
}

func normalizeJSONStruct(value reflect.Value) map[string]any {
	structType := value.Type()
	normalized := make(map[string]any, structType.NumField())
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		if field.PkgPath != "" {
			continue
		}

		fieldName := field.Name
		if jsonTag := field.Tag.Get("json"); jsonTag != "" {
			tagName := strings.TrimSpace(strings.Split(jsonTag, ",")[0])
			if tagName == "-" {
				continue
			}
			if tagName != "" {
				fieldName = tagName
			}
		}

		normalized[fieldName] = normalizeJSONValue(value.Field(index))
	}
	return normalized
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
	if len(serializations) == 0 {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("rpc instance %s has no available serialization metadata", rpcInstanceRes.Spec.GetName()),
		)
	}
	targets := make([]*genericInvokeTarget, 0, len(rpcInstanceRes.Spec.GetEndpoints())*len(serializations))
	for _, endpoint := range rpcInstanceRes.Spec.GetEndpoints() {

		// Try every serialization on the current endpoint before moving to the next endpoint.
		for _, serialization := range serializations {
			targets = append(targets, &genericInvokeTarget{
				instance:      rpcInstanceRes,
				protocol:      endpoint.GetProtocol(),
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
		attemptInvocation.URL = fmt.Sprintf("%s://%s:%d", target.protocol, target.instance.Spec.GetIp(), target.port)

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

	return nil, lastErr
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
