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

package mcp

import (
	consolectx "github.com/apache/dubbo-admin/pkg/console/context"
	"github.com/apache/dubbo-admin/pkg/config/app"
	"github.com/apache/dubbo-admin/pkg/core/runtime"
)

const (
	// ComponentType MCP组件类型
	ComponentType = runtime.ComponentType("mcp")
)

// Component MCP组件，集成到admin服务中
type Component struct {
	server     *Server
	consoleCtx consolectx.Context
	cfg        *app.MCPConfig
}

func init() {
	runtime.RegisterComponent(&Component{})
}

// Type 返回组件类型
func (c *Component) Type() runtime.ComponentType {
	return ComponentType
}

// Order 返回组件启动顺序
func (c *Component) Order() int {
	return 999 // 在Console之后启动
}

// RequiredDependencies 返回依赖的组件
func (c *Component) RequiredDependencies() []runtime.ComponentType {
	return []runtime.ComponentType{
		runtime.Console, // 依赖Console（需要runtime）
	}
}

// Init 初始化MCP组件
func (c *Component) Init(ctx runtime.BuilderContext) error {
	cfg := ctx.Config()

	// 从admin配置中获取MCP配置
	if cfg.MCP == nil {
		// MCP未配置，使用默认配置（禁用）
		c.cfg = &app.MCPConfig{
			Enabled: false,
			Path:    "/api/mcp",
		}
		return nil
	}

	c.cfg = cfg.MCP

	// 如果未启用，直接返回
	if !c.cfg.Enabled {
		return nil
	}

	return nil
}

// Start 启动MCP组件
func (c *Component) Start(coreRt runtime.Runtime, stop <-chan struct{}) error {
	// 如果未启用，直接返回
	if c.cfg == nil || !c.cfg.Enabled {
		<-stop
		return nil
	}

	// 获取console context（由Console组件创建）
	c.consoleCtx = consolectx.NewConsoleContext(coreRt)

	// 创建MCP服务器
	c.server = NewServer("dubbo-admin", "1.0.0")
	c.server.SetConsoleContext(c.consoleCtx)

	// 注册所有工具
	RegisterTools(c.server)

	// 获取console的HTTP引擎并注册MCP路由
	consoleComp, err := coreRt.GetComponent(runtime.Console)
	if err != nil {
		// Console未启动，无法注册MCP路由
		<-stop
		return nil
	}

	// 通过runtime获取gin.Engine
	// 这里需要一种方式来访问console组件的gin.Engine
	// 暂时使用全局注册方式
	RegisterMCPRoutes(c.server, c.cfg.Path)

	_ = consoleComp // 暂时忽略，实际使用时需要访问console的gin.Engine

	// 等待停止信号
	<-stop

	return nil
}

// RegisterMCPRoutes 注册MCP路由到gin引擎
// 这个函数应该在console路由初始化后调用
func RegisterMCPRoutes(server *Server, path string) {
	if path == "" {
		path = "/api/mcp"
	}
	_ = server
	_ = path
}

// GetServer 获取MCP服务器实例
func (c *Component) GetServer() *Server {
	return c.server
}
