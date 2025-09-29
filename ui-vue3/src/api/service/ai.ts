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

import axios from 'axios'

const BASE_URL = '/api/v1'

// 定义接口类型
export interface Session {
  session_id: string
  created_at: string
  updated_at: string
  message_count: number
  status: string
}

export interface ChatMessage {
  id: string
  content: string
  role: 'user' | 'assistant'
  timestamp: number
  type?: 'normal' | 'error' | 'partial_error'
}

export interface ChatResponse {
  data: {
    session_id: string
    messages: ChatMessage[]
  }
}

// AI 服务接口
export const aiService = {
  // 创建新会话
  async createSession(): Promise<string> {
    const response = await axios.post(`${BASE_URL}/ai/sessions`)
    return response.data.data.session_id
  },

  // 获取会话列表
  async getSessions(): Promise<Session[]> {
    const response = await axios.get(`${BASE_URL}/ai/sessions`)
    return response.data.data.sessions || []
  },

  // 获取特定会话信息
  async getSessionInfo(sessionId: string): Promise<ChatResponse> {
    const response = await axios.get(`${BASE_URL}/ai/sessions/${sessionId}`)
    return response.data
  },

  // 删除会话
  async deleteSession(sessionId: string): Promise<void> {
    await axios.delete(`${BASE_URL}/ai/sessions/${sessionId}`)
  },

  // 发送聊天消息（流式响应）
  async sendChatMessage(message: string, sessionId?: string): Promise<ReadableStream> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json'
    }

    if (sessionId) {
      headers['X-Session-ID'] = sessionId
    }

    const response = await fetch(`${BASE_URL}/ai/chat/stream`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        message,
        sessionID: sessionId
      }),
      mode: 'cors', // 允许跨域
      credentials: 'include' // 允许携带 cookie
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }

    return response.body!
  }
}

export default aiService
