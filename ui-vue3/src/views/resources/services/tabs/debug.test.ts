/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import Debug from './debug.vue'

const api = vi.hoisted(() => ({
  getServiceProviderInstancesAPI: vi.fn(),
  getServiceMethodsAPI: vi.fn(),
  getServiceMethodDetailAPI: vi.fn(),
  serviceGenericInvokeAPI: vi.fn()
}))
vi.mock('@/api/service/service', () => api)
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { pathId: 'Echo' } }) }))
vi.mock('@/stores/mesh', () => ({ useMeshStore: () => ({ mesh: 'mesh' }) }))
vi.mock('ant-design-vue', () => ({ message: { error: vi.fn(), warning: vi.fn() } }))
vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: { props: ['modelValue'], template: '<pre>{{ modelValue }}</pre>' }
}))

const slot = { template: '<div><slot /></div>' }
const stubs = {
  ...Object.fromEntries(
    [
      'a-card',
      'a-typography-text',
      'a-spin',
      'a-tabs',
      'a-tab-pane',
      'a-row',
      'a-col',
      'a-tag',
      'a-space',
      'a-modal'
    ].map((name) => [name, slot])
  ),
  'a-button': { template: '<button><slot /></button>' },
  'a-select': {
    props: ['value', 'options'],
    emits: ['update:value'],
    template: `<select :value="value" @change="$emit('update:value', $event.target.value)">
      <option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option>
    </select>`
  },
  'a-input-number': true,
  'a-input': true,
  'a-tree': true,
  'a-empty': true,
  ClockCircleOutlined: true,
  EditOutlined: true,
  MinusCircleOutlined: true,
  PlusOutlined: true
}

async function renderDebug() {
  const wrapper = mount(Debug, { global: { stubs } })
  await flushPromises()
  return wrapper
}

describe('service debug invocation mode', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getServiceProviderInstancesAPI.mockResolvedValue({
      data: [
        {
          name: 'provider',
          generic: 'gson',
          serialization: 'hessian2',
          preferSerialization: 'fastjson2'
        }
      ]
    })
    api.getServiceMethodsAPI.mockResolvedValue({
      data: [
        {
          methodName: 'echo',
          signature: 'echo(java.lang.String)',
          parameterTypes: ['java.lang.String']
        }
      ]
    })
    api.getServiceMethodDetailAPI.mockResolvedValue({
      data: {
        methodName: 'echo',
        signature: 'echo(java.lang.String)',
        parameterTypes: ['java.lang.String'],
        parameters: [],
        returnType: 'java.lang.String',
        types: []
      }
    })
  })

  it('submits gson and retains the request mode when selection changes during the call', async () => {
    let complete!: (value: any) => void
    api.serviceGenericInvokeAPI.mockReturnValueOnce(
      new Promise((resolve) => {
        complete = resolve
      })
    )
    const wrapper = await renderDebug()
    expect(wrapper.text()).toContain('provider 支持情况未确认')
    expect(wrapper.text()).toContain('serialization: hessian2')
    expect(wrapper.text()).toContain('generic: gson')
    const mode = wrapper.get('select[aria-label="调用模式"]')
    await mode.setValue('gson')
    await wrapper.get('.invoke-btn').trigger('click')
    expect(api.serviceGenericInvokeAPI).toHaveBeenCalledWith(
      expect.objectContaining({
        generic: 'gson',
        signature: 'echo(java.lang.String)',
        instanceName: 'provider'
      })
    )
    await mode.setValue('false')
    complete({
      data: {
        generic: 'gson',
        protocol: 'dubbo',
        serialization: 'fastjson2',
        rawResult: 'ok',
        elapsedMs: 5
      }
    })
    await flushPromises()
    expect(wrapper.get('.invocation-info').text()).toContain('generic=gson')
    expect(wrapper.get('.invocation-info').text()).toContain('serialization=fastjson2')
    wrapper.unmount()
  })

  it('labels failed non-generic calls and keeps their attempted transport', async () => {
    api.serviceGenericInvokeAPI.mockRejectedValueOnce({
      code: 'InternalError',
      data: {
        generic: 'false',
        protocol: 'tri',
        serialization: 'hessian2',
        elapsedMs: 12
      }
    })
    const wrapper = await renderDebug()
    await wrapper.get('select[aria-label="调用模式"]').setValue('false')
    await wrapper.get('.invoke-btn').trigger('click')
    await flushPromises()
    expect(api.serviceGenericInvokeAPI).toHaveBeenCalledWith(
      expect.objectContaining({ generic: 'false' })
    )
    expect(wrapper.get('.invocation-info').text()).toContain('普通调用')
    expect(wrapper.get('.invocation-info').text()).toContain('protocol=tri')
    expect(wrapper.get('.invocation-info').text()).toContain('serialization=hessian2')
    wrapper.unmount()
  })
})
