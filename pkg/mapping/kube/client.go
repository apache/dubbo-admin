/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kube

import (
	"flag"
	"fmt"
	"github.com/apache/dubbo-admin/pkg/logger"
	corev1 "github.com/apache/dubbo-admin/pkg/mapping/apis/dubbo.apache.org/v1alpha1"
	"github.com/apache/dubbo-admin/pkg/mapping/config"
	clientsetClient "github.com/apache/dubbo-admin/pkg/mapping/generated/clientset/versioned"
	informers "github.com/apache/dubbo-admin/pkg/mapping/generated/informers/externalversions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"time"
)

var kubeconfig string

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "",
		"Paths to a kubeconfig. Only required if out-of-cluster.")
}

type Client interface {
	Init(options *config.Options) bool
	Admin() clientsetClient.Interface
	InitContainer()
}

type client struct {
	options        *config.Options
	kube           kubernetes.Interface
	informerClient *clientsetClient.Clientset
	kubeClient     *kubernetes.Clientset
}

func NewClient() Client {
	return &client{}
}

func (c *client) Admin() clientsetClient.Interface {
	return c.informerClient
}

func (c *client) Init(options *config.Options) bool {
	c.options = options
	config, err := rest.InClusterConfig()
	if err != nil {
		if len(kubeconfig) <= 0 {
			kubeconfig = os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
			if len(kubeconfig) <= 0 {
				if home := homedir.HomeDir(); home != "" {
					kubeconfig = filepath.Join(home, ".kube", "config")
				}
			}
		}
		logger.Sugar().Infof("Read kubeconfig from %s", kubeconfig)
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			logger.Sugar().Warnf("Failed to load config from kube config file.")
			return false
		}
	}
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create client to kubernetes. " + err.Error())
		return false
	}
	informerClient, err := clientsetClient.NewForConfig(config)
	if err != nil {
		logger.Sugar().Warnf("Failed to create client to kubernetes. " + err.Error())
		return false
	}
	service := &corev1.ServiceNameMapping{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-app",
			Namespace: "default",
			Annotations: map[string]string{
				"dubbo.apache.org/application": "dubbo-admin",
				"dubbo.apache.org/protocol":    "dubbo",
			},
		},
		Spec: corev1.ServiceNameMappingSpec{
			InterfaceName: "dubbo",
			ApplicationNames: []string{
				"dubbo-admin",
			},
		},
	}
	if err != nil {
		fmt.Printf("Failed to convert service to unstructured: %v\n", err)
		os.Exit(1)
	}
	logger.Sugar().Infof("Service created: %s\n", service.Name)
	if err != nil {
		logger.Sugar().Errorf("Failed to convert service to unstructured: %v\n", err)
		os.Exit(1)
	}
	c.kubeClient = clientSet
	c.informerClient = informerClient

	return true

}

func (c *client) InitContainer() {
	logger.Sugar().Info("Init controller...")
	informerFactory := informers.NewSharedInformerFactory(c.informerClient, time.Second*10)
	controller := NewController(informerFactory.Dubbo().V1alpha1().ServiceNameMappings())
	stopCh := make(chan struct{})
	informerFactory.Start(stopCh)
	err := controller.PreRun(1, stopCh)
	if err != nil {
		return
	}
}
