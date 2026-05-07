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

package http

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
)

// Transport HTTP传输层
type Transport struct {
	server     *core.Server
	httpServer *http.Server
	handler    *Handler
	mu         sync.RWMutex
	started    bool
}

// Config HTTP传输配置
type Config struct {
	Host           string
	Port           int
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ShutdownTimeout time.Duration
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Host:            "0.0.0.0",
		Port:            8080,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,
	}
}

// NewTransport 创建HTTP传输层
func NewTransport(server *core.Server) *Transport {
	return NewTransportWithConfig(server, DefaultConfig())
}

// NewTransportWithConfig 使用指定配置创建HTTP传输层
func NewTransportWithConfig(server *core.Server, cfg *Config) *Transport {
	handler := NewHandler(server)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Transport{
		server:     server,
		httpServer: httpServer,
		handler:    handler,
	}
}

// Start 启动HTTP服务器（阻塞运行）
func (t *Transport) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return fmt.Errorf("transport already started")
	}
	t.started = true
	t.mu.Unlock()

	// 启动服务器
	errCh := make(chan error, 1)
	go func() {
		if err := t.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// 等待上下文取消或错误
	select {
	case <-ctx.Done():
		return t.Shutdown()
	case err := <-errCh:
		return err
	}
}

// StartAsync 异步启动HTTP服务器
func (t *Transport) StartAsync(ctx context.Context) error {
	t.mu.Lock()
	if t.started {
		t.mu.Unlock()
		return fmt.Errorf("transport already started")
	}
	t.started = true
	t.mu.Unlock()

	go func() {
		if err := t.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// 记录错误但不退出
			fmt.Printf("HTTP server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown 关闭HTTP服务器
func (t *Transport) Shutdown() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.started {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := t.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	t.started = false
	return nil
}

// Close 关闭传输层
func (t *Transport) Close() error {
	return t.Shutdown()
}

// Addr 返回监听地址
func (t *Transport) Addr() string {
	return t.httpServer.Addr
}

// GetServer 获取HTTP服务器（用于自定义路由）
func (t *Transport) GetServer() *http.Server {
	return t.httpServer
}

// GetHandler 获取HTTP处理器（用于自定义路由）
func (t *Transport) GetHandler() *Handler {
	return t.handler
}
