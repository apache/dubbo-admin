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

package bootstrap

import (
	"context"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/cert/provider"
	"github.com/apache/dubbo-admin/pkg/core/election/kube"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	core_runtime "github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/runtime/component"
	"github.com/apache/dubbo-admin/pkg/cp-server/server"
)

func buildRuntime(appCtx context.Context, cfg *dubbo_cp.Config) (core_runtime.Runtime, error) {
	builder, err := core_runtime.BuilderFor(appCtx, cfg)
	if err != nil {
		return nil, err
	}

	if err := initKubuClient(cfg, builder); err != nil {
		return nil, err
	}

	if cfg.KubeConfig.IsKubernetesConnected == false {
		rt, err := builder.Build()
		if err != nil {
			return nil, err
		}
		return rt, nil
	}

	if err := initCertStorage(cfg, builder); err != nil {
		return nil, err
	}

	if err := initGrpcServer(cfg, builder); err != nil {
		return nil, err
	}

	builder.WithComponentManager(component.NewManager(kube.NewLeaderElection(builder.Config().KubeConfig.Namespace,
		builder.Config().KubeConfig.ServiceName,
		"dubbo-cp-lock",
		builder.KubuClient().GetKubClient())))
	rt, err := builder.Build()
	if err != nil {
		return nil, err
	}
	return rt, nil
}

func Bootstrap(appCtx context.Context, cfg *dubbo_cp.Config) (core_runtime.Runtime, error) {
	runtime, err := buildRuntime(appCtx, cfg)
	if err != nil {
		return nil, err
	}
	return runtime, nil
}

func initKubuClient(cfg *dubbo_cp.Config, builder *core_runtime.Builder) error {
	client := provider.NewClient()
	if !client.Init(cfg) {
		logger.Sugar().Warnf("Failed to connect to Kubernetes cluster. Will ignore OpenID Connect check.")
		cfg.KubeConfig.IsKubernetesConnected = false
	} else {
		cfg.KubeConfig.IsKubernetesConnected = true
	}
	builder.WithKubuClient(client)
	return nil
}

func initCertStorage(cfg *dubbo_cp.Config, builder *core_runtime.Builder) error {
	storage := provider.NewStorage(cfg, builder.KubuClient())
	loadRootCert()
	loadAuthorityCert(storage, cfg, builder)

	storage.GetServerCert("localhost")
	storage.GetServerCert("dubbo-ca." + storage.GetConfig().KubeConfig.Namespace + ".svc")
	storage.GetServerCert("dubbo-ca." + storage.GetConfig().KubeConfig.Namespace + ".svc.cluster.local")
	builder.WithCertStorage(storage)
	return nil
}

func loadRootCert() {
	// TODO
}

func loadAuthorityCert(storage provider.Storage, cfg *dubbo_cp.Config, builder *core_runtime.Builder) {
	if cfg.KubeConfig.IsKubernetesConnected {
		certStr, priStr := builder.KubuClient().GetAuthorityCert(cfg.KubeConfig.Namespace)
		if certStr != "" && priStr != "" {
			storage.GetAuthorityCert().Cert = provider.DecodeCert(certStr)
			storage.GetAuthorityCert().CertPem = certStr
			storage.GetAuthorityCert().PrivateKey = provider.DecodePrivateKey(priStr)
		}
	}
	refreshAuthorityCert(storage, cfg)
}

func refreshAuthorityCert(storage provider.Storage, cfg *dubbo_cp.Config) {
	if storage.GetAuthorityCert().IsValid() {
		logger.Sugar().Infof("Load authority cert from kubernetes secrect success.")
	} else {
		logger.Sugar().Warnf("Load authority cert from kubernetes secrect failed.")
		storage.SetAuthorityCert(provider.GenerateAuthorityCert(storage.GetRootCert(), cfg.Security.CaValidity))

		// TODO lock if multi cp-server
		if storage.GetConfig().KubeConfig.IsKubernetesConnected {
			storage.GetKubuClient().UpdateAuthorityCert(storage.GetAuthorityCert().CertPem,
				provider.EncodePrivateKey(storage.GetAuthorityCert().PrivateKey), storage.GetConfig().KubeConfig.Namespace)
		}
	}

	if storage.GetConfig().KubeConfig.IsKubernetesConnected {
		logger.Sugar().Info("Writing ca to config maps.")
		if storage.GetKubuClient().UpdateAuthorityPublicKey(storage.GetAuthorityCert().CertPem) {
			logger.Sugar().Info("Write ca to config maps success.")
		} else {
			logger.Sugar().Warnf("Write ca to config maps failed.")
		}
	}

	storage.AddTrustedCert(storage.GetAuthorityCert())
}

func initGrpcServer(cfg *dubbo_cp.Config, builder *core_runtime.Builder) error {
	grpcServer := server.NewGrpcServer(builder.CertStorage(), cfg)
	builder.WithGrpcServer(grpcServer)
	return nil
}
