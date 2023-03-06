// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/dubbo-admin/pkg/authority/config"
	"github.com/apache/dubbo-admin/pkg/authority/k8s"
	"github.com/apache/dubbo-admin/pkg/authority/rule"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

func exactEndpoint(c context.Context, options *config.Options, kubeClient k8s.Client) (*rule.Endpoint, error) {
	if options.IsKubernetesConnected && options.EnableOIDCCheck {
		md, ok := metadata.FromIncomingContext(c)
		if !ok {
			return nil, fmt.Errorf("failed to get metadata from context")
		}

		authorization, ok := md["authorization"]
		if !ok || len(authorization) != 1 {
			return nil, fmt.Errorf("failed to get Authorization header from context")
		}

		if !strings.HasPrefix(authorization[0], "Bearer ") {
			return nil, fmt.Errorf("failed to get Authorization header from context")
		}

		token := strings.ReplaceAll(authorization[0], "Bearer ", "")

		endpoint, ok := kubeClient.VerifyServiceAccount(token)
		if !ok {
			return nil, fmt.Errorf("failed to verify Authorization header from kubernetes")
		}

		return endpoint, nil
	}
	// TODO get from SSL context

	p, ok := peer.FromContext(c)
	if !ok {
		return nil, fmt.Errorf("failed to get peer from context")
	}

	// TODO get ip
	return &rule.Endpoint{
		ID:  p.Addr.String(),
		Ips: []string{p.Addr.String()},
	}, nil
}
