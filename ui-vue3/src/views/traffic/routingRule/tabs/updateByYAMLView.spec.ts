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
import { HTTP_STATUS } from '@/base/http/constants'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type UpdateByYAMLViewType from './updateByYAMLView.vue'

const mocks = vi.hoisted(() => ({
  getConditionRuleDetailAPI: vi.fn(),
  updateConditionRuleAPI: vi.fn()
}))

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

vi.mock('vue-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-router')>()
  return {
    ...actual,
    useRoute: () => ({ params: { ruleName: 'demo-rule' } })
  }
})

vi.mock('@/api/service/traffic', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/service/traffic')>()
  return {
    ...actual,
    getConditionRuleDetailAPI: mocks.getConditionRuleDetailAPI,
    updateConditionRuleAPI: mocks.updateConditionRuleAPI
  }
})

vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: defineComponent({
    props: {
      modelValue: { type: String, default: '' }
    },
    setup(props) {
      return () => h('pre', { class: 'yaml-value' }, props.modelValue)
    }
  })
}))

vi.mock('ant-design-vue', async (importOriginal) => {
  const actual = await importOriginal<typeof import('ant-design-vue')>()
  return {
    ...actual,
    message: {
      success: vi.fn(),
      error: vi.fn()
    }
  }
})

const passthrough = defineComponent({
  setup(_props, { slots }) {
    return () => h('div', slots.default?.())
  }
})

const buttonStub = defineComponent({
  emits: ['click'],
  setup(_props, { emit, slots }) {
    return () => h('button', { type: 'button', onClick: () => emit('click') }, slots.default?.())
  }
})

let UpdateByYAMLView: typeof UpdateByYAMLViewType

beforeAll(async () => {
  UpdateByYAMLView = (await import('./updateByYAMLView.vue')).default
})

beforeEach(() => {
  mocks.getConditionRuleDetailAPI.mockReset()
  mocks.updateConditionRuleAPI.mockReset()
})

describe('condition route update YAML', () => {
  it('reloads a complete v3.1 rule when the shared tab state is incomplete', async () => {
    const rule = {
      configVersion: 'v3.1',
      priority: 7,
      enabled: true,
      force: false,
      runtime: true,
      key: 'org.apache.dubbo.quickstart.Greeter:1.0.0:demo',
      scope: 'service',
      conditions: [
        {
          from: { match: 'method=SayHello' },
          to: [{ match: 'application=quickstart-provider', weight: 100 }]
        }
      ]
    }
    mocks.getConditionRuleDetailAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS, data: rule })
    mocks.updateConditionRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const tabState = {
      conditionRule: {
        enabled: true,
        key: rule.key,
        runtime: true,
        scope: 'service'
      }
    }

    const wrapper = mount(UpdateByYAMLView, {
      global: {
        provide: {
          [PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE]: tabState
        },
        stubs: {
          AFlex: passthrough,
          'a-flex': passthrough,
          ACol: passthrough,
          'a-col': passthrough,
          ACard: passthrough,
          'a-card': passthrough,
          ASpace: passthrough,
          'a-space': passthrough,
          ADescriptions: passthrough,
          'a-descriptions': passthrough,
          ADescriptionsItem: passthrough,
          'a-descriptions-item': passthrough,
          AAffix: passthrough,
          'a-affix': passthrough,
          AButton: buttonStub,
          'a-button': buttonStub,
          DoubleLeftOutlined: passthrough,
          DoubleRightOutlined: passthrough
        }
      }
    })
    await flushPromises()

    expect(mocks.getConditionRuleDetailAPI).toHaveBeenCalledWith('demo-rule')
    expect(wrapper.find('.yaml-value').text()).toContain('configVersion: v3.1')
    expect(wrapper.find('.yaml-value').text()).toContain('weight: 100')
    expect(tabState.conditionRule).toEqual(rule)

    const submitButton = wrapper.findAll('button').find((button) => button.text().includes('确认'))
    expect(submitButton).toBeDefined()
    await submitButton!.trigger('click')
    await flushPromises()

    expect(mocks.updateConditionRuleAPI).toHaveBeenCalledWith('demo-rule', rule)
  })
})
