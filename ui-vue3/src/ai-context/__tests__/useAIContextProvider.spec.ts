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

import { defineComponent, nextTick } from 'vue'
import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { aiContextManager } from '../instance'
import { useAIContextProvider } from '../composables/useAIContextProvider'

vi.mock('../instance', () => ({
  aiContextManager: {
    register: vi.fn()
  }
}))

describe('useAIContextProvider', () => {
  const register = vi.mocked(aiContextManager.register)

  beforeEach(() => {
    register.mockReset()
  })

  it('tracks the active route and unregisters on unmount', async () => {
    const unregister = vi.fn()
    register.mockReturnValue(unregister)

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/one', component: { template: '<div />' } },
        { path: '/two', component: { template: '<div />' } }
      ]
    })
    await router.push('/one')
    await router.isReady()

    const host = defineComponent({
      setup() {
        useAIContextProvider({ id: 'test-page', collect: () => undefined })
        return () => null
      }
    })
    const wrapper = mount(host, { global: { plugins: [router] } })
    const registeredProvider = register.mock.calls[0][0]

    expect(registeredProvider.routeKey?.()).toBe('/one')

    await router.push('/two')
    await nextTick()
    expect(registeredProvider.routeKey?.()).toBe('/two')

    wrapper.unmount()
    expect(unregister).toHaveBeenCalledOnce()
  })
})
