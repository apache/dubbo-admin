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
import type UpdateByFormViewType from './updateByFormView.vue'

const mocks = vi.hoisted(() => ({
  updateConditionRuleAPI: vi.fn(),
  getConditionRuleDetailAPI: vi.fn()
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
    updateConditionRuleAPI: mocks.updateConditionRuleAPI,
    getConditionRuleDetailAPI: mocks.getConditionRuleDetailAPI
  }
})

vi.mock('ant-design-vue', async (importOriginal) => {
  const actual = await importOriginal<typeof import('ant-design-vue')>()
  return {
    ...actual,
    message: {
      success: vi.fn(),
      warning: vi.fn(),
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

let i18n: typeof import('@/base/i18n').i18n
let UpdateByFormView: typeof UpdateByFormViewType

beforeAll(async () => {
  i18n = (await import('@/base/i18n')).i18n
  UpdateByFormView = (await import('./updateByFormView.vue')).default
})

beforeEach(() => {
  mocks.updateConditionRuleAPI.mockReset()
  mocks.getConditionRuleDetailAPI.mockReset()
})

describe('condition route form compatibility', () => {
  it('preserves priority force and configVersion during edit round-trip', async () => {
    mocks.updateConditionRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    mocks.getConditionRuleDetailAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS, data: {} })
    const tabState = {
      conditionRule: {
        configVersion: 'v3.1',
        priority: 12,
        enabled: true,
        force: true,
        key: 'org.apache.demo.DemoService:1.0.0:demo',
        scope: 'service',
        runtime: false,
        conditions: [
          {
            from: { match: 'method=SayHello' },
            to: [{ match: 'region=hangzhou', weight: 100 }]
          }
        ]
      }
    }

    const wrapper = mount(UpdateByFormView, {
      global: {
        plugins: [i18n],
        provide: {
          [PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE]: tabState
        },
        stubs: {
          RoutingRuleList: passthrough,
          AFlex: passthrough,
          'a-flex': passthrough,
          ACol: passthrough,
          'a-col': passthrough,
          ACard: passthrough,
          'a-card': passthrough,
          ASpace: passthrough,
          'a-space': passthrough,
          ARow: passthrough,
          'a-row': passthrough,
          AForm: passthrough,
          'a-form': passthrough,
          AFormItem: passthrough,
          'a-form-item': passthrough,
          ADescriptions: passthrough,
          'a-descriptions': passthrough,
          ADescriptionsItem: passthrough,
          'a-descriptions-item': passthrough,
          ASelect: passthrough,
          'a-select': passthrough,
          AInput: passthrough,
          'a-input': passthrough,
          ASwitch: passthrough,
          'a-switch': passthrough,
          AInputNumber: passthrough,
          'a-input-number': passthrough,
          AButton: buttonStub,
          'a-button': buttonStub,
          DoubleLeftOutlined: passthrough,
          DoubleRightOutlined: passthrough
        }
      }
    })
    await flushPromises()

    expect(tabState.conditionRule).toEqual(
      expect.objectContaining({
        configVersion: 'v3.1',
        priority: 12,
        force: true,
        runtime: false,
        conditions: [
          {
            from: { match: 'method=SayHello' },
            to: [{ match: 'region=hangzhou', weight: 100 }]
          }
        ]
      })
    )

    const submitButton = wrapper.findAll('button').find((button) => button.text().includes('确认'))
    expect(submitButton).toBeDefined()
    await submitButton!.trigger('click')
    await flushPromises()

    expect(mocks.updateConditionRuleAPI).toHaveBeenCalledWith(
      'demo-rule',
      expect.objectContaining({
        configVersion: 'v3.1',
        priority: 12,
        force: true,
        runtime: false,
        conditions: [
          {
            from: { match: 'method=SayHello' },
            to: [{ match: 'region=hangzhou', weight: 100 }]
          }
        ]
      })
    )
  })

  it('reloads details instead of publishing an incomplete shared draft', async () => {
    const rule = {
      configVersion: 'v3.1',
      priority: 5,
      enabled: true,
      force: false,
      key: 'demo-service',
      scope: 'service',
      runtime: true,
      conditions: [{ from: { match: 'method=SayHello' }, to: [] }]
    }
    mocks.getConditionRuleDetailAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS, data: rule })
    const tabState = {
      conditionRule: {
        enabled: true,
        key: 'demo-service',
        runtime: true,
        scope: 'service'
      }
    }

    mount(UpdateByFormView, {
      global: {
        plugins: [i18n],
        provide: {
          [PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE]: tabState
        },
        stubs: {
          RoutingRuleList: passthrough,
          AFlex: passthrough,
          'a-flex': passthrough,
          ACol: passthrough,
          'a-col': passthrough,
          ACard: passthrough,
          'a-card': passthrough,
          ASpace: passthrough,
          'a-space': passthrough,
          ARow: passthrough,
          'a-row': passthrough,
          AForm: passthrough,
          'a-form': passthrough,
          AFormItem: passthrough,
          'a-form-item': passthrough,
          ADescriptions: passthrough,
          'a-descriptions': passthrough,
          ADescriptionsItem: passthrough,
          'a-descriptions-item': passthrough,
          ASelect: passthrough,
          'a-select': passthrough,
          AInput: passthrough,
          'a-input': passthrough,
          ASwitch: passthrough,
          'a-switch': passthrough,
          AInputNumber: passthrough,
          'a-input-number': passthrough,
          AButton: buttonStub,
          'a-button': buttonStub,
          DoubleLeftOutlined: passthrough,
          DoubleRightOutlined: passthrough
        }
      }
    })
    await flushPromises()

    expect(mocks.getConditionRuleDetailAPI).toHaveBeenCalledWith('demo-rule')
    expect(tabState.conditionRule).toEqual(rule)
  })
})
