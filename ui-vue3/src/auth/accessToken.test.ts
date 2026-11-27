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

describe('cookie renewal', () => {
  it('shares pending renewal and retries after a failed issuance', async () => {
    const issue = vi.fn().mockRejectedValueOnce(new Error('timeout')).mockResolvedValue(undefined)
    const manager = new AccessTokenManager(issue)
    const pending = manager.refresh()
    expect(manager.refresh()).toBe(pending)
    await expect(pending).rejects.toThrow('timeout')
    await manager.refresh()
    expect(issue).toHaveBeenCalledTimes(2)
  })
})
