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
	"testing"

	meshproto "github.com/apache/dubbo-admin/api/mesh/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/common/bizerror"
	meshresource "github.com/apache/dubbo-admin/pkg/core/resource/apis/mesh/v1alpha1"
	"github.com/stretchr/testify/require"
)

func TestGenericInvokeSignatureCandidatesRespectProviderMode(t *testing.T) {
	for _, mode := range []string{"IDL", "non-IDL"} {
		t.Run(mode, func(t *testing.T) {
			ctx, req := idlTestCatalog(t, 1)
			providers, err := listProviderMeta(ctx, req.BaseServiceReq)
			require.NoError(t, err)
			provider := providers[0]
			provider.Spec.Parameters["IDL-mode"] = mode
			foreignMode := "non-IDL"
			foreignType := "java.lang.String"
			if mode == "non-IDL" {
				provider.Spec.Methods[0].ParameterTypes = []string{"java.lang.String"}
				provider.Spec.Methods[0].ReturnType = "java.lang.String"
				foreignMode = "IDL"
				foreignType = "grpc.health.v1.HealthCheckRequest"
			}
			require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Update(provider))
			expected := methodSig(provider.Spec.Methods[0])

			for _, foreignProviderMode := range []string{foreignMode, "unknown"} {
				foreign := meshresource.NewServiceProviderMetadataResourceWithAttributes("other-"+foreignProviderMode, req.Mesh)
				foreign.Spec.ServiceName = req.ServiceName
				foreign.Spec.ProviderAppName = "other-app"
				foreign.Spec.Parameters = map[string]string{"IDL-mode": foreignProviderMode}
				foreign.Spec.Methods = []*meshproto.Method{{Name: req.MethodName, ParameterTypes: []string{foreignType}, ReturnType: foreignType}}
				require.NoError(t, ctx.stores[meshresource.ServiceProviderMetadataKind].Add(foreign))
			}
			req.Signature = "int->int"
			_, err = InvokeServiceGeneric(ctx, req)
			var fieldErr *GenericInvokeArgumentError
			require.ErrorAs(t, err, &fieldErr)
			require.Equal(t, ArgumentErrorSignature, fieldErr.Kind)
			require.Equal(t, "signature", fieldErr.Field)
			require.Equal(t, expected, fieldErr.Expected)
			require.Equal(t, req.Signature, fieldErr.Actual)
			var bizErr bizerror.Error
			require.ErrorAs(t, err, &bizErr)
			require.Equal(t, bizerror.NotFoundError, bizErr.Code())
		})
	}
}

func TestIDLArgumentCountPreservesStructuredError(t *testing.T) {
	ctx, req := idlTestCatalog(t, 1)
	req.Args = nil
	_, err := InvokeServiceGeneric(ctx, req)
	var fieldErr *GenericInvokeArgumentError
	require.ErrorAs(t, err, &fieldErr)
	require.Equal(t, ArgumentErrorCount, fieldErr.Kind)
	require.Equal(t, "args", fieldErr.Field)
	require.Equal(t, "1", fieldErr.Expected)
	require.Equal(t, "0", fieldErr.Actual)
	var bizErr bizerror.Error
	require.ErrorAs(t, err, &bizErr)
	require.Equal(t, bizerror.InvalidArgument, bizErr.Code())
}
