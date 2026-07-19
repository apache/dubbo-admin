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

import { describe, expect, it } from 'vitest'
import { createAIContextSnapshot, getSerializedSize } from '../snapshot'
import type { AIContextBase } from '../types'

const base: AIContextBase = {
  global: { locale: 'cn' },
  page: {
    path: '/home',
    fullPath: '/home?keyword=user',
    query: { keyword: 'user' }
  },
  scope: { mesh: 'nacos2.5' }
}

describe('createAIContextSnapshot', () => {
  it('creates a deterministic versioned snapshot', () => {
    const snapshot = createAIContextSnapshot(
      base,
      [
        {
          id: 'home',
          priority: 10,
          contribution: {
            evidence: {
              id: 'cluster-overview',
              source: 'cluster-info-api',
              data: { applications: 6 }
            }
          }
        }
      ],
      { now: () => new Date('2026-07-19T12:00:00.000Z') }
    )

    expect(snapshot.version).toBe(1)
    expect(snapshot.capturedAt).toBe('2026-07-19T12:00:00.000Z')
    expect(snapshot.evidence?.[0]).toMatchObject({
      id: 'cluster-overview',
      capturedAt: '2026-07-19T12:00:00.000Z',
      priority: 10
    })
  })

  it('removes lower-priority evidence to satisfy the byte budget', () => {
    const snapshot = createAIContextSnapshot(
      base,
      [
        {
          id: 'important',
          priority: 100,
          contribution: {
            evidence: {
              id: 'important',
              source: 'detail-api',
              data: { value: 'a'.repeat(300) }
            }
          }
        },
        {
          id: 'optional',
          priority: 1,
          contribution: {
            evidence: {
              id: 'optional',
              source: 'list-api',
              data: { value: 'b'.repeat(300) }
            }
          }
        }
      ],
      {
        maxBytes: 700,
        now: () => new Date('2026-07-19T12:00:00.000Z')
      }
    )

    expect(snapshot.evidence?.map((section) => section.id)).toEqual(['important'])
    expect(snapshot.truncation).toEqual({
      truncated: true,
      omittedSections: ['optional']
    })
    expect(getSerializedSize(snapshot)).toBeLessThanOrEqual(700)
  })
})
