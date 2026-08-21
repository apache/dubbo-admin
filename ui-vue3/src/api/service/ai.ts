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
import type { AIContextSnapshot } from '@/ai-context'
import type { ChatResponse, ChatService, Session } from '@/components/ai-chat/types'

export type { ChatMessage, ChatResponse, Session } from '@/components/ai-chat/types'

const BASE_URL = '/api/v1'

// AI 服务接口
export const aiService: ChatService<AIContextSnapshot> = {
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
  async sendChatMessage(
    message: string,
    sessionId?: string,
    context?: AIContextSnapshot
  ): Promise<ReadableStream<Uint8Array>> {
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
        sessionID: sessionId,
        context
      }),
      mode: 'cors', // 允许跨域
      credentials: 'include' // 允许携带 cookie
    })

    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`)
    }
    if (!response.body) {
      throw new Error('AI service returned an empty response body')
    }

    return response.body
  }
}

export default aiService
