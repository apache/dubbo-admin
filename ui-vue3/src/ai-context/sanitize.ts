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

const REDACTED_VALUE = '[REDACTED]'
const CIRCULAR_VALUE = '[CIRCULAR]'
const MAX_STRING_LENGTH = 1000
const MAX_ARRAY_ITEMS = 10
const MAX_DEPTH = 12

const SENSITIVE_KEYS = [
  'password',
  'passwd',
  'token',
  'secret',
  'cookie',
  'authorization',
  'apikey',
  'privatekey',
  'kubeconfig'
]

const SENSITIVE_DESCRIPTOR_KEYS = new Set(['key', 'name'])
const SEMANTIC_VALUE_KEYS = new Set(['value', 'values', 'currentvalue', 'defaultvalue'])

export interface SanitizeOptions {
  maxStringLength?: number
  maxArrayItems?: number
  maxDepth?: number
}

const normalizeKey = (key: string): string => key.toLowerCase().replace(/[^a-z0-9]/g, '')

const isSensitiveKey = (key: string): boolean => {
  const normalized = normalizeKey(key)
  return SENSITIVE_KEYS.some((sensitiveKey) => normalized.includes(sensitiveKey))
}

const hasSensitiveDescriptor = (value: Record<string, unknown>): boolean => {
  // Configuration APIs often encode secrets as { key: 'token', value: '...' }.
  // In that shape the value field is harmless by name, so inspect its sibling descriptor.
  return Object.entries(value).some(
    ([key, item]) =>
      SENSITIVE_DESCRIPTOR_KEYS.has(normalizeKey(key)) &&
      typeof item === 'string' &&
      isSensitiveKey(item)
  )
}

const limitString = (value: string, maxLength: number): string => {
  if (value.length <= maxLength) return value
  return `${value.slice(0, maxLength)}...[TRUNCATED]`
}

const sanitizeUrl = (value: string): string => {
  if (!/^[a-z][a-z\d+.-]*:\/\//i.test(value)) return value

  try {
    const url = new URL(value)
    if (url.username) url.username = ''
    if (url.password) url.password = ''

    for (const key of [...url.searchParams.keys()]) {
      if (isSensitiveKey(key) || key.toLowerCase() === 'username') {
        url.searchParams.set(key, REDACTED_VALUE)
      }
    }
    return url.toString()
  } catch {
    return value
  }
}

export const sanitizeContextValue = (
  value: unknown,
  options: SanitizeOptions = {},
  depth = 0,
  seen = new WeakSet<object>()
): unknown => {
  const maxStringLength = options.maxStringLength ?? MAX_STRING_LENGTH
  const maxArrayItems = options.maxArrayItems ?? MAX_ARRAY_ITEMS
  const maxDepth = options.maxDepth ?? MAX_DEPTH

  if (value === null || typeof value === 'boolean' || typeof value === 'number') {
    return value
  }

  if (typeof value === 'string') {
    return limitString(sanitizeUrl(value), maxStringLength)
  }

  if (typeof value === 'bigint') return value.toString()
  if (value instanceof Date) return value.toISOString()
  if (typeof value !== 'object') return undefined
  if (depth >= maxDepth) return '[MAX_DEPTH]'
  if (seen.has(value)) return CIRCULAR_VALUE

  seen.add(value)

  if (Array.isArray(value)) {
    const result = value
      .slice(0, maxArrayItems)
      .map((item) => sanitizeContextValue(item, options, depth + 1, seen))
      .filter((item) => item !== undefined)
    seen.delete(value)
    return result
  }

  const record = value as Record<string, unknown>
  const sensitiveDescriptor = hasSensitiveDescriptor(record)
  const result: Record<string, unknown> = {}
  for (const key of Object.keys(value).sort()) {
    if (
      isSensitiveKey(key) ||
      (sensitiveDescriptor && SEMANTIC_VALUE_KEYS.has(normalizeKey(key)))
    ) {
      result[key] = REDACTED_VALUE
      continue
    }

    const sanitized = sanitizeContextValue(record[key], options, depth + 1, seen)
    if (sanitized !== undefined) result[key] = sanitized
  }

  seen.delete(value)
  return result
}

export const sanitizeContextRecord = <T extends Record<string, unknown>>(value: T): T => {
  return sanitizeContextValue(value) as T
}
