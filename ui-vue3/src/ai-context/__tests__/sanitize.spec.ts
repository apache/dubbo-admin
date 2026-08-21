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

import { describe, expect, it } from 'vitest'
import { sanitizeContextValue } from '../sanitize'

describe('sanitizeContextValue', () => {
  it('redacts sensitive fields and URL credentials', () => {
    const result = sanitizeContextValue({
      password: 'plain-text',
      api_key: 'key-value',
      registry: 'nacos://admin:secret@127.0.0.1:8848?username=nacos&password=nacos',
      application: 'shop-user'
    }) as Record<string, string>

    expect(result.password).toBe('[REDACTED]')
    expect(result.api_key).toBe('[REDACTED]')
    expect(result.application).toBe('shop-user')
    expect(result.registry).not.toContain('admin:secret')
    expect(result.registry).not.toContain('password=nacos')
    expect(result.registry).not.toContain('username=nacos')
  })

  it('redacts values described by semantic sensitive keys', () => {
    const result = sanitizeContextValue({
      properties: [
        { key: 'access-token', value: 'token-value' },
        { name: 'DB_PASSWORD', currentValue: 'password-value', defaultValue: 'default-value' },
        { key: 'environment', value: 'production' }
      ]
    }) as Record<string, any>

    expect(result.properties).toEqual([
      { key: 'access-token', value: '[REDACTED]' },
      {
        currentValue: '[REDACTED]',
        defaultValue: '[REDACTED]',
        name: 'DB_PASSWORD'
      },
      { key: 'environment', value: 'production' }
    ])
  })

  it('limits strings, arrays, depth, and circular values', () => {
    const circular: Record<string, unknown> = {}
    circular.self = circular

    const result = sanitizeContextValue(
      {
        text: 'abcdefghij',
        items: [1, 2, 3, 4],
        nested: { value: { deeper: { ignored: true } } },
        circular
      },
      {
        maxStringLength: 5,
        maxArrayItems: 2,
        maxDepth: 3
      }
    ) as Record<string, any>

    expect(result.text).toBe('abcde...[TRUNCATED]')
    expect(result.items).toEqual([1, 2])
    expect(result.nested.value.deeper).toBe('[MAX_DEPTH]')
    expect(result.circular.self).toBe('[CIRCULAR]')
  })
})
