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

var invokeGenericServiceRPC = func(callCtx context.Context, invocation genericInvocation) (any, error) {
	ins, err := dubbo.NewInstance(dubbo.WithName(genericInvokeInstanceName))
	if err != nil {
		return nil, err
	}

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

	metadataList, err := listServiceProviderMetadata(ctx, model.ServiceMethodsReq{
		ServiceName:     req.ServiceName,
		Group:           req.Group,
		Version:         req.Version,
		Mesh:            req.Mesh,
		ProviderAppName: req.ProviderAppName,
	})
	if err != nil {
		logger.Errorf("list service provider metadata failed, service=%s, mesh=%s, cause: %v", req.ServiceName, req.Mesh, err)
		return nil, err
	}

	parameterTypes, err := resolveServiceMethodParameterTypes(metadataList, req)
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

	providerAppName, err := selectServiceProviderAppName(req, metadataList)
	if err != nil {
		return nil, err
	}

	instance, err := selectTripleRPCInstance(ctx, req.Mesh, providerAppName)
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
		URL:            fmt.Sprintf("tri://%s:%d", instance.Spec.Ip, instance.Spec.Port),
		ServiceName:    req.ServiceName,
		Group:          req.Group,
		Version:        req.Version,
		MethodName:     req.MethodName,
		ParameterTypes: append([]string{}, parameterTypes...),
		Args:           toHessianObjects(decodedArgs),
	})
	elapsedMs := time.Since(startedAt).Milliseconds()
	if err != nil {
		logger.Errorf("generic invoke failed, service=%s, method=%s, providerApp=%s, target=%s:%d, cause: %v",
			req.ServiceName, req.MethodName, providerAppName, instance.Spec.Ip, instance.Spec.Port, err)
		return nil, bizerror.New(bizerror.InternalError, err.Error())
	}

	return &model.ServiceGenericInvokeResp{
		ElapsedMs: elapsedMs,
		RawResult: result,
	}, nil
}

func resolveServiceMethodParameterTypes(metadataList []*meshresource.ServiceProviderMetadataResource, req model.ServiceGenericInvokeReq) ([]string, error) {
	candidate, err := resolveStructuredServiceMethodCandidate(buildServiceMethodCandidates(metadataList), model.ServiceMethodDetailReq{
		ServiceMethodsReq: model.ServiceMethodsReq{
			ServiceName:     req.ServiceName,
			Group:           req.Group,
			Version:         req.Version,
			Mesh:            req.Mesh,
			ProviderAppName: req.ProviderAppName,
		},
		MethodName: req.MethodName,
		Signature:  req.Signature,
	})
	if err != nil {
		return nil, err
	}
	return append([]string{}, candidate.detail.ParameterTypes...), nil
}

func selectServiceProviderAppName(req model.ServiceGenericInvokeReq, metadataList []*meshresource.ServiceProviderMetadataResource) (string, error) {
	if len(metadataList) == 0 {
		return "", bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("provider metadata not found for service %s", req.ServiceName),
		)
	}

	if req.ProviderAppName != "" {
		return req.ProviderAppName, nil
	}

	providerAppNames := make([]string, 0, len(metadataList))
	providerAppNameSet := make(map[string]struct{}, len(metadataList))
	for _, metadata := range metadataList {
		if metadata == nil || metadata.Spec == nil {
			continue
		}
		providerAppName := metadata.Spec.ProviderAppName
		if providerAppName == "" {
			continue
		}
		if _, exists := providerAppNameSet[providerAppName]; exists {
			continue
		}
		providerAppNameSet[providerAppName] = struct{}{}
		providerAppNames = append(providerAppNames, providerAppName)
	}
	if len(providerAppNames) == 0 {
		return "", bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("provider app not found for service %s", req.ServiceName),
		)
	}

	sort.Strings(providerAppNames)
	return providerAppNames[0], nil
}

func selectTripleRPCInstance(ctx consolectx.Context, mesh string, providerAppName string) (*meshresource.RPCInstanceResource, error) {
	instanceList, err := manager.ListByIndexes[*meshresource.RPCInstanceResource](
		ctx.ResourceManager(),
		meshresource.RPCInstanceKind,
		map[string]string{
			index.ByMeshIndex:          mesh,
			index.ByRPCInstanceAppName: providerAppName,
		},
	)
	if err != nil {
		logger.Errorf("list rpc instances failed, mesh=%s, providerApp=%s, cause: %v", mesh, providerAppName, err)
		return nil, err
	}

	tripleInstances := make([]*meshresource.RPCInstanceResource, 0, len(instanceList))
	for _, instance := range instanceList {
		if instance == nil || instance.Spec == nil {
			continue
		}
		if instance.Spec.Protocol != dubboconstant.TriProtocol {
			continue
		}
		tripleInstances = append(tripleInstances, instance)
	}
	if len(tripleInstances) == 0 {
		return nil, bizerror.New(
			bizerror.NotFoundError,
			fmt.Sprintf("triple instance not found for provider %s", providerAppName),
		)
	}

	sort.Slice(tripleInstances, func(i, j int) bool {
		return tripleInstances[i].ResourceKey() < tripleInstances[j].ResourceKey()
	})
	return tripleInstances[0], nil
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
	for index, arg := range args {
		objects[index] = arg
	}
	return objects
}
