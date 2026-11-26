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

import { AxiosError, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { accessTokenManager } from '@/auth/accessToken'
import { aiClient } from './aiTransport'
import { aiService } from './ai'
import { aiMemoryService } from './aiMemory'

const originalAdapter = aiClient.defaults.adapter

function unexpiredToken(subject: string): string {
  const payload = btoa(JSON.stringify({ sub: subject, exp: Math.floor(Date.now() / 1000) + 3600 }))
  return `header.${payload}.signature`
}

function respond<T>(
  config: InternalAxiosRequestConfig,
  status: number,
  data: T,
  headers: Record<string, string> = {}
): AxiosResponse<T> {
  return { data, status, statusText: '', headers, config }
}

describe('authenticated AI transport', () => {
  afterEach(() => {
    aiClient.defaults.adapter = originalAdapter
    accessTokenManager.setEnabled(false)
    vi.unstubAllGlobals()
  })

  it('attaches the same Bearer token to session and Memory requests', async () => {
    const token = unexpiredToken('user-1')
    accessTokenManager.setEnabled(true)
    accessTokenManager.setTokenForTest(token)
    const seen: string[] = []
    aiClient.defaults.adapter = async (config) => {
      seen.push(String(config.headers.Authorization))
      return respond(config, 200, { data: { session_id: 'session-1' } })
    }

    await aiService.createSession()
    await aiMemoryService.options()

    expect(seen).toEqual([`Bearer ${token}`, `Bearer ${token}`])
  })

  it('refreshes once and retries a challenged Memory request', async () => {
    const stale = unexpiredToken('stale')
    const fresh = unexpiredToken('fresh')
    accessTokenManager.setEnabled(true)
    accessTokenManager.setTokenForTest(stale)
    vi.stubGlobal(
      'fetch',
      vi.fn(
        async () =>
          new Response(JSON.stringify({ data: { accessToken: fresh } }), {
            status: 200,
            headers: { 'Content-Type': 'application/json' }
          })
      )
    )
    const seen: string[] = []
    aiClient.defaults.adapter = async (config) => {
      const authorization = String(config.headers.Authorization)
      seen.push(authorization)
      if (authorization === `Bearer ${stale}`) {
        const response = respond(
          config,
          401,
          { code: 'UNAUTHORIZED' },
          { 'www-authenticate': 'Bearer' }
        )
        // A custom adapter must reject non-2xx responses itself.
        throw new AxiosError('Unauthorized', AxiosError.ERR_BAD_REQUEST, config, null, response)
      }
      return respond(config, 200, { data: { projects: [], task_types: [] } })
    }

    await expect(aiMemoryService.options()).resolves.toEqual({ projects: [], task_types: [] })
    expect(seen).toEqual([`Bearer ${stale}`, `Bearer ${fresh}`])
  })
})
