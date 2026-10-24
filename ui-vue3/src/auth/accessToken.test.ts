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
import { AccessTokenManager } from './accessToken'

function tokenWithExpiry(exp: number): string {
  const payload = btoa(JSON.stringify({ exp }))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=+$/, '')
  return `header.${payload}.signature`
}

describe('AccessTokenManager', () => {
  it('does not issue tokens while disabled', async () => {
    const issue = vi.fn(async () => tokenWithExpiry(Date.now() / 1000 + 3600))
    const manager = new AccessTokenManager(issue)
    await expect(manager.getToken()).resolves.toBeUndefined()
    expect(issue).not.toHaveBeenCalled()
  })

  it('shares refresh and renews tokens expiring within 60 seconds', async () => {
    let resolveIssue: (token: string) => void = () => undefined
    const issue = vi.fn(
      () =>
        new Promise<string>((resolve) => {
          resolveIssue = resolve
        })
    )
    const manager = new AccessTokenManager(issue)
    manager.setEnabled(true)
    const first = manager.getToken()
    const second = manager.getToken()
    expect(issue).toHaveBeenCalledTimes(1)
    const token = tokenWithExpiry(Math.floor(Date.now() / 1000) + 120)
    resolveIssue(token)
    await expect(Promise.all([first, second])).resolves.toEqual([token, token])

    manager.setTokenForTest(tokenWithExpiry(Math.floor(Date.now() / 1000) + 30))
    const refresh = manager.getToken()
    expect(issue).toHaveBeenCalledTimes(2)
    resolveIssue(token)
    await refresh
  })

  it('keeps the token only in runtime memory', () => {
    const storageSpy = vi.spyOn(Storage.prototype, 'setItem')
    const manager = new AccessTokenManager(async () => tokenWithExpiry(1))
    manager.setEnabled(true)
    manager.setTokenForTest(tokenWithExpiry(9999999999))
    manager.clear()
    expect(manager.peek()).toBeUndefined()
    expect(storageSpy).not.toHaveBeenCalled()
  })

  it('discards a token issued after the manager was cleared', async () => {
    let resolveIssue: (token: string) => void = () => undefined
    const manager = new AccessTokenManager(
      () =>
        new Promise<string>((resolve) => {
          resolveIssue = resolve
        })
    )
    manager.setEnabled(true)
    const pending = manager.getToken()
    manager.clear()
    resolveIssue(tokenWithExpiry(Math.floor(Date.now() / 1000) + 120))

    await expect(pending).rejects.toThrow('discarded')
    expect(manager.peek()).toBeUndefined()
  })
})
