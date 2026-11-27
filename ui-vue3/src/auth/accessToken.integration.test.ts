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

import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { AxiosError } from 'axios'
import { createPinia, setActivePinia } from 'pinia'
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'
import router from '@/router'
import request from '@/base/http/request'
import { aiService } from '@/api/service/ai'
import { getAuthState, removeAuthState, updateAuthState } from '@/utils/AuthUtil'
import { accessTokenManager } from './accessToken'

vi.mock('@/router', async () => {
  const { createMemoryHistory, createRouter } = await import('vue-router')
  return {
    default: createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/:pathMatch(.*)*', component: { render: () => null } }]
    })
  }
})

const server = setupServer()
const defaultAdapter = request.defaults.adapter
const tokenResponse = { code: 'Success', message: 'success', data: true }

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterAll(() => server.close())

beforeEach(async () => {
  setActivePinia(createPinia())
  await router.push('/applications?search=demo')
  updateAuthState(true, 'alice')
  accessTokenManager.setEnabled(true)
  // Node fetch needs absolute URLs; keep both browser transports intercepted by MSW.
  const interceptedFetch = globalThis.fetch
  vi.stubGlobal('fetch', (input: RequestInfo | URL, init?: RequestInit) =>
    interceptedFetch(
      typeof input === 'string' ? new URL(input, window.location.origin) : input,
      init
    )
  )
})

afterEach(() => {
  accessTokenManager.setEnabled(false)
  removeAuthState()
  request.defaults.adapter = defaultAdapter
  server.resetHandlers()
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('Admin access token requests', () => {
  it('clears expired login state and redirects to login after an AI Bearer challenge', async () => {
    server.use(
      http.get('*/api/v1/ai/sessions', () =>
        HttpResponse.json(
          { error: 'invalid or missing bearer token' },
          { status: 401, headers: { 'WWW-Authenticate': 'Bearer' } }
        )
      ),
      http.post('*/api/v1/auth/token', () =>
        HttpResponse.json({ code: 'Unauthorized', message: 'login required' }, { status: 401 })
      )
    )

    await expect(aiService.getSessions()).rejects.toBeDefined()
    expect(getAuthState().state).not.toBe(true)
    await vi.waitFor(() => expect(router.currentRoute.value.path).toBe('/login'))
  })

  it('retries an AI request with a new token after the cached token is rejected', async () => {
    let issuances = 0
    const authorizations: (string | null)[] = []
    server.use(
      http.post('*/api/v1/auth/token', () => {
        issuances++
        return HttpResponse.json(tokenResponse)
      }),
      http.get('*/api/v1/ai/sessions', ({ request }) => {
        const authorization = request.headers.get('Authorization')
        authorizations.push(authorization)
        if (issuances === 1) {
          return HttpResponse.json({ code: 'Success', message: 'success', data: { sessions: [] } })
        }
        return HttpResponse.json(
          { error: 'invalid or missing bearer token' },
          {
            status: 401,
            headers: { 'WWW-Authenticate': 'Bearer' }
          }
        )
      })
    )

    await expect(aiService.getSessions()).resolves.toEqual([])
    expect(authorizations).toEqual([null, null])
    expect(issuances).toBe(1)
  })

  it('uses the shared timeout and permits a new issuance after a timeout', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    server.use(http.post('*/api/v1/auth/token', () => HttpResponse.json(tokenResponse)))
    let configuredTimeout: number | undefined
    // Simulate a transport timeout while exercising the real request interceptors.
    request.defaults.adapter = async (config) => {
      configuredTimeout = config.timeout
      throw new AxiosError('timeout of 30000ms exceeded', AxiosError.ECONNABORTED, config)
    }

    const result = await Promise.allSettled([accessTokenManager.refresh()])
    expect(result[0].status).toBe('rejected')
    expect(configuredTimeout).toBe(30_000)
    expect(getAuthState().state).toBe(true)
    expect(router.currentRoute.value.path).toBe('/applications')

    request.defaults.adapter = defaultAdapter
    await expect(accessTokenManager.refresh()).resolves.toBeUndefined()
  })
})
