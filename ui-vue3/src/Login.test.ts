/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Login from './Login.vue'

const { replace, loadAuthConfiguration, syncAuthenticatedPrincipal } = vi.hoisted(() => ({
  replace: vi.fn(),
  loadAuthConfiguration: vi.fn(),
  syncAuthenticatedPrincipal: vi.fn()
}))

vi.mock('vue-router', async (importOriginal) => {
  const original = await importOriginal<typeof import('vue-router')>()
  return {
    ...original,
    useRouter: () => ({ replace }),
    useRoute: () => ({ query: {} })
  }
})

vi.mock('@/auth/session', async (importOriginal) => {
  const original = await importOriginal<typeof import('@/auth/session')>()
  return { ...original, loadAuthConfiguration, syncAuthenticatedPrincipal }
})

vi.mock('@/api/service/globalSearch', () => ({ meshesSearch: vi.fn() }))

describe('Login', () => {
  const stubs = {
    'a-card': { template: '<div><slot /></div>' },
    'a-row': { template: '<div><slot /></div>' },
    'a-form': { template: '<form><slot /></form>' },
    'a-form-item': { template: '<label><slot /></label>' },
    'a-input': { template: '<input />' },
    'a-button': { template: '<button><slot /></button>' }
  }

  beforeEach(() => {
    vi.clearAllMocks()
    syncAuthenticatedPrincipal.mockRejectedValue(new Error('no session'))
  })

  it('renders password and configured Provider login methods', async () => {
    loadAuthConfiguration.mockResolvedValue({
      methods: ['password'],
      providers: [
        {
          id: 'github',
          displayName: 'GitHub',
          iconUrl: '/admin/auth-providers/github.svg'
        }
      ]
    })
    const wrapper = mount(Login, {
      global: {
        plugins: [createPinia()],
        mocks: { $t: (key: string) => key },
        stubs
      }
    })
    await flushPromises()
    expect(wrapper.find('.password-form').exists()).toBe(true)
    expect(wrapper.text()).toContain('GitHub')
    expect(wrapper.get('.provider-icon').attributes('src')).toBe('/admin/auth-providers/github.svg')
  })

  it('hides password form when the method is disabled', async () => {
    loadAuthConfiguration.mockResolvedValue({
      methods: [],
      providers: [{ id: 'sso', displayName: 'Company SSO' }]
    })
    const wrapper = mount(Login, {
      global: {
        plugins: [createPinia()],
        mocks: { $t: (key: string) => key },
        stubs
      }
    })
    await flushPromises()
    expect(wrapper.find('.password-form').exists()).toBe(false)
    expect(wrapper.text()).toContain('Company SSO')
  })

  it('treats null methods and providers as empty lists', async () => {
    loadAuthConfiguration.mockResolvedValue({
      methods: null,
      providers: null
    })
    const wrapper = mount(Login, {
      global: {
        plugins: [createPinia()],
        mocks: { $t: (key: string) => key },
        stubs
      }
    })
    await flushPromises()
    expect(wrapper.find('.password-form').exists()).toBe(false)
    expect(wrapper.find('.provider-list').exists()).toBe(false)
  })

  it('synchronizes OAuth callback identity through userinfo before redirecting', async () => {
    loadAuthConfiguration.mockResolvedValue({
      methods: [],
      providers: [{ id: 'sso', displayName: 'Company SSO' }]
    })
    syncAuthenticatedPrincipal.mockResolvedValue({ username: 'alice' })
    mount(Login, {
      global: {
        plugins: [createPinia()],
        mocks: { $t: (key: string) => key },
        stubs
      }
    })
    await flushPromises()
    expect(syncAuthenticatedPrincipal).toHaveBeenCalledTimes(1)
    expect(replace).toHaveBeenCalledWith('/')
  })
})
