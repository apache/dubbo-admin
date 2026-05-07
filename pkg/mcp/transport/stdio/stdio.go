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

package stdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/apache/dubbo-admin/pkg/mcp/core"
)

// Transport stdio 传输层
type Transport struct {
	server *core.Server
	reader io.Reader
	writer io.Writer
	mu     sync.Mutex
	closed bool
}

// NewTransport 创建 stdio 传输层（使用 stdin/stdout）
func NewTransport(server *core.Server) *Transport {
	return &Transport{
		server: server,
		reader: os.Stdin,
		writer: os.Stdout,
	}
}

// NewTransportWithIO 创建使用指定 reader/writer 的传输层（用于测试）
func NewTransportWithIO(server *core.Server, reader io.Reader, writer io.Writer) *Transport {
	return &Transport{
		server: server,
		reader: reader,
		writer: writer,
	}
}

// Serve 启动 stdio 服务，阻塞运行直到发生错误或上下文取消
func (t *Transport) Serve(ctx context.Context) error {
	// 从 reader 读取请求，写入 writer 响应
	reader := bufio.NewReader(t.reader)
	writer := bufio.NewWriter(t.writer)

	// 确保输出缓冲区刷新
	defer writer.Flush()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 读取一行 JSON 请求
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // 正常结束
			}
			return fmt.Errorf("read stdin: %w", err)
		}

		if len(line) == 0 || line == "\n" {
			continue
		}

		// 解析 JSON-RPC 请求
		var req core.JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			// 发送错误响应
			t.sendError(writer, nil, core.ErrCodeParseError, "Parse error")
			if err := writer.Flush(); err != nil {
				return fmt.Errorf("flush error: %w", err)
			}
			continue
		}

		// 处理请求并获取响应
		resp := t.server.HandleRequest(&req)

		// 发送响应
		respData, err := json.Marshal(resp)
		if err != nil {
			t.sendError(writer, req.ID, core.ErrCodeInternalError, "Failed to marshal response")
		} else {
			writer.Write(respData)
			writer.WriteByte('\n')
		}

		if err := writer.Flush(); err != nil {
			return fmt.Errorf("flush stdout: %w", err)
		}
	}
}

// sendError 发送错误响应
func (t *Transport) sendError(writer *bufio.Writer, id interface{}, code int, message string) {
	resp := core.JSONRPCResponse{
		JSONRPC: core.JSONRPCVersion,
		ID:      id,
		Error: &core.JSONRPCError{
			Code:    code,
			Message: message,
		},
	}
	data, _ := json.Marshal(resp)
	writer.Write(data)
	writer.WriteByte('\n')
}

// Close 关闭传输层
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return nil
	}
	t.closed = true
	return nil
}

// ServeOnce 处理单个请求（用于测试）
func (t *Transport) ServeOnce(input string) (string, error) {
	var req core.JSONRPCRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return "", fmt.Errorf("parse request: %w", err)
	}

	resp := t.server.HandleRequest(&req)
	respData, err := json.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}

	return string(respData) + "\n", nil
}
