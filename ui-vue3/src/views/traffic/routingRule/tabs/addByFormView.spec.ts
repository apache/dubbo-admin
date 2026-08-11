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
import type AddByFormViewType from './addByFormView.vue'

const mocks = vi.hoisted(() => ({
  addConditionRuleAPI: vi.fn(),
  push: vi.fn()
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
    useRouter: () => ({ push: mocks.push })
  }
})

vi.mock('@/api/service/traffic', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/service/traffic')>()
  return {
    ...actual,
    addConditionRuleAPI: mocks.addConditionRuleAPI
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

vi.mock('vue-clipboard3', () => ({
  default: () => ({ toClipboard: vi.fn() })
}))

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
let AddByFormView: typeof AddByFormViewType

beforeAll(async () => {
  i18n = (await import('@/base/i18n')).i18n
  AddByFormView = (await import('./addByFormView.vue')).default
})

beforeEach(() => {
  mocks.addConditionRuleAPI.mockReset()
  mocks.push.mockReset()
})

function mountForm(tabState: any) {
  return mount(AddByFormView, {
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
}

describe('condition route add form', () => {
  it('uses the service condition-router rule name expected by dubbo-go', async () => {
    mocks.addConditionRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const tabState = {
      conditionRule: {
        enabled: true,
        key: 'org.apache.demo.DemoService',
        scope: 'service',
        runtime: true,
        conditions: ['method=sayHello => region=hangzhou']
      },
      addConditionRuleSate: {
        version: '1.0.0',
        group: 'demo'
      }
    }

    const wrapper = mountForm(tabState)
    await flushPromises()

    const submitButton = wrapper.findAll('button')[1]
    await submitButton.trigger('click')
    await flushPromises()

    expect(mocks.addConditionRuleAPI).toHaveBeenCalledWith(
      'org.apache.demo.DemoService:1.0.0:demo.condition-router',
      expect.objectContaining({
        configVersion: 'v3.0',
        scope: 'service',
        key: 'org.apache.demo.DemoService',
        conditions: ['method=sayHello => region=hangzhou']
      })
    )
  })

  it('uses the application condition-router rule name expected by dubbo-go', async () => {
    mocks.addConditionRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const tabState = {
      conditionRule: {
        enabled: true,
        key: 'demo-provider',
        scope: 'application',
        runtime: true,
        conditions: ['host=1.1.1.1 => host=2.2.2.2']
      }
    }

    const wrapper = mountForm(tabState)
    await flushPromises()

    const submitButton = wrapper.findAll('button')[1]
    await submitButton.trigger('click')
    await flushPromises()

    expect(mocks.addConditionRuleAPI).toHaveBeenCalledWith(
      'demo-provider.condition-router',
      expect.objectContaining({
        scope: 'application',
        key: 'demo-provider',
        conditions: ['host=1.1.1.1 => host=2.2.2.2']
      })
    )
  })
})
