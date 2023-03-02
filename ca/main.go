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

package main

import (
	"github.com/apache/dubbo-admin/ca/pkg/config"
	"github.com/apache/dubbo-admin/ca/pkg/logger"
	"github.com/apache/dubbo-admin/ca/pkg/security"
	"github.com/ianschenck/envflag"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	logger.Init()
	namespace := envflag.String("namespace", "dubbo-system", "options")
	serviceName := envflag.String("serviceName", "dubbo-ca", "options")
	plainServerPort := envflag.Int("plainServerPort", 30060, "options")
	secureServerPort := envflag.Int("secureServerPort", 30062, "options")
	debugPort := envflag.Int("debugPort", 30070, "options")
	webhookPort := envflag.Int("webhookPort", 30080, "options")
	webhookAllowOnErr := envflag.Bool("webhookAllowOnErr", false, "options")
	inPodEnv := envflag.Bool("inPodEnv", false, "options")
	isKubernetesConnected := envflag.Bool("isKubernetesConnected", false, "options")
	enableOIDCCheck := envflag.Bool("enableOIDCCheck", false, "options")
	envflag.Parse()

	// TODO read options from env
	options := &config.Options{
		Namespace:   *namespace,
		ServiceName: *serviceName,

		PlainServerPort:  *plainServerPort,
		SecureServerPort: *secureServerPort,
		DebugPort:        *debugPort,

		WebhookPort:       int32(*webhookPort),
		WebhookAllowOnErr: *webhookAllowOnErr,

		CaValidity:   30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity: 1 * 60 * 60 * 1000,       // 1 hour

		InPodEnv:              *inPodEnv,
		IsKubernetesConnected: *isKubernetesConnected,
		EnableOIDCCheck:       *enableOIDCCheck,
	}

	s := security.NewServer(options)

	s.Init()
	s.Start()

	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.StopChan, syscall.SIGINT, syscall.SIGTERM)
	signal.Notify(s.CertStorage.GetStopChan(), syscall.SIGINT, syscall.SIGTERM)

	<-c
}
