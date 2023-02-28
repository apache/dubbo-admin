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
	"os"
	"os/signal"
	"syscall"
)

// TODO read namespace from env
const namespace = "dubbo-system"
const serviceName = "dubbo-ca"

func main() {
	logger.Init()
	// TODO read options from env
	options := &config.Options{
		Namespace:   namespace,
		ServiceName: serviceName,

		PlainServerPort:  30060,
		SecureServerPort: 30062,
		DebugPort:        30070,

		WebhookPort:       30080,
		WebhookAllowOnErr: false,

		CaValidity:   30 * 24 * 60 * 60 * 1000, // 30 day
		CertValidity: 1 * 60 * 60 * 1000,       // 1 hour

		InPodEnv:              false,
		IsKubernetesConnected: false,
		EnableOIDCCheck:       false,
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
