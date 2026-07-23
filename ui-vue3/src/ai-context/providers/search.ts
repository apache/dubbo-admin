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

import type { AIContextContribution } from '../types'

export interface SearchFilterParam {
  param?: unknown
}

type SearchFilterValue = string | number | boolean | SearchFilterValue[]

const normalizeFilterValue = (value: unknown): SearchFilterValue | undefined => {
  if (typeof value === 'string') {
    const normalized = value.trim()
    return normalized || undefined
  }
  if (typeof value === 'number') return Number.isFinite(value) ? value : undefined
  if (typeof value === 'boolean') return value
  if (!Array.isArray(value)) return undefined

  const normalized = value
    .map(normalizeFilterValue)
    .filter((item): item is SearchFilterValue => item !== undefined)
  return normalized.length ? normalized : undefined
}

export const createSearchFiltersContribution = (
  params: readonly SearchFilterParam[] | undefined,
  queryForm: unknown
): AIContextContribution | undefined => {
  if (!params?.length || !queryForm || typeof queryForm !== 'object' || Array.isArray(queryForm)) {
    return undefined
  }

  const query = queryForm as Record<string, unknown>
  const filters: Record<string, SearchFilterValue> = {}

  for (const item of params) {
    if (typeof item.param !== 'string' || !item.param) continue
    const value = normalizeFilterValue(query[item.param])
    if (value !== undefined) filters[item.param] = value
  }

  return Object.keys(filters).length ? { state: { filters } } : undefined
}
