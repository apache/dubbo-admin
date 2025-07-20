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
	"fmt"
	"math"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/bakito/go-log-logr-adapter/adapter"

	diagnosticsconfig "github.com/apache/dubbo-admin/pkg/config/diagnostics"
	"github.com/apache/dubbo-admin/pkg/core"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

func init() {
	runtime.RegisterComponent(&diagnosticsServer{})
}

var diagnosticsServerLog = core.Log.WithName("diagnostics")

const DiagnosticsServer = "diagnostics server"

type diagnosticsServer struct {
	config *diagnosticsconfig.Config
}

var (
	_ runtime.Component = &diagnosticsServer{}
)

func (s *diagnosticsServer) Type() runtime.ComponentType {
	return DiagnosticsServer
}

func (s *diagnosticsServer) SubType() runtime.ComponentType {
	return runtime.DefaultComponentSubType
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
		ErrorLog:          adapter.ToStd(diagnosticsServerLog),
	}

	diagnosticsServerLog.Info("starting diagnostic server", "interface", "0.0.0.0", "port", s.config.ServerPort)
	errChan := make(chan error)
	go func() {
		defer close(errChan)
		var err error
		err = httpServer.ListenAndServe()
		if err != nil {
			switch err {
			case http.ErrServerClosed:
				diagnosticsServerLog.Info("shutting down server")
			default:
				diagnosticsServerLog.Error(err, "could not start HTTP Server")
				errChan <- err
			}
			return
		}
		diagnosticsServerLog.Info("terminated normally")
	}()

	select {
	case <-stop:
		diagnosticsServerLog.Info("stopping")
		return httpServer.Shutdown(context.Background())
	case err := <-errChan:
		return err
	}
}
