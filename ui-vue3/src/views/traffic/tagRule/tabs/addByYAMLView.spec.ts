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
import type AddByYAMLViewType from './addByYAMLView.vue'

const mocks = vi.hoisted(() => ({
  addTagRuleAPI: vi.fn(),
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
    addTagRuleAPI: mocks.addTagRuleAPI
  }
})

vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: defineComponent({
    setup(_props, { slots }) {
      return () => h('div', slots.default?.())
    }
  })
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

let AddByYAMLView: typeof AddByYAMLViewType

beforeAll(async () => {
  AddByYAMLView = (await import('./addByYAMLView.vue')).default
})

beforeEach(() => {
  mocks.addTagRuleAPI.mockReset()
  mocks.push.mockReset()
})

describe('tag route add YAML', () => {
  it('uses the application tag-router rule name expected by dubbo-go', async () => {
    mocks.addTagRuleAPI.mockResolvedValue({ code: HTTP_STATUS.SUCCESS })
    const tabState = {
      tagRule: {
        configVersion: 'v3.0',
        enabled: true,
        key: 'demo-provider',
        scope: 'application',
        runtime: true,
        tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] }]
      }
    }

    const wrapper = mount(AddByYAMLView, {
      global: {
        provide: {
          [PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE]: tabState
        },
        stubs: {
          MonacoEditor: passthrough,
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

    const submitButton = wrapper.findAll('button').find((button) => button.text().includes('确认'))
    expect(submitButton).toBeDefined()
    await submitButton!.trigger('click')
    await flushPromises()

    expect(mocks.addTagRuleAPI).toHaveBeenCalledWith(
      'demo-provider.tag-router',
      expect.objectContaining({
        configVersion: 'v3.0',
        scope: 'application',
        key: 'demo-provider',
        tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] }]
      })
    )
  })
})
