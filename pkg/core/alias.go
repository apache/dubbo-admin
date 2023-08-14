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

package core

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/google/uuid"
)

var SetupSignalHandler = func() (context.Context, context.Context) {
	gracefulCtx, gracefulCancel := context.WithCancel(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 3)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		s := <-c
		logger.Sugar().Info("Received signal, stopping instance gracefully", "signal", s.String())
		gracefulCancel()
		s = <-c
		logger.Sugar().Info("Received second signal, stopping instance", "signal", s.String())
		cancel()
		s = <-c
		logger.Sugar().Info("Received third signal, force exit", "signal", s.String())
		os.Exit(1)
	}()
	return gracefulCtx, ctx
}

func NewUUID() string {
	return uuid.NewString()
}
