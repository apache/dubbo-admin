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
import { flushPromises, mount, shallowMount } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import { createMemoryHistory, createRouter, RouterView } from 'vue-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import AppTabHeaderSlot from '@/views/resources/applications/slots/AppTabHeaderSlot.vue'
import InstanceDetailPage from '@/views/resources/instances/tabs/detail.vue'
import ServiceTabHeaderSlot from '@/views/resources/services/slots/ServiceTabHeaderSlot.vue'
import AddConditionRuleTabHeaderSlot from '@/views/traffic/routingRule/slots/addConditionRuleTabHeaderSlot.vue'
import AgentDrawer from '@/components/AgentDrawer.vue'
import SearchTable from '@/components/SearchTable.vue'
import { AI_CONTEXT_UNSAVED_CHANGES_SECTION_ID } from '../selection'
import type { AIContextProvider, AIContextSnapshot } from '../types'

const mocks = vi.hoisted(() => ({
  register: vi.fn(),
  snapshot: vi.fn(),
  createSession: vi.fn(),
  sendChatMessage: vi.fn(),
  getSessions: vi.fn(),
  getSessionInfo: vi.fn(),
  deleteSession: vi.fn(),
  getInstanceDetail: vi.fn()
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

vi.mock('@/api/service/instance', () => ({
  getInstanceDetail: mocks.getInstanceDetail
}))

vi.mock('vue-clipboard3', () => ({
  default: () => ({ toClipboard: vi.fn() })
}))

vi.mock('@/components/ai-chat/MessageList.vue', async () => {
  const { defineComponent, h } = await import('vue')
  return {
    default: defineComponent({
      name: 'MessageList',
      props: {
        messages: { type: Array, default: () => [] }
      },
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
      props: ['messages', 'isLoading', 'labels', 'showHistory'],
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

vi.mock('@/components/ai-context/AIContextPreview.vue', async () => {
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

const createMessageStream = (content: string) => {
  const encoder = new TextEncoder()
  return new ReadableStream({
    start(controller) {
      controller.enqueue(
        encoder.encode(
          `event: message_start\ndata: {}\n\nevent: content_block_delta\ndata: ${JSON.stringify({
            index: 0,
            delta: { type: 'text_delta', text: content }
          })}\n\nevent: message_stop\ndata: {}\n\n`
        )
      )
      controller.close()
    }
  })
}

const routerHost = defineComponent({
  setup: () => () => h(RouterView)
})

const layoutStubs = {
  'a-row': { template: '<div><slot /></div>' },
  'a-col': { template: '<div><slot /></div>' }
}

const searchTableStubs = {
  'a-button': true,
  'a-card': true,
  'a-col': true,
  'a-flex': true,
  'a-form': true,
  'a-form-item': true,
  'a-input': true,
  'a-radio-button': true,
  'a-radio-group': true,
  'a-row': true,
  'a-select': true,
  'a-select-option': true,
  'a-skeleton-button': true,
  'a-table': true
}

const instanceDetailStubs = {
  'a-card': true,
  'a-card-grid': true,
  'a-col': true,
  'a-descriptions': true,
  'a-descriptions-item': true,
  'a-flex': true,
  'a-row': true,
  'a-space': true,
  'a-tag': true,
  'a-typography-link': true,
  'a-typography-paragraph': true
}

describe('page AI context integration', () => {
  beforeEach(() => {
    mocks.register.mockReset()
    mocks.register.mockImplementation(() => vi.fn())
    mocks.getInstanceDetail.mockReset()
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

  it('collects the selected service and its qualifiers from the detail route', async () => {
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/services/:pathId/:group/:version',
          component: ServiceTabHeaderSlot
        }
      ]
    })
    await router.push('/services/org.apache.dubbo.DemoService/prod/1.0.0')
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
    expect(provider.collect()).toEqual({
      scope: { service: 'org.apache.dubbo.DemoService' },
      state: { selection: { group: 'prod', version: '1.0.0' } }
    })
    wrapper.unmount()
  })

  it('collects instance details after the page data loads', async () => {
    mocks.getInstanceDetail.mockResolvedValue({
      data: {
        appName: 'shop-comment',
        ip: '10.244.1.83',
        rpcPort: '20887',
        lifecycleState: 'Running',
        registerState: 'Registered',
        deployState: 'Ready',
        deployCluster: 'prod-k8s',
        registerClusters: ['nacos2.5'],
        workloadName: 'shop-comment(deployment)',
        labels: { app: 'shop-comment', version: 'v1' }
      }
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/instances/:name/:pathId/:appName',
          component: InstanceDetailPage
        }
      ]
    })
    await router.push('/instances/shop-comment10.244.1.83:20887/10.244.1.83/shop-comment')
    await router.isReady()

    const wrapper = shallowMount(InstanceDetailPage, {
      global: {
        plugins: [router],
        mocks: { $t: (key: string) => key },
        stubs: instanceDetailStubs
      }
    })
    await flushPromises()

    const provider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(provider.id).toBe('instance-detail')
    expect(provider.collect()).toMatchObject({
      evidence: {
        id: 'instance-detail',
        data: {
          application: 'shop-comment',
          ip: '10.244.1.83',
          rpcPort: '20887',
          lifecycleState: 'Running',
          registerState: 'Registered',
          deployState: 'Ready',
          deployCluster: 'prod-k8s',
          registerClusters: 'nacos2.5',
          workloadName: 'shop-comment(deployment)',
          labels: '{"app":"shop-comment","version":"v1"}'
        }
      }
    })
    wrapper.unmount()
  })

  it('collects current filters from the shared search table', async () => {
    const searchDomain = reactive({
      params: [{ param: 'keywords' }, { param: 'status' }],
      queryForm: { keywords: 'shop', status: '' },
      noPaged: false,
      paged: { pageSize: 10, curPage: 1, total: 0 },
      table: { columns: [] },
      tableStyle: {},
      result: [],
      onSearch: vi.fn()
    })
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/applications', component: { template: '<div />' } }]
    })
    await router.push('/applications')
    await router.isReady()

    const wrapper = shallowMount(SearchTable, {
      global: {
        plugins: [router],
        provide: { [PROVIDE_INJECT_KEY.SEARCH_DOMAIN]: searchDomain },
        mocks: { $t: (key: string) => key },
        stubs: searchTableStubs
      }
    })

    const provider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(provider.collect()).toEqual({ state: { filters: { keywords: 'shop' } } })

    const resultsProvider = mocks.register.mock.calls[1][0] as AIContextProvider
    expect(resultsProvider.collect()).toMatchObject({
      evidence: {
        id: 'search-results',
        data: { rows: [], total: 0 }
      }
    })

    searchDomain.queryForm.keywords = 'shop-order'
    searchDomain.queryForm.status = 'healthy'
    expect(provider.collect()).toEqual({
      state: { filters: { keywords: 'shop-order', status: 'healthy' } }
    })
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

  it('parses the same SSE response with context enabled or disabled', async () => {
    const consoleLog = vi.spyOn(console, 'log').mockImplementation(() => undefined)
    mocks.sendChatMessage.mockImplementation(async () => createMessageStream('streamed answer'))
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
    const exposedInput = input.vm.$.exposed as { inputMessage: { value: string } }

    exposedInput.inputMessage.value = 'with context'
    await input.vm.$emit('sendMessage')
    await flushPromises()

    await preview.vm.$emit('update:enabled', false)
    exposedInput.inputMessage.value = 'without context'
    await input.vm.$emit('sendMessage')
    await flushPromises()

    expect(mocks.sendChatMessage.mock.calls[0][2]).toEqual(snapshot)
    expect(mocks.sendChatMessage.mock.calls[1][2]).toBeUndefined()

    const messages = wrapper.findComponent({ name: 'MessageList' }).props('messages') as Array<{
      role: string
      content: string
    }>
    expect(
      messages.filter((item) => item.role === 'assistant').map((item) => item.content)
    ).toEqual(['streamed answer', 'streamed answer'])
    consoleLog.mockRestore()
    wrapper.unmount()
  })
})
