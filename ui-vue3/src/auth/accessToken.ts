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

type TokenIssuer = () => Promise<string>

export class AccessTokenManager {
  private enabled = false
  private token: string | undefined
  private refreshPromise: Promise<string> | undefined
  private generation = 0

  constructor(private readonly issue: TokenIssuer) {}

  setEnabled(enabled: boolean): void {
    this.enabled = enabled
    if (!enabled) this.clear()
  }

  isEnabled(): boolean {
    return this.enabled
  }

  async getToken(): Promise<string | undefined> {
    if (!this.enabled) return undefined
    if (this.token) {
      return this.token
    }
    if (!this.refreshPromise) {
      const generation = this.generation
      const refresh = this.issue()
        .then((token) => {
          if (!token) throw new Error('Admin returned an empty AI access token')
          if (generation !== this.generation) {
            throw new Error('AI access token issuance was discarded')
          }
          this.token = token
          return token
        })
        .finally(() => {
          if (this.refreshPromise === refresh) this.refreshPromise = undefined
        })
      this.refreshPromise = refresh
    }
    return this.refreshPromise
  }

  async refreshAfterUnauthorized(rejectedToken: string | undefined): Promise<string | undefined> {
    if (!this.enabled) return undefined
    if (rejectedToken !== undefined && this.token === rejectedToken) {
      this.generation++
      this.token = undefined
    }
    return this.getToken()
  }

  clear(): void {
    this.generation++
    this.token = undefined
    this.refreshPromise = undefined
  }

  peek(): string | undefined {
    return this.token
  }

  setTokenForTest(token: string): void {
    this.token = token
  }
}

export const accessTokenManager = new AccessTokenManager(async () => {
  // Load the API lazily: the shared HTTP client imports this manager to clear it on 401.
  const { issueAccessToken } = await import('@/api/service/login')
  const { data } = await issueAccessToken()
  return data?.accessToken ?? ''
})
