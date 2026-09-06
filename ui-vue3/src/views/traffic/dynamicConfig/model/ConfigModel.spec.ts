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

import { describe, expect, it, vi } from 'vitest'
import { ViewDataModel } from './ConfigModel'

vi.hoisted(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: () => null,
      setItem: () => undefined,
      removeItem: () => undefined
    },
    configurable: true
  })
})

describe('ViewDataModel traffic form compatibility', () => {
  it('round-trips dynamic config configVersion and config row fields', () => {
    const model = new ViewDataModel()
    model.fromApiOutput({
      ruleName: 'demo.configurators',
      scope: 'service',
      key: 'demo',
      enabled: true,
      configVersion: 'v3.1',
      configs: [
        {
          enabled: false,
          side: 'consumer',
          match: {
            application: {
              oneof: [{ exact: 'shop-web' }]
            }
          },
          parameters: {
            timeout: '3000',
            retries: '2'
          }
        }
      ]
    })

    const payload = model.toApiInput()

    expect(payload.configVersion).toBe('v3.1')
    expect(payload.enabled).toBe(true)
    expect(payload.configs[0].enabled).toBe(false)
    expect(payload.configs[0].side).toBe('consumer')
    expect(payload.configs[0].parameters).toEqual({ timeout: '3000', retries: '2' })
  })
})
