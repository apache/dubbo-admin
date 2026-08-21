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
import { AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID, selectAIContextSnapshot } from '../selection'
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

  it('preserves bounded structured evidence for page summaries', () => {
    const snapshot = createAIContextSnapshot(base, [
      {
        id: 'table',
        priority: 30,
        contribution: {
          evidence: {
            id: 'search-results',
            source: 'search-result-table',
            data: {
              rows: [{ appName: 'shop-user', clusters: ['nacos-a', 'nacos-b'] }]
            }
          }
        }
      }
    ])

    expect(snapshot.evidence?.[0].data.rows).toEqual([
      { appName: 'shop-user', clusters: ['nacos-a', 'nacos-b'] }
    ])
  })

  it('preserves nested traffic rule matches within the global depth limit', () => {
    const snapshot = createAIContextSnapshot(base, [
      {
        id: 'rule',
        priority: 80,
        contribution: {
          evidence: {
            id: 'rule-content',
            source: 'traffic-rule-page',
            data: {
              content: {
                tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] }]
              }
            }
          }
        }
      }
    ])

    expect(snapshot.evidence?.[0].data).toEqual({
      content: {
        tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] }]
      }
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

  it('selects optional evidence without mutating the collected snapshot', () => {
    const snapshot = createAIContextSnapshot(base, [
      {
        id: 'home',
        priority: 10,
        contribution: {
          state: {
            filters: { keyword: 'shop' },
            unsavedChanges: { kind: 'condition-rule', entryCount: 1 }
          },
          evidence: [
            { id: 'overview', source: 'overview-api', data: { applications: 3 } },
            { id: 'filters', source: 'page-state', data: { keyword: 'shop' } }
          ]
        }
      }
    ])

    const selected = selectAIContextSnapshot(snapshot, {
      excludedSectionIds: ['filters', AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID]
    })

    expect(selected?.evidence?.map((section) => section.id)).toEqual(['overview'])
    expect(selected?.state).toEqual({ filters: { keyword: 'shop' } })
    expect(snapshot.evidence?.map((section) => section.id)).toEqual(['filters', 'overview'])
    expect(snapshot.state?.unsavedChanges).toEqual({ kind: 'condition-rule', entryCount: 1 })
    expect(selectAIContextSnapshot(snapshot, { enabled: false })).toBeUndefined()
  })
})
