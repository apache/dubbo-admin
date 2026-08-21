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

export interface SearchResultColumn {
  key?: unknown
  dataIndex?: unknown
  __hide?: unknown
}

export interface SearchResultPaging {
  curPage?: unknown
  pageSize?: unknown
  total?: unknown
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

const normalizeResultCell = (value: unknown): SearchFilterValue | undefined => {
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return normalizeFilterValue(value)
  }
  if (!Array.isArray(value)) return undefined

  const items = value
    .map(normalizeFilterValue)
    .filter((item): item is SearchFilterValue => item !== undefined)
  return items.length ? items : undefined
}

export const createSearchResultsContribution = (
  rows: unknown,
  columns: readonly SearchResultColumn[] | undefined,
  paging?: SearchResultPaging
): AIContextContribution | undefined => {
  if (!Array.isArray(rows)) return undefined

  // Mirror what the user can inspect: visible columns and at most the first ten displayed rows.
  const columnKeys = (columns || [])
    .filter((column) => column.__hide !== true)
    .map((column) => column.dataIndex ?? column.key)
    .filter((key): key is string => typeof key === 'string' && key !== 'idx')
  const visibleRows = rows.slice(0, 10).map((row) => {
    if (!row || typeof row !== 'object' || Array.isArray(row)) return {}

    const source = row as Record<string, unknown>
    const result: Record<string, SearchFilterValue> = {}
    for (const key of columnKeys) {
      const value = normalizeResultCell(source[key])
      if (value !== undefined) result[key] = value
    }
    return result
  })
  const total = normalizeFilterValue(paging?.total) ?? rows.length

  return {
    evidence: {
      id: 'search-results',
      source: 'search-result-table',
      data: {
        total,
        currentPage: normalizeFilterValue(paging?.curPage),
        pageSize: normalizeFilterValue(paging?.pageSize),
        displayedCount: rows.length,
        includedCount: visibleRows.length,
        truncated: rows.length > visibleRows.length,
        rows: visibleRows
      }
    }
  }
}
