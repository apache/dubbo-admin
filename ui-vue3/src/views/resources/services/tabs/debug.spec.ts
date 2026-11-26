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

import { flushPromises, shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { ServiceMethodDetail } from '@/types/api'
import ServiceDebugTab from './debug.vue'

const api = vi.hoisted(() => ({
  getServiceProviderInstancesAPI: vi.fn(),
  getServiceMethodsAPI: vi.fn(),
  getServiceMethodDetailAPI: vi.fn(),
  serviceGenericInvokeAPI: vi.fn()
}))

vi.mock('@/api/service/service', () => api)
vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { pathId: 'org.apache.dubbo.samples.UserService' } })
}))
vi.mock('@/stores/mesh', () => ({ useMeshStore: () => ({ mesh: 'default' }) }))
vi.mock('ant-design-vue', () => ({
  message: { warning: vi.fn(), error: vi.fn() }
}))
// Keep Monaco's browser runtime out of the test; inspect its actual v-model input.
vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: { name: 'MonacoEditor', template: '<div />' }
}))

const editorStub = {
  props: ['modelValue', 'editorId'],
  template: '<textarea :data-editor="editorId" :value="modelValue" />'
}
const stubs = Object.fromEntries(
  [
    'a-card',
    'a-typography-text',
    'a-spin',
    'a-empty',
    'a-tabs',
    'a-tab-pane',
    'a-row',
    'a-col',
    'a-tree',
    'a-tag',
    'a-select',
    'a-input-number',
    'a-button',
    'a-modal',
    'a-input'
  ].map((name) => [name, true])
)

let wrapper: ReturnType<typeof shallowMount> | undefined

beforeEach(() => {
  vi.resetAllMocks()
  api.getServiceProviderInstancesAPI.mockResolvedValue({
    data: [{ name: 'dubbo-provider-0', appName: 'provider', ip: '127.0.0.1' }]
  })
})

afterEach(() => {
  wrapper?.unmount()
  wrapper = undefined
})

function mountDebug() {
  wrapper = shallowMount(ServiceDebugTab, {
    global: {
      renderStubDefaultSlot: true,
      stubs: { ...stubs, MonacoEditor: editorStub }
    }
  })
  return wrapper
}

async function mountDetail(parameterTypes: string[], types: ServiceMethodDetail['types'] = []) {
  const detail: ServiceMethodDetail = {
    methodName: 'query',
    signature: `${parameterTypes.join(',')}->org.apache.demo.UserDTO`,
    parameterTypes,
    parameters: parameterTypes.map((type, index) => ({ name: `arg${index}`, type })),
    returnType: 'org.apache.demo.UserDTO',
    types
  }
  api.getServiceMethodsAPI.mockResolvedValue({
    data: [{ methodName: detail.methodName, signature: detail.signature, parameterTypes }]
  })
  api.getServiceMethodDetailAPI.mockResolvedValue({ data: detail })
  const mounted = mountDebug()
  await flushPromises()
  expect(api.getServiceMethodDetailAPI).toHaveBeenCalledWith({
    serviceName: 'org.apache.dubbo.samples.UserService',
    methodName: 'query',
    signature: detail.signature,
    group: undefined,
    version: undefined
  })
  return mounted
}

async function requestTemplate(parameterTypes: string[], types: ServiceMethodDetail['types'] = []) {
  const mounted = await mountDetail(parameterTypes, types)
  return JSON.parse(
    mounted.get<HTMLTextAreaElement>('[data-editor="requestEditor-0"]').element.value
  )
}

