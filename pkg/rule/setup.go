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

package rule

import (
	"time"

	"github.com/apache/dubbo-admin/api/mesh"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	core_runtime "github.com/apache/dubbo-admin/pkg/core/runtime"
	informers "github.com/apache/dubbo-admin/pkg/rule/clientgen/informers/externalversions"
	"github.com/apache/dubbo-admin/pkg/rule/crd"
	"github.com/apache/dubbo-admin/pkg/rule/crd/authentication"
	"github.com/apache/dubbo-admin/pkg/rule/crd/authorization"
	"github.com/apache/dubbo-admin/pkg/rule/crd/conditionroute"
	"github.com/apache/dubbo-admin/pkg/rule/crd/dynamicconfig"
	"github.com/apache/dubbo-admin/pkg/rule/crd/servicemapping"
	"github.com/pkg/errors"

	"github.com/apache/dubbo-admin/pkg/rule/crd/tagroute"
	"github.com/apache/dubbo-admin/pkg/rule/server"
	"github.com/apache/dubbo-admin/pkg/rule/storage"
)

func Setup(rt core_runtime.Runtime) error {
	ruleServer := server.NewRuleServer(rt.Config())
	ruleServer.CertStorage = rt.CertStorage()
	ruleServer.KubeClient = rt.KubuClient()
	ruleServer.Storage = storage.NewStorage()
	if err := RegisterController(ruleServer); err != nil {
		return errors.Wrap(err, "Controller Register failed")
	}
	if err := RegisterObserveService(rt, ruleServer); err != nil {
		return errors.Wrap(err, "RuleService Register failed")
	}
	if err := rt.Add(ruleServer); err != nil {
		return errors.Wrap(err, "RuleServer component add failed")
	}
	return nil
}

func RegisterController(s *server.RuleServer) error {
	logger.Sugar().Info("Init rule controller...")
	informerFactory := informers.NewSharedInformerFactory(s.KubeClient.GetInformerClient(), time.Second*30)
	s.InformerFactory = informerFactory
	authenticationHandler := authentication.NewHandler(s.Storage)
	authorizationHandler := authorization.NewHandler(s.Storage)
	serviceMappingHandler := servicemapping.NewHandler(s.Storage)
	conditionRouteHandler := conditionroute.NewHandler(s.Storage)
	tagRouteHandler := tagroute.NewHandler(s.Storage)
	dynamicConfigHandler := dynamicconfig.NewHandler(s.Storage)
	controller := crd.NewController(s.Options.KubeConfig.Namespace,
		authenticationHandler,
		authorizationHandler,
		serviceMappingHandler,
		tagRouteHandler,
		conditionRouteHandler,
		dynamicConfigHandler,
		informerFactory.Dubbo().V1beta1().AuthenticationPolicies(),
		informerFactory.Dubbo().V1beta1().AuthorizationPolicies(),
		informerFactory.Dubbo().V1beta1().ServiceNameMappings(),
		informerFactory.Dubbo().V1beta1().TagRoutes(),
		informerFactory.Dubbo().V1beta1().ConditionRoutes(),
		informerFactory.Dubbo().V1beta1().DynamicConfigs(),
	)
	s.Controller = controller
	return nil
}

func RegisterObserveService(rt core_runtime.Runtime, service *server.RuleServer) error {
	mesh.RegisterRuleServiceServer(rt.GrpcServer().PlainServer, service)
	mesh.RegisterRuleServiceServer(rt.GrpcServer().SecureServer, service)
	return nil
}
