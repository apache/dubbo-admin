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

const normalizeText = (value: unknown): string | undefined => {
  const candidate = Array.isArray(value) ? value[0] : value
  if (typeof candidate !== 'string' && typeof candidate !== 'number') return undefined
  const normalized = String(candidate).trim()
  return normalized || undefined
}

export const createConfigurationStateContribution = (
  section: unknown,
  form: unknown
): AIContextContribution | undefined => {
  if (!form || typeof form !== 'object' || Array.isArray(form)) return undefined

  const values = sanitizeContextValue(form, {
    maxArrayItems: 10,
    maxDepth: 4,
    maxStringLength: 500
  })

  return {
    evidence: {
      id: 'configuration-state',
      source: 'configuration-form',
      data: {
        section: normalizeText(section),
        values
      }
    }
  }
}

const getDashboardTimeRange = (url: unknown): Record<string, string> | undefined => {
  if (typeof url !== 'string' || !url) return undefined

  // The embedded dashboard may be cross-origin; only retain safe URL metadata owned by this page.
  try {
    const parsed = new URL(url)
    const timeRange: Record<string, string> = {}
    for (const key of ['from', 'to', 'refresh']) {
      const value = parsed.searchParams.get(key)
      if (value) timeRange[key] = value
    }
    return Object.keys(timeRange).length ? timeRange : undefined
  } catch {
    return undefined
  }
}

export const createDashboardStateContribution = (
  dashboard: unknown,
  params: unknown,
  url?: unknown
): AIContextContribution => ({
  evidence: {
    id: 'dashboard-state',
    source: 'dashboard-page',
    data: {
      dashboard: normalizeText(dashboard),
      loaded: typeof url === 'string' && Boolean(url),
      parameters: sanitizeContextValue(params, { maxDepth: 3, maxArrayItems: 10 }),
      timeRange: getDashboardTimeRange(url)
    }
  }
})

export const createResourceDetailsContribution = (
  id: string,
  source: string,
  value: unknown,
  fields: readonly string[]
): AIContextContribution | undefined => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined

  const record = value as Record<string, unknown>
  const selected: Record<string, unknown> = {}
  // Page components provide an explicit allowlist instead of forwarding complete API responses.
  for (const field of fields) {
    const fieldValue = record[field]
    if (fieldValue !== undefined && fieldValue !== null && fieldValue !== '') {
      selected[field] = fieldValue
    }
  }
  if (!Object.keys(selected).length) return undefined

  return {
    evidence: {
      id,
      source,
      data: sanitizeContextValue(selected, {
        maxArrayItems: 10,
        maxDepth: 5,
        maxStringLength: 500
      }) as Record<string, unknown>
    }
  }
}

export interface TopologyContextInput {
  nodes?: unknown
  edges?: unknown
}

export interface TopologySelectionInput {
  key?: unknown
  type?: unknown
  detail?: unknown
}

export const createTopologyStateContribution = (
  graph: TopologyContextInput | undefined,
  selection?: TopologySelectionInput
): AIContextContribution | undefined => {
  if (!graph || (!Array.isArray(graph.nodes) && !Array.isArray(graph.edges))) return undefined

  // Graph nodes may carry large framework metadata, so keep only identity and connectivity fields.
  const nodes = (Array.isArray(graph.nodes) ? graph.nodes : []).slice(0, 10).map((node) => {
    if (!node || typeof node !== 'object' || Array.isArray(node)) return {}
    const item = node as Record<string, unknown>
    return {
      id: normalizeText(item.id),
      label: normalizeText(item.label),
      type: normalizeText(item.type)
    }
  })
  const edges = (Array.isArray(graph.edges) ? graph.edges : []).slice(0, 10).map((edge) => {
    if (!edge || typeof edge !== 'object' || Array.isArray(edge)) return {}
    const item = edge as Record<string, unknown>
    return {
      source: normalizeText(item.source),
      target: normalizeText(item.target)
    }
  })

  return {
    evidence: {
      id: 'topology-state',
      source: 'topology-page',
      data: {
        nodeCount: Array.isArray(graph.nodes) ? graph.nodes.length : 0,
        edgeCount: Array.isArray(graph.edges) ? graph.edges.length : 0,
        nodes,
        edges,
        selectedNode: normalizeText(selection?.key),
        selectedNodeType: normalizeText(selection?.type),
        selectedNodeDetail: sanitizeContextValue(selection?.detail, {
          maxArrayItems: 10,
          maxDepth: 4,
          maxStringLength: 500
        })
      }
    }
  }
}

export const createEventListContribution = (events: unknown): AIContextContribution | undefined => {
  if (!Array.isArray(events)) return undefined

  const items = events.slice(0, 10).map((event) => {
    if (!event || typeof event !== 'object' || Array.isArray(event)) return {}
    const item = event as Record<string, unknown>
    return sanitizeContextValue(
      {
        type: item.type,
        description: item.description ?? item.desc,
        status: item.status,
        time: item.time ?? item.timestamp
      },
      { maxDepth: 2, maxStringLength: 500 }
    )
  })

  return {
    evidence: {
      id: 'event-list',
      source: 'event-api',
      data: {
        total: events.length,
        includedCount: items.length,
        truncated: events.length > items.length,
        events: items
      }
    }
  }
}

const parseJsonContext = (value: unknown): unknown => {
  if (typeof value !== 'string' || !value.trim()) return undefined
  try {
    // Parse debug payloads before sanitizing so nested credentials are still detected.
    return sanitizeContextValue(JSON.parse(value), {
      maxArrayItems: 10,
      maxDepth: 5,
      maxStringLength: 500
    })
  } catch {
    return { parseError: true }
  }
}

export interface ServiceDebugContextInput {
  methods?: unknown
  providers?: unknown
  selectedInstance?: unknown
  method?: unknown
  request?: unknown
  response?: unknown
  elapsedMs?: unknown
  timeoutMs?: unknown
  attachmentKeys?: unknown
}

export const createServiceDebugContribution = (
  input: ServiceDebugContextInput
): AIContextContribution => {
  const methods = Array.isArray(input.methods)
    ? input.methods.slice(0, 10).map((method) => {
        if (!method || typeof method !== 'object' || Array.isArray(method)) return {}
        const item = method as Record<string, unknown>
        return {
          methodName: normalizeText(item.methodName),
          signature: normalizeText(item.signature),
          parameterTypes: sanitizeContextValue(item.parameterTypes, { maxDepth: 2 })
        }
      })
    : []
  const providers = Array.isArray(input.providers)
    ? input.providers.slice(0, 10).map((provider) => {
        if (!provider || typeof provider !== 'object' || Array.isArray(provider)) return {}
        const item = provider as Record<string, unknown>
        return {
          name: normalizeText(item.name),
          application: normalizeText(item.appName),
          ip: normalizeText(item.ip)
        }
      })
    : []

  return {
    evidence: {
      id: 'service-debug',
      source: 'service-debug-page',
      data: {
        methodCount: Array.isArray(input.methods) ? input.methods.length : 0,
        methods,
        providerCount: Array.isArray(input.providers) ? input.providers.length : 0,
        providers,
        selectedInstance: normalizeText(input.selectedInstance),
        selectedMethod: sanitizeContextValue(input.method, { maxDepth: 4, maxArrayItems: 10 }),
        request: parseJsonContext(input.request),
        response: parseJsonContext(input.response),
        elapsedMs: input.elapsedMs,
        timeoutMs: input.timeoutMs,
        attachmentKeys: sanitizeContextValue(input.attachmentKeys, { maxDepth: 2 })
      }
    }
  }
}
