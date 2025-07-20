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

package cli

import (
	"github.com/apache/dubbo-kubernetes/operator/pkg/util/ptr"
	"github.com/apache/dubbo-kubernetes/pkg/kube"
	"k8s.io/client-go/rest"
)

type instance struct {
	cliClient map[string]kube.CLIClient
	RootFlags
}

type Context interface {
	CLIClient() (kube.CLIClient, error)
	CLIClientWithRevision(rev string) (kube.CLIClient, error)
	Namespace() string
}

func NewCLIContext(rootFlags *RootFlags) Context {
	if rootFlags == nil {
		rootFlags = &RootFlags{
			kubeconfig: ptr.Of[string](""),
			Context:    ptr.Of[string](""),
		}
	}
	return &instance{
		RootFlags: *rootFlags,
	}
}

func (i *instance) CLIClient() (kube.CLIClient, error) {
	return i.CLIClientWithRevision("")
}

func (i *instance) CLIClientWithRevision(rev string) (kube.CLIClient, error) {
	if i.cliClient == nil {
		i.cliClient = make(map[string]kube.CLIClient)
	}

	if i.cliClient[rev] == nil {
		impersonationConfig := rest.ImpersonationConfig{}
		client, err := newKubeClientWithRevision(*i.kubeconfig, *i.Context, rev, impersonationConfig)
		if err != nil {
			return nil, err
		}
		i.cliClient[rev] = client
	}

	return i.cliClient[rev], nil
}

func (i *instance) Namespace() string {
	return "dubbo-system"
}

func newKubeClientWithRevision(kubeconfig, context, revision string, impersonationConfig rest.ImpersonationConfig) (kube.CLIClient, error) {
	drc, err := kube.DefaultRestConfig(kubeconfig, context, func(config *rest.Config) {
		config.QPS = 50
		config.Burst = 100
		config.Impersonate = impersonationConfig
	})

	if err != nil {
		return nil, err
	}

	return kube.NewCLIClient(kube.NewClientConfigForRestConfig(drc), kube.WithRevision(revision))
}
