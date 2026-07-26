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

import { afterEach, describe, expect, it, vi } from 'vitest'
import { aiService } from '../ai'
import type { AIContextSnapshot } from '@/ai-context'

const context: AIContextSnapshot = {
  version: 1,
  capturedAt: '2026-07-19T13:00:00.000Z',
  global: { locale: 'cn' },
  page: { path: '/home' },
  scope: { mesh: 'nacos2.5' }
}

describe('aiService.sendChatMessage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('sends the selected page context with the existing chat payload', async () => {
    const stream = new ReadableStream()
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, body: stream })
    vi.stubGlobal('fetch', fetchMock)

    await aiService.sendChatMessage('hello', 'session-1', context)

    const request = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(String(request.body))).toEqual({
      message: 'hello',
      sessionID: 'session-1',
      context
    })
  })

  it('omits context when page context is disabled', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, body: new ReadableStream() })
    vi.stubGlobal('fetch', fetchMock)

    await aiService.sendChatMessage('hello', 'session-1')

    const request = fetchMock.mock.calls[0][1] as RequestInit
    expect(JSON.parse(String(request.body))).toEqual({
      message: 'hello',
      sessionID: 'session-1'
    })
  })

  it('rejects successful responses without a stream body', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, body: null }))

    await expect(aiService.sendChatMessage('hello', 'session-1')).rejects.toThrow(
      'AI service returned an empty response body'
    )
  })
})
