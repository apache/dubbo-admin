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
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
)

// SSETransport Server-Sent Events传输层
type SSETransport struct {
	server   *core.Server
	clients  map[*SSEClient]bool
	mu       sync.RWMutex
	broadcast chan []byte
}

// SSEClient SSE客户端连接
type SSEClient struct {
	id   string
	ch   chan []byte
	ctx  context.Context
	done context.CancelFunc
}

// NewSSEClient 创建SSE客户端
func NewSSEClient(id string) *SSEClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &SSEClient{
		id:   id,
		ch:   make(chan []byte, 256),
		ctx:  ctx,
		done: cancel,
	}
}

// NewSSETransport 创建SSE传输层
func NewSSETransport(server *core.Server) *SSETransport {
	return &SSETransport{
		server:   server,
		clients:  make(map[*SSEClient]bool),
		broadcast: make(chan []byte, 256),
	}
}

// HandleSSE 处理SSE连接
func (t *SSETransport) HandleSSE(w http.ResponseWriter, r *http.Request) {
	// 设置SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 创建客户端
	client := NewSSEClient(r.RemoteAddr)

	t.mu.Lock()
	t.clients[client] = true
	t.mu.Unlock()

	// 发送连接成功消息
	t.sendToClient(client, t.sseEvent("connected", "SSE connection established"))

	// 等待断开连接
	<-client.ctx.Done()

	t.mu.Lock()
	delete(t.clients, client)
	close(client.ch)
	t.mu.Unlock()
}

// HandleRPCWithSSE 处理带有SSE的RPC请求（支持异步响应流）
func (t *SSETransport) HandleRPCWithSSE(w http.ResponseWriter, r *http.Request) {
	// 设置SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	// 读取请求
	decoder := json.NewDecoder(r.Body)
	var req core.JSONRPCRequest
	if err := decoder.Decode(&req); err != nil {
		t.sendSSE(w, "error", err.Error())
		return
	}

	// 发送处理中状态
	t.sendSSE(w, "processing", "Processing request...")

	// 处理请求
	resp := t.server.HandleRequest(&req)

	// 发送响应
	respData, _ := json.Marshal(resp)
	t.sendSSE(w, "result", string(respData))

	// 发送完成事件
	t.sendSSE(w, "done", "")
	flusher.Flush()
}

// BroadcastEvent 广播事件到所有客户端
func (t *SSETransport) BroadcastEvent(eventType string, data interface{}) {
	event := t.sseEvent(eventType, data)
	select {
	case t.broadcast <- event:
	default:
		// broadcast channel满，丢弃事件
	}
}

// StartBroadcasting 启动广播协程
func (t *SSETransport) StartBroadcasting(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-t.broadcast:
				t.mu.RLock()
				for client := range t.clients {
					select {
					case client.ch <- msg:
					default:
						// client channel满，关闭客户端
						client.done()
					}
				}
				t.mu.RUnlock()
			}
		}
	}()
}

// sendToClient 发送消息到指定客户端
func (t *SSETransport) sendToClient(client *SSEClient, msg []byte) {
	select {
	case client.ch <- msg:
	default:
		client.done()
	}
}

// sseEvent 创建SSE事件格式
func (t *SSETransport) sseEvent(eventType string, data interface{}) []byte {
	var dataStr string
	switch v := data.(type) {
	case string:
		dataStr = v
	case []byte:
		dataStr = string(v)
	default:
		bytes, _ := json.Marshal(data)
		dataStr = string(bytes)
	}
	return []byte(fmt.Sprintf("event: %s\ndata: %s\n\n", eventType, dataStr))
}

// sendSSE 发送SSE事件到ResponseWriter
func (t *SSETransport) sendSSE(w http.ResponseWriter, eventType, data string) {
	event := t.sseEvent(eventType, data)
	w.Write(event)
}
