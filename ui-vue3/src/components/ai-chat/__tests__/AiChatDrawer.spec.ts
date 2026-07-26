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

import { defineComponent, h, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import AiChatDrawer from '../AiChatDrawer.vue'
import type { ChatService, ChatSuggestion } from '../types'

const DrawerStub = defineComponent({
  name: 'DrawerStub',
  props: {
    open: Boolean,
    title: String
  },
  emits: ['update:open'],
  setup(_, { emit, slots }) {
    return () =>
      h('section', [
        h('button', { class: 'close-drawer', onClick: () => emit('update:open', false) }),
        slots.default?.()
      ])
  }
})

const MessageListStub = defineComponent({
  name: 'MessageList',
  props: ['messages', 'labels', 'suggestions'],
  setup(_, { expose }) {
    expose({ scrollToBottom: vi.fn() })
    return () => h('div')
  }
})

const ChatInputStub = defineComponent({
  name: 'ChatInput',
  props: ['labels', 'showHistory'],
  emits: ['sendMessage'],
  setup(_, { emit, expose }) {
    const inputMessage = ref('')
    expose({ inputMessage, focus: vi.fn() })
    return () => h('button', { class: 'send-message', onClick: () => emit('sendMessage') })
  }
})

const SessionHistoryModalStub = defineComponent({
  name: 'SessionHistoryModal',
  props: ['visible', 'sessions', 'labels'],
  setup() {
    return () => h('div')
  }
})

const createEmptyStream = () =>
  new ReadableStream<Uint8Array>({
    start(controller) {
      controller.close()
    }
  })

const createMessageStream = (content: string) => {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
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

const mountDrawer = (service: ChatService<unknown>, extraProps = {}) =>
  mount(AiChatDrawer, {
    props: {
      open: true,
      service,
      ...extraProps
    },
    global: {
      stubs: {
        'a-drawer': DrawerStub,
        MessageList: MessageListStub,
        ChatInput: ChatInputStub,
        SessionHistoryModal: SessionHistoryModalStub
      }
    }
  })

const mountDrawerWithMessages = (service: ChatService<unknown>) => {
  Object.defineProperty(HTMLElement.prototype, 'scrollTo', {
    configurable: true,
    value: vi.fn()
  })
  return mount(AiChatDrawer, {
    props: { open: true, service },
    global: {
      stubs: {
        'a-drawer': DrawerStub,
        'a-button': true,
        ChatInput: ChatInputStub
      }
    }
  })
}

describe('AiChatDrawer', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('sends messages through the injected service with collected context', async () => {
    const context = { page: 'instances', resourceId: 'demo' }
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockResolvedValue(createEmptyStream())
    }
    const collectContext = vi.fn(() => context)
    const wrapper = mountDrawer(service, { collectContext })
    const input = wrapper.findComponent(ChatInputStub)
    const exposed = input.vm.$.exposed as { inputMessage: { value: string } }

    exposed.inputMessage.value = 'inspect this resource'
    await input.find('.send-message').trigger('click')
    await flushPromises()

    expect(service.createSession).toHaveBeenCalledOnce()
    expect(collectContext).toHaveBeenCalledOnce()
    expect(service.sendChatMessage).toHaveBeenCalledWith(
      'inspect this resource',
      'session-1',
      context
    )
  })

  it('works without session history and forwards labels and suggestions', () => {
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockResolvedValue(createEmptyStream())
    }
    const suggestions: ChatSuggestion[] = [
      { icon: '?', iconColor: 'text-blue-500', title: 'Inspect', content: 'Inspect this page' }
    ]
    const wrapper = mountDrawer(service, {
      labels: { title: 'Project Assistant' },
      suggestions
    })

    expect(wrapper.findComponent(ChatInputStub).props('showHistory')).toBe(false)
    expect(wrapper.findComponent(SessionHistoryModalStub).exists()).toBe(false)
    expect(wrapper.findComponent(MessageListStub).props('labels').title).toBe('Project Assistant')
    expect(wrapper.findComponent(MessageListStub).props('suggestions')).toEqual(suggestions)
  })

  it('enables history only when the complete history service is available', () => {
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockResolvedValue(createEmptyStream()),
      getSessions: vi.fn().mockResolvedValue([]),
      getSessionInfo: vi.fn(),
      deleteSession: vi.fn()
    }
    const wrapper = mountDrawer(service)

    expect(wrapper.findComponent(ChatInputStub).props('showHistory')).toBe(true)
    expect(wrapper.findComponent(SessionHistoryModalStub).exists()).toBe(true)
  })

  it('emits an open state update when the drawer closes', async () => {
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockResolvedValue(createEmptyStream())
    }
    const wrapper = mountDrawer(service)

    await wrapper.find('.close-drawer').trigger('click')

    expect(wrapper.emitted('update:open')).toContainEqual([false])
  })

  it('escapes raw HTML in user and model messages', async () => {
    vi.spyOn(console, 'log').mockImplementation(() => undefined)
    const rawHTML = '<img src="x" onerror="alert(1)">'
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockResolvedValue(createMessageStream(rawHTML))
    }
    const wrapper = mountDrawerWithMessages(service)
    const input = wrapper.findComponent(ChatInputStub)
    const exposed = input.vm.$.exposed as { inputMessage: { value: string } }

    exposed.inputMessage.value = rawHTML
    await input.find('.send-message').trigger('click')
    await flushPromises()

    expect(wrapper.find('img[onerror]').exists()).toBe(false)
    expect(wrapper.text()).toContain(rawHTML)
  })

  it('escapes raw HTML in error messages', async () => {
    vi.spyOn(console, 'error').mockImplementation(() => undefined)
    const rawHTML = '<img src="x" onerror="alert(1)">'
    const service: ChatService<unknown> = {
      createSession: vi.fn().mockResolvedValue('session-1'),
      sendChatMessage: vi.fn().mockRejectedValue(new Error(rawHTML))
    }
    const wrapper = mountDrawerWithMessages(service)
    const input = wrapper.findComponent(ChatInputStub)
    const exposed = input.vm.$.exposed as { inputMessage: { value: string } }

    exposed.inputMessage.value = 'trigger an error'
    await input.find('.send-message').trigger('click')
    await flushPromises()

    expect(wrapper.find('img[onerror]').exists()).toBe(false)
    expect(wrapper.text()).toContain(rawHTML)
  })
})
