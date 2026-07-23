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

import { defineComponent, h, reactive } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import AppTabHeaderSlot from '@/views/resources/applications/slots/AppTabHeaderSlot.vue'
import AddConditionRuleTabHeaderSlot from '@/views/traffic/routingRule/slots/addConditionRuleTabHeaderSlot.vue'
import AgentDrawer from '@/components/AgentDrawer.vue'
import { AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID } from '../selection'
import type { AIContextProvider, AIContextSnapshot } from '../types'

const mocks = vi.hoisted(() => ({
  register: vi.fn(),
  snapshot: vi.fn(),
  createSession: vi.fn(),
  sendChatMessage: vi.fn(),
  getSessions: vi.fn(),
  getSessionInfo: vi.fn(),
  deleteSession: vi.fn()
}))

vi.mock('../instance', () => ({
  aiContextManager: {
    register: mocks.register,
    snapshot: mocks.snapshot
  }
}))

vi.mock('@/api/service/ai', () => ({
  aiService: {
    createSession: mocks.createSession,
    sendChatMessage: mocks.sendChatMessage,
    getSessions: mocks.getSessions,
    getSessionInfo: mocks.getSessionInfo,
    deleteSession: mocks.deleteSession
  }
}))

vi.mock('@/components/ai-chat/MessageList.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name: 'MessageList',
      setup(_, { expose }) {
        expose({ scrollToBottom: vi.fn() })
        return () => h('div')
      }
    })
  }
})

vi.mock('@/components/ai-chat/ChatInput.vue', async () => {
  const { defineComponent, h, ref } = await import('vue')
  return {
    default: defineComponent({
      name: 'ChatInput',
      emits: ['sendMessage'],
      setup(_, { emit, expose }) {
        const inputMessage = ref('')
        expose({
          inputMessage,
          setInputMessage: (value: string) => {
            inputMessage.value = value
          }
        })
        return () => h('button', { onClick: () => emit('sendMessage') }, 'send')
      }
    })
  }
})

vi.mock('@/components/ai-chat/AIContextPreview.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name: 'AIContextPreview',
      props: ['snapshot', 'enabled', 'excludedSectionIds'],
      emits: ['update:enabled', 'update:excludedSectionIds', 'refresh'],
      setup() {
        return () => h('div')
      }
    })
  }
})

vi.mock('@/components/ai-chat/SessionHistoryModal.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name: 'SessionHistoryModal',
      setup() {
        return () => h('div')
      }
    })
  }
})

const createEmptyStream = () =>
  new ReadableStream({
    start(controller) {
      controller.close()
    }
  })

const routerHost = defineComponent({
  setup: () => () => h(RouterView)
})

const layoutStubs = {
  'a-row': { template: '<div><slot /></div>' },
  'a-col': { template: '<div><slot /></div>' }
}

describe('page AI context integration', () => {
  beforeEach(() => {
    mocks.register.mockReset()
    mocks.register.mockImplementation(() => vi.fn())
  })

  it('unregisters a resource provider when its page is left', async () => {
    const unregister = vi.fn()
    mocks.register.mockReturnValueOnce(unregister)
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/applications/:pathId', component: AppTabHeaderSlot },
        { path: '/home', component: { template: '<div />' } }
      ]
    })
    await router.push('/applications/shop-user')
    await router.isReady()

    const wrapper = mount(routerHost, {
      global: {
        plugins: [router],
        stubs: layoutStubs,
        mocks: { $t: (key: string) => key }
      }
    })
    await flushPromises()

    const provider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(provider.collect()).toEqual({ scope: { application: 'shop-user' } })

    await router.push('/home')
    await flushPromises()

    expect(unregister).toHaveBeenCalledOnce()
    wrapper.unmount()
  })

  it('keeps the shared draft while switching between form and YAML tabs', async () => {
    const tabState = reactive({
      conditionRule: {
        scope: 'service',
        key: 'DemoService',
        enabled: true,
        conditions: ['host=10.0.0.1 => address=10.0.0.2']
      }
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/rules/form',
          name: 'addConditionRuleByFormView',
          component: AddConditionRuleTabHeaderSlot
        },
        {
          path: '/rules/yaml',
          name: 'addConditionRuleByYAMLView',
          component: AddConditionRuleTabHeaderSlot
        }
      ]
    })
    const i18n = createI18n({
      legacy: false,
      locale: 'en',
      messages: { en: { routingRuleDomain: { createNewRoutingRule: 'Create routing rule' } } }
    })
    await router.push('/rules/form')
    await router.isReady()

    const wrapper = mount(routerHost, {
      global: {
        plugins: [router, i18n],
        provide: { [PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE]: tabState },
        stubs: layoutStubs
      }
    })
    await flushPromises()

    const formProvider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(formProvider.collect()?.state?.unsavedChanges).toMatchObject({
      representation: 'form',
      key: 'DemoService',
      entryCount: 1
    })

    tabState.conditionRule.conditions.push('method=sayHello => address=10.0.0.3')
    await router.push('/rules/yaml')
    await flushPromises()

    expect(mocks.register).toHaveBeenCalledOnce()
    expect(formProvider.collect()?.state?.unsavedChanges).toMatchObject({
      representation: 'yaml',
      key: 'DemoService',
      entryCount: 2
    })
    wrapper.unmount()
  })
})

describe('AI context request integration', () => {
  const snapshot: AIContextSnapshot = {
    version: 1,
    capturedAt: '2026-07-23T08:00:00.000Z',
    global: { locale: 'en' },
    page: { path: '/rules/form', fullPath: '/rules/form' },
    scope: { mesh: 'nacos2.5', service: 'DemoService' },
    state: {
      selection: { group: 'prod' },
      unsavedChanges: { kind: 'condition-rule', representation: 'form', entryCount: 1 }
    }
  }

  beforeEach(() => {
    mocks.snapshot.mockReset()
    mocks.snapshot.mockReturnValue(snapshot)
    mocks.createSession.mockReset()
    mocks.createSession.mockResolvedValue('session-1')
    mocks.sendChatMessage.mockReset()
    mocks.sendChatMessage.mockImplementation(async () => createEmptyStream())
  })

  it('excludes the draft from the next request and resets the one-shot selection', async () => {
    const wrapper = mount(AgentDrawer, {
      props: { agentDrawerOpen: true },
      global: {
        stubs: {
          'a-drawer': { template: '<div><slot /></div>' }
        }
      }
    })
    await flushPromises()

    const preview = wrapper.findComponent({ name: 'AIContextPreview' })
    const input = wrapper.findComponent({ name: 'ChatInput' })
    const exposedInput = input.vm.$.exposed as {
      inputMessage: { value: string }
    }
    await preview.vm.$emit('update:excludedSectionIds', [AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID])
    exposedInput.inputMessage.value = 'review this rule'
    await input.vm.$emit('sendMessage')
    await flushPromises()

    const firstContext = mocks.sendChatMessage.mock.calls[0][2] as AIContextSnapshot
    expect(firstContext.state).toEqual({ selection: { group: 'prod' } })

    exposedInput.inputMessage.value = 'review it again'
    await input.vm.$emit('sendMessage')
    await flushPromises()

    const secondContext = mocks.sendChatMessage.mock.calls[1][2] as AIContextSnapshot
    expect(secondContext.state?.unsavedChanges).toEqual(snapshot.state?.unsavedChanges)
    wrapper.unmount()
  })
})
