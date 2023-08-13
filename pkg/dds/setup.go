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

package dds

import (
	"github.com/apache/dubbo-admin/api/dds"
	core_runtime "github.com/apache/dubbo-admin/pkg/core/runtime"
	"github.com/apache/dubbo-admin/pkg/core/schema/collections"
	"github.com/apache/dubbo-admin/pkg/dds/kube/crdclient"
	"github.com/apache/dubbo-admin/pkg/dds/server"
	"github.com/apache/dubbo-admin/pkg/dds/storage"
	"github.com/pkg/errors"
)

func Setup(rt core_runtime.Runtime) error {
	if !rt.Config().KubeConfig.IsKubernetesConnected {
		return nil
	}
	cache, err := crdclient.New(rt.KubeClient(), rt.Config().KubeConfig.DomainSuffix)
	if err != nil {
		return errors.Wrap(err, "crd client New error")
	}
	ddsServer := server.NewRuleServer(rt.Config(), cache)
	ddsServer.CertStorage = rt.CertStorage()
	ddsServer.Storage = storage.NewStorage(rt.Config())
	ddsServer.CertClient = rt.CertClient()

	schemas := collections.Rule.All()
	for _, schema := range schemas {
		cache.RegisterEventHandler(schema.Resource().GroupVersionKind(), crdclient.EventHandler{
			Resource: crdclient.NewHandler(ddsServer.Storage, rt.Config().KubeConfig.Namespace, cache),
		})
	}
	if err := RegisterObserveService(rt, ddsServer); err != nil {
		return errors.Wrap(err, "RuleService Register failed")
	}
	if err := rt.Add(ddsServer); err != nil {
		return errors.Wrap(err, "RuleServer component add failed")
	}
	return nil
}

func RegisterObserveService(rt core_runtime.Runtime, service *server.DdsServer) error {
	dds.RegisterRuleServiceServer(rt.GrpcServer().PlainServer, service)
	dds.RegisterRuleServiceServer(rt.GrpcServer().SecureServer, service)
	return nil
}
