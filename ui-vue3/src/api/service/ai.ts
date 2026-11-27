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

import axios, { type InternalAxiosRequestConfig } from 'axios'
import { AccessTokenManager, accessTokenManager } from '@/auth/accessToken'
import type { AIContextSnapshot } from '@/ai-context'
import type { ChatResponse, ChatService, Session } from '@/components/ai-chat/types'
export type { ChatMessage, ChatResponse, Session } from '@/components/ai-chat/types'

const BASE_URL = '/api/v1'

type RetryConfig = InternalAxiosRequestConfig & { _aiAuthRetried?: boolean }

const aiClient = axios.create({ baseURL: `${BASE_URL}/ai`, headers: { 'X-AI-Cookie': '1' } })

aiClient.interceptors.response.use(undefined, async (error) => {
  const config = error.config as RetryConfig | undefined
  const challenge = error.response?.headers?.['www-authenticate']
  if (
    accessTokenManager.isEnabled() &&
    error.response?.status === 401 &&
    typeof challenge === 'string' &&
    challenge.toLowerCase().includes('bearer') &&
    config &&
    !config._aiAuthRetried
  ) {
    config._aiAuthRetried = true
    await accessTokenManager.refresh()
    return aiClient.request(config)
  }
  return Promise.reject(error)
})

export async function authenticatedFetch(
  input: RequestInfo | URL,
  init: RequestInit = {},
  manager: AccessTokenManager = accessTokenManager,
  fetcher: typeof fetch = fetch
): Promise<Response> {
  const headers = new Headers(init.headers)
  headers.set('X-AI-Cookie', '1')
  const options: RequestInit = { ...init, headers, credentials: 'same-origin', mode: 'same-origin' }
  const response = await fetcher(input, options)
  if (
    manager.isEnabled() &&
    response.status === 401 &&
    response.headers.get('WWW-Authenticate')?.toLowerCase().includes('bearer')
  ) {
    await manager.refresh()
    return fetcher(input, options)
  }
  return response
}
// AI 服务接口
export const aiService: ChatService<AIContextSnapshot> = {
  // 创建新会话
  async createSession(): Promise<string> {
    const response = await aiClient.post('/sessions')
    return response.data.data.session_id
  },

  // 获取会话列表
  async getSessions(): Promise<Session[]> {
    const response = await aiClient.get('/sessions')
    return response.data.data.sessions || []
  },

  // 获取特定会话信息
  async getSessionInfo(sessionId: string): Promise<ChatResponse> {
    const response = await aiClient.get(`/sessions/${sessionId}`)
    return response.data
  },

  // 删除会话
  async deleteSession(sessionId: string): Promise<void> {
    await aiClient.delete(`/sessions/${sessionId}`)
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

    const response = await authenticatedFetch(`${BASE_URL}/ai/chat/stream`, {
      method: 'POST',
      headers,
      body: JSON.stringify({
        message,
        sessionID: sessionId,
        context
      })
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
