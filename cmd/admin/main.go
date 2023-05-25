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

package main

import (
	"os"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"github.com/apache/dubbo-admin/pkg/admin/config"
	mock "github.com/apache/dubbo-admin/pkg/admin/providers/mock"
	"github.com/apache/dubbo-admin/pkg/admin/router"
	"github.com/apache/dubbo-admin/pkg/admin/services"
)

// @title           Dubbo-Admin API
// @version         1.0
// @description     This is a dubbo-admin swagger ui server.
// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html
// @host      127.0.0.1:38080
// @BasePath  /
func main() {
	config.LoadConfig()
	go services.StartSubscribe(config.RegistryCenter)
	defer func() {
		services.DestroySubscribe(config.RegistryCenter)
	}()
	os.Setenv(constant.ConfigFileEnvKey, config.MockProviderConf)
	go mock.RunMockServiceServer()
	router := router.InitRouter()
	_ = router.Run(":38080")
}
