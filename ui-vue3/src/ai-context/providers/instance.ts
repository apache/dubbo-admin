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
import type { AIContextContribution } from '../types'

const MAX_LIST_ITEMS = 10
const MAX_LABELS = 10

type ContextScalar = string | number | boolean

const normalizeScalar = (value: unknown): ContextScalar | undefined => {
  if (typeof value === 'string') {
    const normalized = value.trim()
    return normalized || undefined
  }
  if (typeof value === 'number') return Number.isFinite(value) ? value : undefined
  if (typeof value === 'boolean') return value
  return undefined
}

const formatList = (value: unknown): string | undefined => {
  if (!Array.isArray(value)) return undefined

  const items = value
    .map(normalizeScalar)
    .filter((item): item is ContextScalar => item !== undefined)
    .slice(0, MAX_LIST_ITEMS)
  if (!items.length) return undefined

  const suffix = value.length > items.length ? ` (+${value.length - items.length} more)` : ''
  return `${items.join(', ')}${suffix}`
}

const formatLabels = (value: unknown): string | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined

  const entries = Object.entries(value)
    .sort(([left], [right]) => left.localeCompare(right))
    .slice(0, MAX_LABELS)
  if (!entries.length) return undefined

  const labels = Object.fromEntries(entries)
  const sanitized = sanitizeContextValue(labels, { maxDepth: 2 })
  const suffix =
    Object.keys(value).length > entries.length
      ? ` (+${Object.keys(value).length - entries.length} more)`
      : ''
  return `${JSON.stringify(sanitized)}${suffix}`
}

const formatProbe = (value: unknown): string | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined

  const probe = value as Record<string, unknown>
  const details = [
    ['open', normalizeScalar(probe.open)],
    ['type', normalizeScalar(probe.type)],
    ['port', normalizeScalar(probe.port)]
  ]
    .filter((entry): entry is [string, ContextScalar] => entry[1] !== undefined)
    .map(([key, entryValue]) => `${key}=${entryValue}`)

  return details.length ? details.join(', ') : undefined
}

export const createInstanceDetailContribution = (
  value: unknown
): AIContextContribution | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined

  const detail = value as Record<string, unknown>
  const probes =
    detail.probes && typeof detail.probes === 'object' && !Array.isArray(detail.probes)
      ? (detail.probes as Record<string, unknown>)
      : {}
  const data: Record<string, ContextScalar> = {}

  // Flatten only operational fields that are useful for diagnosing this instance.
  const add = (key: string, fieldValue: unknown) => {
    const normalized = normalizeScalar(fieldValue)
    if (normalized !== undefined) data[key] = normalized
  }

  add('application', detail.appName)
  add('ip', detail.ip)
  add('rpcPort', detail.rpcPort)
  add('lifecycleState', detail.lifecycleState)
  add('registerState', detail.registerState ?? detail.registerStates)
  add('deployState', detail.deployState)
  add('registeredAt', detail.registerTime)
  add('startedAt', detail.startTime)
  add('readyAt', detail.readyTime)
  add('createdAt', detail.createTime)
  add('deployCluster', detail.deployCluster)
  add('registerClusters', formatList(detail.registerClusters))
  add('node', detail.node)
  add('workloadName', detail.workloadName)
  add('image', detail.image)
  add('labels', formatLabels(detail.labels))
  add('startupProbe', formatProbe(probes.startupProbe))
  add('readinessProbe', formatProbe(probes.readinessProbe))
  add('livenessProbe', formatProbe(probes.livenessProbe))

  if (!Object.keys(data).length) return undefined

  return {
    evidence: {
      id: 'instance-detail',
      source: 'instance-detail-api',
      data
    }
  }
}
