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

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getTraceStartTime } from './trace'

const { request } = vi.hoisted(() => ({ request: vi.fn() }))
vi.mock('@/base/http/request', () => ({ default: request }))

describe('getTraceStartTime', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('classifies a transport failure with no response as a network error', async () => {
    request.mockRejectedValueOnce(undefined)

    await expect(getTraceStartTime('766b9604d84bd8e6b17b2292dfd9bdd0')).rejects.toMatchObject({
      code: 'NetWorkError'
    })
  })

  it('preserves the backend error code', async () => {
    request.mockRejectedValueOnce({ code: 'NotFoundError' })

    await expect(getTraceStartTime('766b9604d84bd8e6b17b2292dfd9bdd0')).rejects.toMatchObject({
      code: 'NotFoundError'
    })
  })
})
