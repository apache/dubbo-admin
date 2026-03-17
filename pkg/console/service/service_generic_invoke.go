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
	"fmt"
	"sort"
	"time"

	hessian "github.com/apache/dubbo-go-hessian2"

	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	dubboconstant "dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports"

	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/console/model"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/manager"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	coremodel "github.com/apache/dubbo-admin/pkg/core/resource/model"
	"github.com/apache/dubbo-admin/pkg/core/store/index"
)

const genericInvokeInstanceName = "dubbo-admin-generic-invoke"

type genericInvocation struct {
	URL            string
	ServiceName    string
	Group          string
	Version        string
	MethodName     string
	ParameterTypes []string
	Args           []hessian.Object
}

type tripleInvokeTarget struct {
	instance *meshresource.RPCInstanceResource
	port     int64
}

var invokeGenericServiceRPC = func(callCtx context.Context, invocation genericInvocation) (any, error) {
	// TODO: Cache generic invoke clients to avoid recreating the Dubbo instance/client on every call.
	ins, err := dubbo.NewInstance(dubbo.WithName(genericInvokeInstanceName))
	if err != nil {
		return nil, err
	}

	// TODO: Derive client protocol and serialization from target service metadata when expanding beyond Triple/Hessian2.
	cli, err := ins.NewClient(
		client.WithClientProtocolTriple(),
		client.WithClientSerialization(dubboconstant.Hessian2Serialization),
	)
	if err != nil {
		return nil, err
	}

	svc, err := cli.NewGenericService(
		invocation.ServiceName,
		client.WithURL(invocation.URL),
		client.WithVersion(invocation.Version),
		client.WithGroup(invocation.Group),
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

	target, err := selectTripleInvokeTarget(ctx, instanceRes)
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
	result, err := invokeGenericServiceRPC(callCtx, genericInvocation{
		URL:            fmt.Sprintf("tri://%s:%d", target.instance.Spec.Ip, target.port),
		ServiceName:    req.ServiceName,
		Group:          req.Group,
		Version:        req.Version,
		MethodName:     req.MethodName,
		ParameterTypes: append([]string{}, parameterTypes...),
		Args:           toHessianObjects(decodedArgs),
	})
	elapsedMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		logger.Errorf("generic invoke failed, service=%s, method=%s, instance=%s, target=%s:%d, cause: %v",
			req.ServiceName, req.MethodName, req.InstanceName, target.instance.Spec.Ip, target.port, err)
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

func selectTripleInvokeTarget(
	ctx consolectx.Context,
	instanceRes *meshresource.InstanceResource,
) (*tripleInvokeTarget, error) {
	rpcInstanceRes, err := findRPCInstanceByInstance(ctx, instanceRes)
	if err != nil {
		return nil, err
	}

	targets := make([]*tripleInvokeTarget, 0, len(rpcInstanceRes.Spec.GetEndpoints())+1)
	seen := make(map[string]struct{}, len(rpcInstanceRes.Spec.GetEndpoints())+1)
	if rpcInstanceRes.Spec.GetProtocol() == dubboconstant.TriProtocol && rpcInstanceRes.Spec.GetPort() > 0 {
		appendTripleInvokeTarget(&targets, seen, rpcInstanceRes, rpcInstanceRes.Spec.GetPort())
	}
	for _, endpoint := range rpcInstanceRes.Spec.GetEndpoints() {
		if endpoint == nil || endpoint.GetProtocol() != dubboconstant.TriProtocol || endpoint.GetPort() <= 0 {
			continue
		}
		appendTripleInvokeTarget(&targets, seen, rpcInstanceRes, endpoint.GetPort())
	}
	if len(targets) == 0 {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("triple instance not found for instance %s", instanceRes.Spec.Name),
		)
	}

	sort.Slice(targets, func(i, j int) bool {
		if targets[i].instance.ResourceKey() != targets[j].instance.ResourceKey() {
			return targets[i].instance.ResourceKey() < targets[j].instance.ResourceKey()
		}
		return targets[i].port < targets[j].port
	})
	return targets[0], nil
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

	instanceList, err := manager.ListByIndexes[*meshresource.RPCInstanceResource](
		ctx.ResourceManager(),
		meshresource.RPCInstanceKind,
		map[string]string{
			index.ByMeshIndex:          instanceRes.Mesh,
			index.ByRPCInstanceAppName: instanceRes.Spec.AppName,
		},
	)
	if err != nil {
		logger.Errorf("list rpc instances failed, mesh=%s, providerApp=%s, cause: %v",
			instanceRes.Mesh, instanceRes.Spec.AppName, err)
		return nil, err
	}
	for _, instance := range instanceList {
		if instance == nil || instance.Spec == nil {
			continue
		}
		if instance.Spec.GetIp() == instanceRes.Spec.Ip && instance.Spec.GetPort() == instanceRes.Spec.RpcPort {
			return instance, nil
		}
	}
	return nil, bizerror.New(
		bizerror.NotFoundError,
		fmt.Sprintf("rpc instance not found for instance %s", instanceRes.Spec.Name),
	)
}

func appendTripleInvokeTarget(targets *[]*tripleInvokeTarget, seen map[string]struct{}, instance *meshresource.RPCInstanceResource, port int64) {
	if instance == nil || instance.Spec == nil || port <= 0 {
		return
	}
	key := fmt.Sprintf("%s:%d", instance.ResourceKey(), port)
	if _, ok := seen[key]; ok {
		return
	}
	seen[key] = struct{}{}
	*targets = append(*targets, &tripleInvokeTarget{
		instance: instance,
		port:     port,
	})
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
