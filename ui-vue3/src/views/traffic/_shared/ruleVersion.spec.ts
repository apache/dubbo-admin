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

import { beforeAll, describe, expect, it } from 'vitest'
import type { RuleVersion } from '@/api/service/traffic'

let helpers: typeof import('./ruleVersion')
let ruleVersionMock: typeof import('@/mocks/handlers/ruleVersion').ruleVersionMock

beforeAll(async () => {
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: () => null,
      setItem: () => undefined,
      removeItem: () => undefined
    },
    configurable: true
  })
  helpers = await import('./ruleVersion')
  ruleVersionMock = (await import('@/mocks/handlers/ruleVersion')).ruleVersionMock
})

const version = (
  id: string,
  isCurrent: boolean,
  operation: RuleVersion['operation'] = 'UPDATE'
): RuleVersion => ({
  id,
  ruleKind: 'ConditionRoute',
  mesh: '',
  resourceKey: `/demo-${id}`,
  ruleName: `demo-${id}`,
  versionNo: 1,
  contentHash: `hash-${id}`,
  specJson: '{}',
  source: 'ADMIN',
  operation,
  author: 'admin',
  createdAt: '2026-06-19T00:00:00Z',
  isCurrent
})

describe('ruleVersion helpers', () => {
  it('preserves int64 IDs as strings in current state', () => {
    const state = helpers.currentVersionStateFromItems([version('7473321752550968337', true)])

    expect(state.id).toBe('7473321752550968337')
    expect(helpers.rollbackExpectedVersionId(state)).toBe('7473321752550968337')
  })

  it('uses 0 as rollback CAS token for deleted state', () => {
    const state = helpers.currentVersionStateFromItems([version('7473321752550968337', false)])

    expect(state.deleted).toBe(true)
    expect(state.id).toBeUndefined()
    expect(helpers.rollbackExpectedVersionId(state)).toBe('0')
  })

  it('treats latest DELETE ledger event as deleted current state', () => {
    const state = helpers.currentVersionStateFromItems([
      version('7473321752550968338', false, 'DELETE'),
      version('7473321752550968337', false, 'UPDATE')
    ])

    expect(state.deleted).toBe(true)
    expect(state.id).toBeUndefined()
    expect(helpers.rollbackExpectedVersionId(state)).toBe('0')
  })

  it('uses currentDeleted state instead of an unreachable current DELETE branch', () => {
    const state = helpers.currentVersionStateFromItems([
      version('delete-marker', false, 'DELETE'),
      version('previous', false, 'UPDATE')
    ])

    expect(state).toEqual({ id: undefined, versionNo: undefined, deleted: true })
  })

  it('separates conflict and pending error classification', () => {
    expect(helpers.isVersionConflict({ code: 'VERSION_CONFLICT' })).toBe(true)
    expect(helpers.isVersionConflict({ code: 'VERSION_LEDGER_PENDING' })).toBe(false)
    expect(helpers.isVersionLedgerPending({ code: 'VERSION_LEDGER_PENDING' })).toBe(true)
    expect(helpers.isVersionLedgerPending({ code: 'VERSION_CONFLICT' })).toBe(false)
  })

  it('rejects stale history responses by request sequence', () => {
    expect(helpers.isCurrentHistoryRequest(2, 2, false)).toBe(true)
    expect(helpers.isCurrentHistoryRequest(1, 2, false)).toBe(false)
    expect(helpers.isCurrentHistoryRequest(2, 2, true)).toBe(false)
  })

  it('does not swallow non-versioning errors', () => {
    expect(helpers.notifyRuleVersionError({ code: 'InternalError', message: 'boom' })).toBe(false)
    expect(helpers.ruleVersionErrorMessage({ code: 'InternalError', message: 'boom' })).toBe('boom')
  })

  it('requires a non-empty abandon reason after trimming', () => {
    expect(helpers.normalizeIntentReason('   ')).toBe('')
    expect(helpers.normalizeIntentReason('  operator cleanup  ')).toBe('operator cleanup')
  })

  it('formats Monaco diff labels for target and current sides', () => {
    expect(helpers.versionDiffLabel('Target version', 7)).toBe('Target version v7')
    expect(helpers.versionDiffLabel('Current deleted')).toBe('Current deleted')
  })

  it('keeps rule version mock as named scenario fixtures', () => {
    expect([...ruleVersionMock.scenarios]).toEqual([
      'normal',
      'deleted',
      'empty',
      'conflict',
      'pending',
      'repair-success',
      'repair-failure',
      'abandon-success',
      'backend-error',
      'diff',
      'rollback-success'
    ])
    expect(ruleVersionMock.scenarioOf('demo-conflict')).toBe('conflict')
    expect(ruleVersionMock.shouldConflict('demo-conflict')).toBe(true)
    expect(ruleVersionMock.shouldPend('demo-repair-success')).toBe(true)
    expect(ruleVersionMock.shouldPend('demo-normal')).toBe(false)
  })
})