describe('ServiceDebugTab request templates', () => {
  it.each(['char', 'java.lang.Character'])('generates "a" for %s', async (type) => {
    expect(await requestTemplate([type])).toEqual(['a'])
  })

  it.each(['String', 'java.lang.String'])('keeps the empty string for %s', async (type) => {
    expect(await requestTemplate([type])).toEqual([''])
  })

  it.each(['boolean', 'java.lang.Boolean'])('keeps false for %s', async (type) => {
    expect(await requestTemplate([type])).toEqual([false])
  })

  it('keeps zero for int', async () => {
    expect(await requestTemplate(['int'])).toEqual([0])
  })

  it.each(['char[]', 'java.lang.String[]'])(
    'keeps an empty array for %s metadata',
    async (type) => {
      expect(
        await requestTemplate(
          [type],
          [{ type, properties: {}, items: [type.slice(0, -2)], enums: [] }]
        )
      ).toEqual([[]])
    }
  )

  it('recursively generates custom objects from metadata', async () => {
    expect(
      await requestTemplate(
        ['example.Request'],
        [
          {
            type: 'example.Request',
            properties: { profile: 'example.Profile', enabled: 'java.lang.Boolean' },
            items: [],
            enums: []
          },
          {
            type: 'example.Profile',
            properties: { name: 'java.lang.String', age: 'int' },
            items: [],
            enums: []
          }
        ]
      )
    ).toEqual([{ profile: { name: '', age: 0 }, enabled: false }])
  })

  it('keeps the first enum value from metadata', async () => {
    expect(
      await requestTemplate(
        ['example.Status'],
        [{ type: 'example.Status', properties: {}, items: [], enums: ['ACTIVE', 'INACTIVE'] }]
      )
    ).toEqual(['ACTIVE'])
  })

  it.each(['char', 'java.lang.Character'])(
    'generates a single %s inside nested objects',
    async (type) => {
      expect(
        await requestTemplate(
          ['example.Request'],
          [
            {
              type: 'example.Request',
              properties: { profile: 'example.Profile' },
              items: [],
              enums: []
            },
            { type: 'example.Profile', properties: { initial: type }, items: [], enums: [] }
          ]
        )
      ).toEqual([{ profile: { initial: 'a' } }])
    }
  )

  it('passes the selected signature and generated args to invoke and displays rawResult', async () => {
    const mounted = await mountDetail(['java.lang.String'])
    const result = { id: '1001', name: 'Alice', age: 18 }
    api.serviceGenericInvokeAPI.mockResolvedValue({ data: { elapsedMs: 12, rawResult: result } })

    await mounted.get('.invoke-btn').trigger('click')
    await flushPromises()

    expect(api.serviceGenericInvokeAPI).toHaveBeenCalledWith({
      mesh: 'default',
      instanceName: 'dubbo-provider-0',
      serviceName: 'org.apache.dubbo.samples.UserService',
      methodName: 'query',
      signature: 'java.lang.String->org.apache.demo.UserDTO',
      args: [''],
      group: undefined,
      version: undefined,
      timeoutMs: 3000,
      attachments: undefined
    })
    expect(
      JSON.parse(mounted.get<HTMLTextAreaElement>('[data-editor="responseEditor-0"]').element.value)
    ).toEqual(result)
  })

  it('switches query overloads without mixing detail signatures, templates or invoke args', async () => {
    const details: ServiceMethodDetail[] = ['java.lang.String', 'char'].map((type) => ({
      methodName: 'query',
      signature: `${type}->org.apache.demo.UserDTO`,
      parameterTypes: [type],
      parameters: [{ name: 'arg0', type }],
      returnType: 'org.apache.demo.UserDTO',
      types: []
    }))
    const results = [
      { id: '1001', name: 'Alice', age: 18 },
      { id: '1002', name: 'Bob', age: 20 }
    ]
    api.getServiceMethodsAPI.mockResolvedValue({ data: details })
    api.getServiceMethodDetailAPI.mockImplementation(async ({ signature }) => ({
      data: details.find((detail) => detail.signature === signature)
    }))
    const mounted = mountDebug()
    await flushPromises()

    for (const index of [0, 1, 0]) {
      if (api.serviceGenericInvokeAPI.mock.calls.length > 0) {
        const tabs = mounted.getComponent('a-tabs-stub')
        tabs.vm.$emit('update:activeKey', String(index))
        tabs.vm.$emit('change', String(index))
        await flushPromises()
      }
      const detail = details[index]
      const args = index === 0 ? [''] : ['a']
      expect(api.getServiceMethodDetailAPI).toHaveBeenLastCalledWith({
        serviceName: 'org.apache.dubbo.samples.UserService',
        methodName: 'query',
        signature: detail.signature,
        group: undefined,
        version: undefined
      })
      expect(
        JSON.parse(
          mounted.get<HTMLTextAreaElement>(`[data-editor="requestEditor-${index}"]`).element.value
        )
      ).toEqual(args)
      api.serviceGenericInvokeAPI.mockResolvedValue({
        data: { elapsedMs: 12, rawResult: results[index] }
      })
      await mounted.findAll('.invoke-btn')[index].trigger('click')
      await flushPromises()
      expect(api.serviceGenericInvokeAPI).toHaveBeenLastCalledWith(
        expect.objectContaining({ methodName: 'query', signature: detail.signature, args })
      )
      expect(
        JSON.parse(
          mounted.get<HTMLTextAreaElement>(`[data-editor="responseEditor-${index}"]`).element.value
        )
      ).toEqual(results[index])
    }
    expect(api.getServiceMethodDetailAPI).toHaveBeenCalledTimes(3)
    expect(api.serviceGenericInvokeAPI).toHaveBeenCalledTimes(3)
  })
})
