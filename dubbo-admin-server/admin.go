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
	"admin/pkg/config"
	"admin/pkg/services"
	"admin/router"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	config.LoadConfig()
	go services.StartSubscribe(config.RegistryCenter)
	router := router.InitRouter()
	quit := make(chan os.Signal)
	go func() {
		_ = router.Run(":38080")
		close(quit)
	}()
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")
	services.DestroySubscribe(config.RegistryCenter)
}
