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

package client

import (
	"os"
	"path/filepath"

	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type KubeClient interface {
	GetKubernetesClientSet() *kubernetes.Clientset
}

type kubeClientImpl struct {
	kubernetesClientSet *kubernetes.Clientset
}

func NewKubuClient() *kubeClientImpl {
	return &kubeClientImpl{}
}

func (k *kubeClientImpl) GetKubernetesClientSet() *kubernetes.Clientset {
	return k.kubernetesClientSet
}

func (k *kubeClientImpl) Init(options *dubbo_cp.Config) bool {
	config, err := rest.InClusterConfig()
	options.KubeConfig.InPodEnv = err == nil
	kubeconfig := options.KubeConfig.KubeFileConfig
	if err != nil {
		logger.Sugar().Infof("Failed to load config from Pod. Will fall back to kube config file.")
		// Read kubeconfig from command line
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
			return false
		}
	}

	// set qps and burst for rest config
	config.QPS = float32(options.KubeConfig.RestConfigQps)
	config.Burst = options.KubeConfig.RestConfigBurst
	// creates the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create clientgen to kubernetes. " + err.Error())
		return false
	}
	if err != nil {
		logger.Sugar().Warnf("Failed to create clientgen to kubernetes. " + err.Error())
		return false
	}
	k.kubernetesClientSet = clientSet
	return true
}
