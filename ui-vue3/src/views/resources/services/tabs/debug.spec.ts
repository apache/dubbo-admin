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

import { flushPromises, mount } from '@vue/test-utils'
import { expect, it, vi } from 'vitest'
import Debug from './debug.vue'
import { serviceGenericInvokeAPI } from '@/api/service/service'

const { request } = vi.hoisted(() => ({ request: vi.fn() }))
vi.mock('@/base/http/request', () => ({ default: request }))
vi.mock('@/base/constants', () => ({ PRIMARY_COLOR: '#fff', PRIMARY_COLOR_R: '#000' }))
vi.mock('vue-router', () => ({ useRoute: () => ({ params: { pathId: 'Service' } }) }))
vi.mock('@/stores/mesh', () => ({ useMeshStore: () => ({ mesh: 'mesh' }) }))
vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: {
    props: ['editorId', 'modelValue'],
    template:
      '<textarea :id="editorId" :value="modelValue" @input="$emit(\'update:modelValue\', $event.target.value)" />'
  }
}))

it('retains quoted and numeric tokens and rejects invalid editor JSON before posting', async () => {
  request.mockClear()
  const envelope = {
    mesh: 'mesh',
    instanceName: 'instance',
    serviceName: 'Service',
    methodName: 'echo',
    signature: 'long->void'
  }
  for (const args of [
    '["9223372036854775807"]',
    '[9223372036854775807]',
    '{"id":9223372036854775806.9}'
  ]) {
    await serviceGenericInvokeAPI({ ...envelope, args })
    const { data, headers } = request.mock.lastCall![0]
    expect(data).toContain('"args":' + (args.startsWith('[') ? args : `[${args}]`))
    expect(headers['Content-Type']).toBe('application/json')
  }
  expect(() => serviceGenericInvokeAPI({ ...envelope, args: '[1], "args": []' })).toThrow(
    SyntaxError
  )
  expect(request).toHaveBeenCalledTimes(3)
})

it('binds the template, submission and field errors to the active overload', async () => {
  const methods = ['long', 'User'].map((type) => ({
    methodName: 'echo',
    parameterTypes: [type],
    signature: type + '->void'
  }))
  const types = [
    { type: 'User', properties: { profile: 'Profile', role: 'Role', ids: 'map[string]int64' } },
    { type: 'Profile', properties: { id: 'long' } },
    { type: 'Role', enums: ['ADMIN', 'USER'] }
  ]
  let resolveFirst!: (res: unknown) => void
  const delayed = new Promise((resolve) => {
    resolveFirst = resolve
  })
  const reason = 'user.profile.id: expected long, got missing (required field)'
  request.mockImplementation(({ url, params }) => {
    if (url.endsWith('/methods')) return { data: methods }
    if (url.endsWith('/provider-instances')) return { data: [{ name: 'instance', appName: 'app' }] }
    if (url.endsWith('/detail')) {
      if (params.signature === methods[0].signature) return delayed
      return { data: { ...methods[1], types } }
    }
    return Promise.reject({ code: 'InvalidArgument', message: reason })
  })
  const pass = { template: '<div><slot /></div>' }
  const tabs = { ...pass, emits: ['update:activeKey'] }
  const wrapper = mount(Debug, {
    global: {
      mocks: { $t: (key: string) => key },
      stubs: {
        ACard: pass,
        ASpin: pass,
        ARow: pass,
        ACol: pass,
        ATypographyText: pass,
        ATabs: tabs,
        ATabPane: {
          props: ['tab'],
          template: '<section><label>{{ tab }}</label><slot /></section>'
        },
        AButton: { template: '<button><slot /></button>' },
        ASelect: true,
        ATag: true,
        ATree: true,
        AModal: true,
        AInputNumber: true,
        AEmpty: true,
        AInput: true
      }
    }
  })
  await flushPromises()
  expect(wrapper.findAll('label').map((label) => label.text())).toEqual([
    'echo(long)',
    'echo(User)'
  ])
  wrapper.findComponent(tabs).vm.$emit('update:activeKey', '1')
  await flushPromises()
  resolveFirst({ data: { ...methods[0], types: [] } })
  await flushPromises()
  const editor = wrapper.get<HTMLTextAreaElement>('#requestEditor-1')
  expect(JSON.parse(editor.element.value)).toEqual([
    { profile: { id: '0' }, role: 'ADMIN', ids: {} }
  ])
  expect(request).toHaveBeenCalledWith(
    expect.objectContaining({
      params: expect.objectContaining({ providerAppName: 'app', signature: methods[1].signature })
    })
  )
  const args = '[{"profile":{},"role":"ADMIN","ids":{}}]'
  await editor.setValue(args)
  await wrapper.findAll('.invoke-btn')[1].trigger('click')
  await flushPromises()
  expect(JSON.parse(request.mock.lastCall![0].data)).toMatchObject({
    signature: 'User->void',
    args: JSON.parse(args)
  })
  expect(wrapper.get<HTMLTextAreaElement>('#responseEditor-1').element.value).toContain(reason)
  wrapper.unmount()
})
