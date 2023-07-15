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
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"github.com/apache/dubbo-admin/pkg/cp-server/server"
	"github.com/pkg/errors"
	"os"
	"time"
)

type BuilderContext interface {
	ComponentManager() component.Manager
	Config() *dubbo_cp.Config
	CertStorage() provider.Storage
	KubuClient() provider.Client
}

var _ BuilderContext = &Builder{}

type Builder struct {
	cfg    *dubbo_cp.Config
	cm     component.Manager
	appCtx context.Context

	grpcServer  *server.GrpcServer
	kubuClient  provider.Client
	certStorage provider.Storage
	*runtimeInfo
}

func (b *Builder) KubuClient() provider.Client {
	return b.kubuClient
}

func (b *Builder) CertStorage() provider.Storage {
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
	if b.kubuClient == nil {
		return nil, errors.Errorf("kubuClient has not been configured")
	}
	return &runtime{
		RuntimeInfo: b.runtimeInfo,
		RuntimeContext: &runtimeContext{
			cfg:         b.cfg,
			grpcServer:  b.grpcServer,
			certStorage: b.certStorage,
			kubuClient:  b.kubuClient,
		},
		Manager: b.cm,
	}, nil
}

func (b *Builder) WithCertStorage(storage provider.Storage) *Builder {
	b.certStorage = storage
	return b
}

func (b *Builder) WithKubuClient(client provider.Client) *Builder {
	b.kubuClient = client
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
