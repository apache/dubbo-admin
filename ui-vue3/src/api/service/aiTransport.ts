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

export class AIAPIError extends Error {
  constructor(
    public code: string,
    public status: number
  ) {
    super(code)
  }
}

type RetryConfig = InternalAxiosRequestConfig & { _aiAuthRetried?: boolean }

// Every AI endpoint verifies the Admin-issued Bearer token; cookies alone are
// rejected by the production RS256/JWKS verifier. Keep one transport so chat,
// Memory and any future endpoint share token issuance and 401 refresh.
const BASE_URL = '/api/v1'

export const aiClient = axios.create({ baseURL: `${BASE_URL}/ai` })

aiClient.interceptors.request.use(async (config) => {
  const token = await accessTokenManager.getToken()
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

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
    const authorization = config.headers.Authorization
    const rejectedToken =
      typeof authorization === 'string' && authorization.startsWith('Bearer ')
        ? authorization.slice(7)
        : undefined
    const token = await accessTokenManager.refreshAfterUnauthorized(rejectedToken)
    if (token) config.headers.Authorization = `Bearer ${token}`
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
  const execute = async (retried: boolean, suppliedToken?: string): Promise<Response> => {
    const token = suppliedToken ?? (await manager.getToken())
    const headers = new Headers(init.headers)
    if (token) headers.set('Authorization', `Bearer ${token}`)
    const response = await fetcher(input, { ...init, headers })
    if (
      !retried &&
      manager.isEnabled() &&
      response.status === 401 &&
      response.headers.get('WWW-Authenticate')?.toLowerCase().includes('bearer')
    ) {
      const refreshedToken = await manager.refreshAfterUnauthorized(token)
      return execute(true, refreshedToken)
    }
    return response
  }
  return execute(false)
}
