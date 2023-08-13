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
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/apache/dubbo-admin/pkg/authority"
	"github.com/apache/dubbo-admin/pkg/config"
	dubbo_cp "github.com/apache/dubbo-admin/pkg/config/app/dubbo-cp"
	"github.com/apache/dubbo-admin/pkg/core/bootstrap"
	"github.com/apache/dubbo-admin/pkg/core/cert"
	"github.com/apache/dubbo-admin/pkg/core/kubeclient"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	cp_server "github.com/apache/dubbo-admin/pkg/cp-server"
	"github.com/apache/dubbo-admin/pkg/dds"
	"github.com/apache/dubbo-admin/pkg/snp"
)

const gracefullyShutdownDuration = 3 * time.Second

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

func main() {
	cfg := dubbo_cp.DefaultConfig()
	err := config.Load("", &cfg)
	if err != nil {
		logger.Sugar().Error(err, "could not load the configuration")
		return
	}
	gracefulCtx, ctx := SetupSignalHandler()

	rt, err := bootstrap.Bootstrap(gracefulCtx, &cfg)
	if err != nil {
		logger.Sugar().Error(err, "unable to set up Control Plane runtime")
		return
	}
	cfgForDisplay, err := config.ConfigForDisplay(&cfg)
	if err != nil {
		logger.Sugar().Error(err, "unable to prepare config for display")
		return
	}
	cfgBytes, err := config.ToJson(cfgForDisplay)
	if err != nil {
		logger.Sugar().Error(err, "unable to convert config to json")
		return
	}
	logger.Sugar().Info(fmt.Sprintf("Current config %s", cfgBytes))

	//if err := admin.Setup(rt); err != nil {
	//	logger.Sugar().Error(err, "unable to set up Metrics")
	//}

	if err := cert.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up certProvider")
	}

	if err := authority.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up authority")
	}

	if err := dds.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up dds")
	}

	if err := snp.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up snp")
	}

	if err := cp_server.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up grpc server")
	}

	// This must be last, otherwise we will not know which informers to register
	if err := kubeclient.Setup(rt); err != nil {
		logger.Sugar().Error(err, "unable to set up kube client")
	}

	logger.Sugar().Info("starting Control Plane")
	if err := rt.Start(gracefulCtx.Done()); err != nil {
		logger.Sugar().Error(err, "problem running Control Plane")
		return
	}

	logger.Sugar().Info("Stop signal received. Waiting 3 seconds for components to stop gracefully...")
	select {
	case <-ctx.Done():
	case <-time.After(gracefullyShutdownDuration):
	}
	logger.Sugar().Info("Stopping Control Plane")
	return
}
