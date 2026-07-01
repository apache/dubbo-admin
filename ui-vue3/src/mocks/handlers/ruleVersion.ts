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

import { http, HttpResponse, type HttpHandler } from 'msw'
import { success, base } from '../utils'
import type {
  RuleVersion,
  RuleVersionDiff,
  RuleVersionList,
  TrafficRuleKind
} from '@/api/service/traffic'

const KINDS: TrafficRuleKind[] = ['condition-rule', 'tag-rule', 'configurator']
const SCENARIOS = ['normal', 'deleted', 'empty', 'backend-error', 'diff'] as const

type Scenario = (typeof SCENARIOS)[number]

const scenarioOf = (ruleName: string): Scenario =>
  SCENARIOS.find((scenario) => ruleName.includes(`-${scenario}`)) ?? 'normal'

const decodeName = (raw: string) => {
  try {
    return decodeURIComponent(raw)
  } catch {
    return raw
  }
}

const spec = (ruleName: string, marker: string) =>
  JSON.stringify({ configVersion: 'v3.0', key: ruleName, marker })

const version = (
  kind: TrafficRuleKind,
  ruleName: string,
  id: string,
  versionNo: number,
  operation: RuleVersion['operation'],
  source: RuleVersion['source'],
  isLatestRecorded: boolean,
  marker = `v${versionNo}`
): RuleVersion => ({
  id,
  ruleKind: kind,
  mesh: 'default',
  resourceKey: `/${ruleName}`,
  ruleName,
  versionNo,
  contentHash: `sha256:${ruleName}:${marker}`,
  specJson: spec(ruleName, marker),
  source,
  operation,
  author: 'user name',
  createdAt: `2026-05-${(20 + versionNo).toString().padStart(2, '0')}T08:00:00Z`,
  recordedAt: `2026-05-${(20 + versionNo).toString().padStart(2, '0')}T08:01:00Z`,
  isLatestRecorded
})

const fixtureVersions = (kind: TrafficRuleKind, ruleName: string): RuleVersion[] => {
  switch (scenarioOf(ruleName)) {
    case 'empty':
      return []
    case 'deleted':
      return [
        version(kind, ruleName, '2003', 3, 'DELETE', 'ADMIN', false, 'deleted'),
        version(kind, ruleName, '2002', 2, 'UPDATE', 'ADMIN', false),
        version(kind, ruleName, '2001', 1, 'CREATE', 'BOOTSTRAP', false)
      ]
    case 'diff':
      return [
        version(kind, ruleName, '4002', 2, 'UPDATE', 'ADMIN', true, 'right'),
        version(kind, ruleName, '4001', 1, 'CREATE', 'BOOTSTRAP', false, 'left')
      ]
    default:
      return [
        version(kind, ruleName, '1005', 5, 'UPDATE', 'ADMIN', true),
        version(kind, ruleName, '1004', 4, 'UPDATE', 'ADMIN', false),
        version(kind, ruleName, '1003', 3, 'UPDATE', 'ADMIN', false),
        version(kind, ruleName, '1002', 2, 'UPDATE', 'ADMIN', false),
        version(kind, ruleName, '1001', 1, 'CREATE', 'BOOTSTRAP', false)
      ]
  }
}

const latestRecordedVersionOf = (versions: RuleVersion[]) =>
  versions.find((version) => version.isLatestRecorded)

const versionList = (versions: RuleVersion[]): RuleVersionList => {
  const latestRecorded = latestRecordedVersionOf(versions)
  const head = versions[0]
  return {
    items: versions,
    total: versions.length,
    latestRecordedVersionId: latestRecorded?.id,
    latestRecordedVersionNo: latestRecorded?.versionNo,
    latestRecordedDeleted: Boolean(head?.operation === 'DELETE')
  }
}

const bizError = (code: string, message: string, status = 200) =>
  HttpResponse.json({ code, message, data: null }, { status })

const notFoundResp = (message: string) => bizError('NotFoundError', message, 404)

