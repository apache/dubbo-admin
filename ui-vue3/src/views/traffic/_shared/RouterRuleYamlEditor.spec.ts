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
import { beforeAll, beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent, h } from 'vue'
import yaml from 'js-yaml'
import { HTTP_STATUS } from '@/base/http/constants'
import type RouterRuleYamlEditorType from './RouterRuleYamlEditor.vue'

const mocks = vi.hoisted(() => ({
  addRouterRuleAPI: vi.fn(),
  getRouterRuleAPI: vi.fn(),
  updateRouterRuleAPI: vi.fn(),
  push: vi.fn(),
  route: { params: {} as Record<string, string> }
}))

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => mocks.route,
    useRouter: () => ({ push: mocks.push }),
    onBeforeRouteLeave: vi.fn()
  }
})

vi.mock('@/api/service/traffic', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/service/traffic')>()
  return {
    ...actual,
    addRouterRuleAPI: mocks.addRouterRuleAPI,
    getRouterRuleAPI: mocks.getRouterRuleAPI,
    updateRouterRuleAPI: mocks.updateRouterRuleAPI
  }
})

vi.mock('ant-design-vue', async (importOriginal) => {
  const actual = await importOriginal<typeof import('ant-design-vue')>()
  return {
    ...actual,
    message: {
      error: vi.fn(),
      success: vi.fn()
    },
    Modal: { confirm: vi.fn() }
  }
})

vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: {
    name: 'MonacoEditor',
    template: '<textarea />'
  }
}))

const passthrough = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.())
  }
})

const buttonStub = defineComponent({
  props: {
    disabled: Boolean,
    loading: Boolean
  },
  emits: ['click'],
  setup(props, { emit, slots }) {
    return () =>
      h(
        'button',
        { disabled: props.disabled || props.loading, type: 'button', onClick: () => emit('click') },
        slots.default?.()
      )
  }
})

const monacoStub = defineComponent({
  props: { modelValue: { type: String, default: '' } },
  emits: ['update:modelValue'],
  setup(props, { emit }) {
    return () =>
      h('textarea', {
        'data-test': 'yaml-editor',
        value: props.modelValue,
        onInput: (event: Event) =>
          emit('update:modelValue', (event.target as HTMLTextAreaElement).value)
      })
  }
})

let i18n: typeof import('@/base/i18n').i18n
let RouterRuleYamlEditor: typeof RouterRuleYamlEditorType

beforeAll(async () => {
  i18n = (await import('@/base/i18n')).i18n
  RouterRuleYamlEditor = (await import('./RouterRuleYamlEditor.vue')).default
})

beforeEach(() => {
  mocks.addRouterRuleAPI.mockReset()
  mocks.getRouterRuleAPI.mockReset()
  mocks.updateRouterRuleAPI.mockReset()
  mocks.push.mockReset()
  mocks.route.params = {}
})

const mountEditor = (kind: 'affinity-rule' | 'script-rule') =>
  mount(RouterRuleYamlEditor, {
    props: { kind },
    global: {
      plugins: [i18n],
      stubs: {
        ACard: passthrough,
        'a-card': passthrough,
        AAlert: passthrough,
        'a-alert': passthrough,
        ASpin: passthrough,
        'a-spin': passthrough,
        ASpace: passthrough,
        'a-space': passthrough,
        AButton: buttonStub,
        'a-button': buttonStub,
        MonacoEditor: monacoStub
      }
    }
  })

describe('RouterRuleYamlEditor', () => {
  it('creates affinity rules with the key-derived affinity suffix', async () => {
    mocks.addRouterRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const wrapper = mountEditor('affinity-rule')
    await flushPromises()

    await wrapper.findAll('button')[0].trigger('click')
    await flushPromises()

    expect(mocks.addRouterRuleAPI).toHaveBeenCalledWith(
      'affinity-rule',
      'demo-provider.affinity-router',
      expect.objectContaining({
        configVersion: 'v3.1',
        key: 'demo-provider',
        affinityAware: { key: 'region', ratio: 80 }
      })
    )
    expect(mocks.push).toHaveBeenCalledWith('/traffic/affinityRule')
  })

  it('creates script rules with the consumer-supported JavaScript template', async () => {
    mocks.addRouterRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const wrapper = mountEditor('script-rule')
    await flushPromises()

    await wrapper.findAll('button')[0].trigger('click')
    await flushPromises()

    const [, ruleName, payload] = mocks.addRouterRuleAPI.mock.calls[0]
    expect(ruleName).toBe('demo-provider.script-router')
    expect(payload).toMatchObject({
      configVersion: 'v3.0',
      key: 'demo-provider',
      scope: 'application',
      type: 'javascript'
    })
    expect((payload as { script: string }).script).toContain('(invokers, invocation, context)')
    expect(
      yaml.load((wrapper.find('[data-test="yaml-editor"]').element as HTMLTextAreaElement).value)
    ).toMatchObject(payload)
    expect(mocks.push).toHaveBeenCalledWith('/traffic/scriptRule')
  })
})
