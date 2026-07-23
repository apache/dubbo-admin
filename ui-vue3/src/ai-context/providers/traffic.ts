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

import { sanitizeContextValue } from '../sanitize'
import type { AIContextContribution, AIContextScope } from '../types'

export type TrafficDraftKind = 'condition-rule' | 'tag-rule' | 'dynamic-config'
export type TrafficDraftMode = 'create' | 'update'
export type TrafficDraftRepresentation = 'form' | 'yaml'

export interface TrafficDraftOptions {
  kind: TrafficDraftKind
  mode: TrafficDraftMode
  representation: TrafficDraftRepresentation
  rule?: unknown
  draft?: unknown
  version?: unknown
  group?: unknown
}

const asRecord = (value: unknown): Record<string, unknown> | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
  return value as Record<string, unknown>
}

const normalizeIdentifier = (value: unknown): string | undefined => {
  const candidate = Array.isArray(value) ? value[0] : value
  if (typeof candidate !== 'string') return undefined

  const normalized = candidate.trim()
  return normalized || undefined
}

const getBoolean = (value: unknown): boolean | undefined => {
  return typeof value === 'boolean' ? value : undefined
}

const getArrayLength = (value: unknown): number | undefined => {
  return Array.isArray(value) ? value.length : undefined
}

const getDraftData = (
  kind: TrafficDraftKind,
  draft: unknown
): { data?: Record<string, unknown>; entryCount?: number } => {
  const draftRecord = asRecord(draft)
  if (!draftRecord) return {}

  if (kind === 'dynamic-config') {
    return {
      data: asRecord(draftRecord.basicInfo),
      entryCount: getArrayLength(draftRecord.config)
    }
  }

  return {
    data: draftRecord,
    entryCount: getArrayLength(
      kind === 'condition-rule' ? draftRecord.conditions : draftRecord.tags
    )
  }
}

const selectFields = (
  value: Record<string, unknown> | undefined,
  fields: readonly string[]
): Record<string, unknown> => {
  const result: Record<string, unknown> = {}
  if (!value) return result

  for (const field of fields) {
    const fieldValue = value[field]
    if (fieldValue !== undefined && fieldValue !== null && fieldValue !== '') {
      result[field] = fieldValue
    }
  }
  return result
}

const createDynamicConfigContent = (draft: Record<string, unknown>): Record<string, unknown> => {
  const basicInfo = selectFields(asRecord(draft.basicInfo), [
    'configVersion',
    'ruleName',
    'scope',
    'key',
    'enabled'
  ])
  const config = Array.isArray(draft.config)
    ? draft.config.slice(0, 10).map((item) => {
        const record = asRecord(item)
        const parametersValue = asRecord(record?.parametersValue)
        const builtinParameters: Record<string, unknown> = {}
        // Custom parameters are open-ended; only known diagnostic fields are eligible for context.
        for (const key of ['retries', 'timeout', 'accesslog', 'weight']) {
          const parameter = asRecord(parametersValue?.[key])
          if (parameter?.value !== undefined && parameter.value !== '') {
            builtinParameters[key] = parameter.value
          }
        }

        return {
          ...selectFields(record, ['enabled', 'side', 'matchesKeys', 'parametersKeys']),
          matchesValue: record?.matchesValue,
          builtinParameters
        }
      })
    : []

  return { basicInfo, config, truncated: Array.isArray(draft.config) && draft.config.length > 10 }
}

const createTrafficRuleContent = (
  kind: TrafficDraftKind,
  draft: unknown
): Record<string, unknown> | undefined => {
  const record = asRecord(draft)
  if (!record) return undefined

  // Each rule kind has its own allowlist. Recursive sanitization remains the final safety layer.
  const content =
    kind === 'condition-rule'
      ? selectFields(record, [
          'configVersion',
          'scope',
          'key',
          'enabled',
          'runtime',
          'force',
          'conditions'
        ])
      : kind === 'tag-rule'
        ? selectFields(record, ['configVersion', 'scope', 'key', 'enabled', 'runtime', 'tags'])
        : createDynamicConfigContent(record)

  return Object.keys(content).length
    ? (sanitizeContextValue(content, {
        maxArrayItems: 10,
        maxDepth: 6,
        maxStringLength: 500
      }) as Record<string, unknown>)
    : undefined
}

export const createTrafficRuleContentContribution = (
  kind: TrafficDraftKind,
  draft: unknown
): AIContextContribution | undefined => {
  const content = createTrafficRuleContent(kind, draft)
  if (!content) return undefined

  return {
    evidence: {
      id: 'rule-content',
      source: 'traffic-rule-page',
      data: {
        kind,
        content
      }
    }
  }
}

const createRuleName = (
  options: TrafficDraftOptions,
  data: Record<string, unknown> | undefined,
  scope: string | undefined,
  key: string | undefined
): string | undefined => {
  const explicitRule = normalizeIdentifier(options.rule)
  if (explicitRule && explicitRule !== '_tmp') return explicitRule

  const draftRule = normalizeIdentifier(data?.ruleName)
  if (draftRule && draftRule !== '_tmp') return draftRule
  if (!key) return undefined

  if (options.kind === 'condition-rule' && scope === 'service') {
    const version = normalizeIdentifier(options.version)
    const group = normalizeIdentifier(options.group)
    return version || group ? `${key}:${version || ''}:${group || ''}` : key
  }

  return options.kind === 'dynamic-config' ? `${key}.configurators` : key
}

export const createTrafficRuleResourceContribution = (
  rule: unknown
): AIContextContribution | undefined => {
  const normalizedRule = normalizeIdentifier(rule)
  if (!normalizedRule || normalizedRule === '_tmp') return undefined

  return {
    scope: {
      rule: normalizedRule
    }
  }
}

export const createTrafficDraftContribution = (
  options: TrafficDraftOptions
): AIContextContribution => {
  const { data, entryCount } = getDraftData(options.kind, options.draft)
  const scopeName = normalizeIdentifier(data?.scope)
  const key = normalizeIdentifier(data?.key)
  const rule = createRuleName(options, data, scopeName, key)
  const enabled = getBoolean(data?.enabled)
  const runtime = getBoolean(data?.runtime)
  const force = getBoolean(data?.force)

  const scope: Partial<AIContextScope> = {
    ...(rule ? { rule } : {}),
    ...(scopeName === 'application' && key ? { application: key } : {}),
    ...(scopeName === 'service' && key ? { service: key } : {})
  }
  const content = createTrafficRuleContent(options.kind, options.draft)

  return {
    ...(Object.keys(scope).length ? { scope } : {}),
    state: {
      unsavedChanges: {
        kind: options.kind,
        mode: options.mode,
        representation: options.representation,
        ...(scopeName ? { scope: scopeName } : {}),
        ...(key ? { key } : {}),
        ...(enabled !== undefined ? { enabled } : {}),
        ...(runtime !== undefined ? { runtime } : {}),
        ...(force !== undefined ? { force } : {}),
        ...(entryCount !== undefined ? { entryCount } : {})
      }
    },
    ...(content
      ? {
          evidence: {
            id: 'rule-content',
            source: 'traffic-rule-draft',
            data: { kind: options.kind, content }
          }
        }
      : {})
  }
}