const readJsonBody = async (request: Request): Promise<Record<string, unknown>> => {
  try {
    const body = (await request.json()) as Record<string, unknown> | null
    return body ?? {}
  } catch {
    return {}
  }
}

const validateReason = (reason: string) => {
  const trimmed = reason.trim()
  if (!trimmed) return bizError('InvalidArgument', 'reason must not be empty', 400)
  if (trimmed.length > 1024)
    return bizError('InvalidArgument', 'reason must be at most 1024 characters', 400)
  return null
}

const buildVersionHandlersForKind = (kind: TrafficRuleKind): HttpHandler[] => [
  http.get(`${base}/${kind}/:ruleName/versions`, ({ params }) => {
    const ruleName = decodeName(params.ruleName as string)
    if (scenarioOf(ruleName) === 'backend-error')
      return bizError('InternalError', 'backend error', 500)
    const versions = fixtureVersions(kind, ruleName)
    return success<RuleVersionList>(versionList(versions))
  }),

  http.get(`${base}/${kind}/:ruleName/versions/:versionId`, ({ params }) => {
    const ruleName = decodeName(params.ruleName as string)
    const versionId = String(params.versionId || '').trim()
    if (!versionId) return bizError('InvalidArgument', 'versionId must be an integer', 400)
    const found = fixtureVersions(kind, ruleName).find((item) => item.id === versionId)
    return found ? success(found) : notFoundResp('rule version not found')
  }),

  http.get(`${base}/${kind}/:ruleName/versions/:versionId/diff`, ({ params, request }) => {
    const ruleName = decodeName(params.ruleName as string)
    const versionId = String(params.versionId || '').trim()
    if (!versionId) return bizError('InvalidArgument', 'versionId must be an integer', 400)
    const versions = fixtureVersions(kind, ruleName)
    const left = versions.find((item) => item.id === versionId)
    if (!left) return notFoundResp('rule version not found')
    const against = new URL(request.url).searchParams.get('against') || 'current'
    if (against !== 'current' && against !== 'previous' && !/^\d+$/.test(against)) {
      return bizError(
        'InvalidArgument',
        "against must be 'current', 'previous', or a version ID",
        400
      )
    }
    const leftIndex = versions.findIndex((item) => item.id === versionId)
    const right =
      against === 'current'
        ? latestRecordedVersionOf(versions)
        : against === 'previous'
          ? versions[leftIndex + 1]
          : versions.find((item) => item.id === against)
    if (!right) return notFoundResp('rule version not found')
    return success<RuleVersionDiff>({
      left: { id: left.id, versionNo: left.versionNo, specJson: left.specJson },
      right: { id: right.id, versionNo: right.versionNo, specJson: right.specJson }
    })
  }),

  http.post(
    `${base}/${kind}/:ruleName/versions/:versionId/rollback`,
    async ({ params, request }) => {
      const ruleName = decodeName(params.ruleName as string)
      const body = await readJsonBody(request)
      const reasonErr = validateReason(typeof body.reason === 'string' ? body.reason : '')
      if (reasonErr) return reasonErr

      const versions = fixtureVersions(kind, ruleName)
      const target = versions.find((item) => item.id === String(params.versionId || '').trim())
      if (!target) return notFoundResp('rule version not found')
      if (target.operation === 'DELETE')
        return bizError('InvalidArgument', 'cannot roll back to a deleted rule version', 400)
      const latestRecorded = latestRecordedVersionOf(versions)

      return success({
        rolledBackFromId: target.id,
        versionId: '9901',
        versionNo: (latestRecorded?.versionNo ?? 0) + 1,
        source: 'ROLLBACK',
        historyRecorded: true
      })
    }
  )
]

export const ruleVersionHandlers: HttpHandler[] = KINDS.flatMap(buildVersionHandlersForKind)

export const ruleVersionMock = {
  scenarios: SCENARIOS,
  scenarioOf,
  reset() {}
}
