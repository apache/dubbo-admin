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
	informerclient "github.com/apache/dubbo-admin/pkg/rule/clientgen/clientset/versioned"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
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
	ruleServer.CertClient = rt.CertStorage().GetCertClient()
	ruleServer.Storage = storage.NewStorage()
	if err := initInformerClient(rt, ruleServer); err != nil {
		return errors.Wrap(err, "InformerClient Register failed")
	}
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

func initInformerClient(rt core_runtime.Runtime, server *server.RuleServer) error {
	config, err := rest.InClusterConfig()
	rt.Config().KubeConfig.InPodEnv = err == nil
	kubeconfig := rt.Config().KubeConfig.KubeFileConfig
	if err != nil {
		logger.Sugar().Infof("Failed to load config from Pod. Will fall back to kube config file.")
		if len(kubeconfig) <= 0 {
			// Read kubeconfig from env
			kubeconfig = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
			if len(kubeconfig) <= 0 {
				// Read kubeconfig from home dir
				if home := homedir.HomeDir(); home != "" {
					kubeconfig = filepath.Join(home, ".kube", "config")
				}
			}
		}
		// use the current context in kubeconfig
		logger.Sugar().Infof("Read kubeconfig from %s", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Sugar().Warnf("Failed to load config from kube config file.")
			return err
		}
	}
	// set qps and burst for rest config
	config.QPS = float32(rt.Config().KubeConfig.RestConfigQps)
	config.Burst = rt.Config().KubeConfig.RestConfigBurst
	informerClient, err := informerclient.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create informerclient to kubernetes. " + err.Error())
		return err
	}
	server.InformerClient = informerClient
	return nil
}

func RegisterController(s *server.RuleServer) error {
	logger.Sugar().Info("Init rule controller...")
	informerFactory := informers.NewSharedInformerFactory(s.InformerClient, time.Second*30)
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
