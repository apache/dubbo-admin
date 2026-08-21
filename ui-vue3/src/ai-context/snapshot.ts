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

import { sanitizeContextValue } from './sanitize'
import {
  AI_CONTEXT_DEFAULT_MAX_BYTES,
  AI_CONTEXT_VERSION,
  type AIContextBase,
  type AIContextContribution,
  type AIContextSection,
  type AIContextSnapshot,
  type AIContextSnapshotOptions
} from './types'

export interface PrioritizedContribution {
  id: string
  priority: number
  contribution: AIContextContribution
}

export const getSerializedSize = (value: unknown): number => {
  return new TextEncoder().encode(JSON.stringify(value)).byteLength
}

const normalizeEvidence = (
  contribution: PrioritizedContribution,
  capturedAt: string
): AIContextSection[] => {
  const evidence = contribution.contribution.evidence
  if (!evidence) return []

  const sections = Array.isArray(evidence) ? evidence : [evidence]
  return sections.map((section) => ({
    ...section,
    capturedAt: section.capturedAt || capturedAt,
    priority: section.priority ?? contribution.priority
  }))
}

const applyBudget = (snapshot: AIContextSnapshot, maxBytes: number): AIContextSnapshot => {
  if (getSerializedSize(snapshot) <= maxBytes) return snapshot

  const result: AIContextSnapshot = {
    ...snapshot,
    evidence: snapshot.evidence ? [...snapshot.evidence] : undefined,
    truncation: {
      truncated: true,
      omittedSections: []
    }
  }

  // Evidence is ordered by descending priority, so remove optional low-priority sections first.
  while (result.evidence?.length && getSerializedSize(result) > maxBytes) {
    const omitted = result.evidence.pop()
    if (omitted) result.truncation?.omittedSections.push(omitted.id)
  }

  if (getSerializedSize(result) > maxBytes && result.state) {
    delete result.state
    result.truncation?.omittedSections.push('state')
  }

  if (getSerializedSize(result) > maxBytes && result.page.query) {
    delete result.page.query
    result.truncation?.omittedSections.push('page.query')
  }

  if (getSerializedSize(result) > maxBytes && result.page.params) {
    delete result.page.params
    result.truncation?.omittedSections.push('page.params')
  }

  if (getSerializedSize(result) > maxBytes && result.page.fullPath) {
    delete result.page.fullPath
    result.truncation?.omittedSections.push('page.fullPath')
  }

  if (!result.evidence?.length) delete result.evidence
  return result
}

export const createAIContextSnapshot = (
  base: AIContextBase,
  contributions: PrioritizedContribution[],
  options: AIContextSnapshotOptions = {}
): AIContextSnapshot => {
  const capturedAt = (options.now?.() || new Date()).toISOString()
  const sortedAscending = [...contributions].sort(
    (a, b) => a.priority - b.priority || a.id.localeCompare(b.id)
  )

  const scope = { ...base.scope }
  const state = {}
  const evidence: AIContextSection[] = []

  for (const item of sortedAscending) {
    Object.assign(scope, item.contribution.scope)
    Object.assign(state, item.contribution.state)
    evidence.push(...normalizeEvidence(item, capturedAt))
  }

  evidence.sort((a, b) => (b.priority || 0) - (a.priority || 0) || a.id.localeCompare(b.id))

  const rawSnapshot: AIContextSnapshot = {
    version: AI_CONTEXT_VERSION,
    capturedAt,
    global: base.global,
    page: base.page,
    scope,
    ...(Object.keys(state).length ? { state } : {}),
    ...(evidence.length ? { evidence } : {})
  }

  // Sanitize before measuring so the transmitted snapshot itself satisfies the byte budget.
  const sanitized = sanitizeContextValue(rawSnapshot) as AIContextSnapshot
  return applyBudget(sanitized, options.maxBytes ?? AI_CONTEXT_DEFAULT_MAX_BYTES)
}
