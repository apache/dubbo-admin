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
    }
  }
}
