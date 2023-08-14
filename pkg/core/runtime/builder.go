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

package runtime

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/apache/dubbo-admin/pkg/core/kubeclient/client"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"github.com/apache/dubbo-admin/pkg/cp-server/server"
	"github.com/pkg/errors"
)

type BuilderContext interface {
	ComponentManager() component.Manager
	Config() *dubbo_cp.Config
	CertStorage() *provider.CertStorage
	KubeClient() *client.KubeClient
	CertClient() provider.Client
}

var _ BuilderContext = &Builder{}

type Builder struct {
	cfg    *dubbo_cp.Config
	cm     component.Manager
	appCtx context.Context

	kubeClient  *client.KubeClient
	grpcServer  *server.GrpcServer
	certStorage *provider.CertStorage
	certClient  provider.Client
	*runtimeInfo
}

func (b *Builder) CertClient() provider.Client {
	return b.certClient
}

func (b *Builder) KubeClient() *client.KubeClient {
	return b.kubeClient
}

func (b *Builder) CertStorage() *provider.CertStorage {
	return b.certStorage
}

func (b *Builder) Config() *dubbo_cp.Config {
	return b.cfg
}

func (b *Builder) ComponentManager() component.Manager {
	return b.cm
}

func BuilderFor(appCtx context.Context, cfg *dubbo_cp.Config) (*Builder, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, errors.Wrap(err, "could not get hostname")
	}
	suffix := core.NewUUID()[0:4]
	return &Builder{
		cfg:    cfg,
		appCtx: appCtx,
		runtimeInfo: &runtimeInfo{
			instanceId: fmt.Sprintf("%s-%s", hostname, suffix),
			startTime:  time.Now(),
		},
	}, nil
}

func (b *Builder) Build() (Runtime, error) {
	if b.grpcServer == nil {
		return nil, errors.Errorf("grpcServer has not been configured")
	}
	if b.certStorage == nil {
		return nil, errors.Errorf("certStorage has not been configured")
	}
	return &runtime{
		RuntimeInfo: b.runtimeInfo,
		RuntimeContext: &runtimeContext{
			cfg:         b.cfg,
			grpcServer:  b.grpcServer,
			certStorage: b.certStorage,
			kubeClient:  b.kubeClient,
			certClient:  b.certClient,
		},
		Manager: b.cm,
	}, nil
}

func (b *Builder) WithCertClient(certClient provider.Client) *Builder {
	b.certClient = certClient
	return b
}

func (b *Builder) WithKubeClient(kubeClient *client.KubeClient) *Builder {
	b.kubeClient = kubeClient
	return b
}

func (b *Builder) WithCertStorage(storage *provider.CertStorage) *Builder {
	b.certStorage = storage
	return b
}

func (b *Builder) WithGrpcServer(grpcServer server.GrpcServer) *Builder {
	b.grpcServer = &grpcServer
	return b
}

func (b *Builder) WithComponentManager(cm component.Manager) *Builder {
	b.cm = cm
	return b
}
