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

import { describe, expect, it, vi } from 'vitest'
import { AccessTokenManager } from '@/auth/accessToken'
import { authenticatedFetch } from './ai'

describe('authenticatedFetch', () => {
  it('keeps anonymous compatibility while access tokens are disabled', async () => {
    const manager = new AccessTokenManager(async () => undefined)
    const fetcher = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      void input
      void init
      return new Response('', { status: 200 })
    })
    await authenticatedFetch('/api/v1/ai/test', {}, manager, fetcher)
    expect(new Headers(fetcher.mock.calls[0][1]?.headers).get('Authorization')).toBeNull()
  })

  it('retries one challenged 401 with same-origin cookies', async () => {
    const issue = vi.fn().mockResolvedValue(undefined)
    const manager = new AccessTokenManager(issue)
    manager.setEnabled(true)
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce(
        new Response('', { status: 401, headers: { 'WWW-Authenticate': 'Bearer' } })
      )
      .mockResolvedValueOnce(new Response('', { status: 200 }))
    const response = await authenticatedFetch('/api/v1/ai/test', {}, manager, fetcher)
    expect(response.status).toBe(200)
    expect(issue).toHaveBeenCalledTimes(1)
    expect(fetcher).toHaveBeenCalledTimes(2)
    expect(fetcher.mock.calls[1][1].credentials).toBe('same-origin')
    expect(new Headers(fetcher.mock.calls[1][1].headers).has('Authorization')).toBe(false)
  })

  it('does not refresh a 403', async () => {
    const issue = vi.fn().mockResolvedValue(undefined)
    const manager = new AccessTokenManager(issue)
    manager.setEnabled(true)
    const fetcher = vi.fn(async () => new Response('', { status: 403 }))
    await authenticatedFetch('/api/v1/ai/test', {}, manager, fetcher)
    expect(issue).toHaveBeenCalledTimes(0)
    expect(fetcher).toHaveBeenCalledTimes(1)
  })

  it('shares one refresh across concurrent Bearer challenges', async () => {
    const issue = vi.fn().mockResolvedValue(undefined)
    const manager = new AccessTokenManager(issue)
    manager.setEnabled(true)
    const fetcher = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      void init
      return issue.mock.calls.length > 0
        ? new Response('', { status: 200 })
        : new Response('', { status: 401, headers: { 'WWW-Authenticate': 'Bearer' } })
    })

    const responses = await Promise.all([
      authenticatedFetch('/api/v1/ai/test', {}, manager, fetcher),
      authenticatedFetch('/api/v1/ai/test', {}, manager, fetcher)
    ])

    expect(responses.map((response) => response.status)).toEqual([200, 200])
    expect(issue).toHaveBeenCalledTimes(1)
  })
})
