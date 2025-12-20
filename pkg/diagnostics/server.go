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

package diagnostics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/http/pprof"
	"time"

	diagnosticsconfig "github.com/apache/dubbo-admin/pkg/config/diagnostics"
	"github.com/apache/dubbo-admin/pkg/core/logger"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	runtime.RegisterComponent(&diagnosticsServer{})
}

const DiagnosticsServer = "diagnostics server"

type diagnosticsServer struct {
	config *diagnosticsconfig.Config
}

var (
	_ runtime.Component = &diagnosticsServer{}
)

func (s *diagnosticsServer) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{} // Diagnostics server has no dependencies
}

func (s *diagnosticsServer) Type() runtime.ComponentType {
	return DiagnosticsServer
}

func (s *diagnosticsServer) Order() int {
	return math.MaxInt
}

func (s *diagnosticsServer) Init(ctx runtime.BuilderContext) error {
	s.config = ctx.Config().Diagnostics
	return nil
}

func (s *diagnosticsServer) Start(_ runtime.Runtime, stop <-chan struct{}) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ready", func(resp http.ResponseWriter, _ *http.Request) {
		resp.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/healthy", func(resp http.ResponseWriter, _ *http.Request) {
		resp.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.config.ServerPort),
		Handler:           mux,
		ReadHeaderTimeout: time.Second,
	}

	logger.Infof("starting diagnostic server, endpoint is 0.0.0.0: %d", s.config.ServerPort)
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		var err error
		err = httpServer.ListenAndServe()
		if err != nil {
			switch {
			case errors.Is(err, http.ErrServerClosed):
				logger.Info("shutting down diagnostics server")
			default:
				logger.Error("could not start diagnostics http server, unknown err: %s", err)
				errChan <- err
			}
			return
		}
		logger.Info("terminated normally")
	}()

	select {
	case <-stop:
		logger.Info("received stop signal, stopping diagnostics server ...")
		return httpServer.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}
