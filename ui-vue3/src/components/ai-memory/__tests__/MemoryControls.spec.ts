/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount, type VueWrapper } from '@vue/test-utils'
import { createI18n } from 'vue-i18n'
import Antd, { Select } from 'ant-design-vue'
import { defineComponent, h } from 'vue'
import en from '@/base/i18n/en'
import { AIAPIError, aiMemoryService, type ExplicitMemory } from '@/api/service/aiMemory'
import MemoryEditor from '../MemoryEditor.vue'
import MemoryPanel from '../MemoryPanel.vue'
import MemoryMatchSelector from '../MemoryMatchSelector.vue'

vi.mock('@/api/service/aiMemory', async (original) => {
  const actual = await original<typeof import('@/api/service/aiMemory')>()
  return {
    ...actual,
    aiMemoryService: {
      list: vi.fn(),
      save: vi.fn(),
      options: vi.fn(),
      delete: vi.fn(),
      clear: vi.fn(),
      setEnabled: vi.fn()
    }
  }
})
const settings = { generation: 2, state_version: 3, enabled: true }
const options = {
  projects: [{ id: 'project-a' }],
  task_types: ['diagnosis', 'configuration', 'inspection']
}
const record: ExplicitMemory = {
  id: 'memory-1',
  memory_type: 'preference',
  fact_key: 'response.language',
  value: '<script>old</script>',
  applicability: {},
  status: 'confirmed',
  version: 4,
  provenance: 'explicit',
  source_session_id: 'session-source',
  updated_at: new Date().toISOString(),
  expires_at: '2020-01-01T00:00:00Z'
}
let wrapper: VueWrapper
const ModalStub = defineComponent({
  name: 'AModal',
  props: ['open'],
  emits: ['cancel', 'ok'],
  setup:
    (props, { slots }) =>
    () =>
      props.open ? h('section', slots.default?.()) : null
})
function globalOptions() {
  return {
    plugins: [Antd, createI18n({ legacy: false, locale: 'en', messages: { en } })],
    stubs: { 'a-modal': ModalStub }
  }
}
beforeEach(() => {
  vi.clearAllMocks()
  vi.stubGlobal(
    'matchMedia',
    vi.fn(() => ({ matches: false, addListener: vi.fn(), removeListener: vi.fn() }))
  )
  vi.spyOn(aiMemoryService, 'list').mockResolvedValue({ ...settings, records: [record] })
  vi.spyOn(aiMemoryService, 'options').mockResolvedValue(options as never)
})
afterEach(() => {
  wrapper?.unmount()
  document.body.innerHTML = ''
  vi.unstubAllGlobals()
})

describe('Memory editor', () => {
  it('maps business categories to typed applicability, requiring a project', async () => {
    wrapper = mount(MemoryEditor, {
      props: { settings, options: options as never },
      global: globalOptions()
    })
    await wrapper.findComponent(Select).vm.$emit('update:value', 'background')
    await flushPromises()
    await wrapper.find('textarea').setValue('project context')
    await wrapper.find('form').trigger('submit')
    expect(aiMemoryService.save).not.toHaveBeenCalled()
    await wrapper.findAllComponents(Select)[1].vm.$emit('update:value', 'project-a')
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(aiMemoryService.save).toHaveBeenCalledWith(
      {
        generation: 2,
        state_version: 3,
        scope_type: 'user',
        memory_type: 'task_context',
        fact_key: 'project.background',
        value: 'project context',
        applicability: { projectRef: 'project-a' },
        expires_at: null
      },
      undefined
    )
  })
  it('uses the original record and control versions and preserves a conflicting draft', async () => {
    vi.mocked(aiMemoryService.save).mockRejectedValue(
      new AIAPIError('MEMORY_VERSION_CONFLICT', 409)
    )
    wrapper = mount(MemoryEditor, {
      props: { settings, options: options as never, record },
      global: globalOptions()
    })
    await wrapper.find('input[type="datetime-local"]').setValue('')
    await wrapper.find('textarea').setValue('my corrected draft')
    await wrapper.setProps({ settings: { ...settings, generation: 9 } })
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(aiMemoryService.save).toHaveBeenCalledWith(
      expect.objectContaining({
        generation: 2,
        state_version: 3,
        version: 4,
        value: 'my corrected draft'
      }),
      'memory-1'
    )
    expect(wrapper.find('textarea').element.value).toBe('my corrected draft')
    expect(wrapper.emitted('stale')).toHaveLength(1)
    expect(wrapper.find('[data-testid="memory-save"]').attributes('disabled')).toBeDefined()
    expect(wrapper.text()).toContain('Your draft is preserved')
  })
  it('fixes read-only diagnosis to the supported value and task', async () => {
    wrapper = mount(MemoryEditor, {
      props: { settings, options: options as never },
      global: globalOptions()
    })
    await wrapper.findComponent(Select).vm.$emit('update:value', 'readOnly')
    await flushPromises()
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(aiMemoryService.save).toHaveBeenCalledWith(
      expect.objectContaining({
        value: 'true',
        fact_key: 'diagnostic.read_only',
        applicability: { taskType: 'diagnosis' }
      }),
      undefined
    )
  })
})
describe('Memory panel', () => {
  it('blocks editing unavailable sources but allows confirmed deletion', async () => {
    vi.mocked(aiMemoryService.list).mockResolvedValue({
      ...settings,
      records: [{ ...record, status: 'source_unavailable', value: '' }]
    })
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Source conversation unavailable; content hidden')
    const edit = wrapper.find('[data-testid="memory-edit"]')
    expect(edit.attributes('disabled')).toBeDefined()
    await edit.trigger('click')
    expect(wrapper.findComponent(MemoryEditor).exists()).toBe(false)
    expect(aiMemoryService.save).not.toHaveBeenCalled()
    expect(wrapper.find('[data-testid="memory-delete"]').attributes('disabled')).toBeUndefined()
    await wrapper.find('[data-testid="memory-delete"]').trigger('click')
    expect(aiMemoryService.delete).not.toHaveBeenCalled()
    await wrapper.findAllComponents(ModalStub).at(-1)!.vm.$emit('ok')
    await flushPromises()
    expect(aiMemoryService.delete).toHaveBeenCalledWith(record.id, settings, record.version)
  })

  it('allows editing an expired memory whose source is available', async () => {
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    expect(wrapper.find('[data-testid="memory-edit"]').attributes('disabled')).toBeUndefined()
    await wrapper.find('[data-testid="memory-edit"]').trigger('click')
    expect(wrapper.findComponent(MemoryEditor).exists()).toBe(true)
  })

  it('clears a load error after a later refresh succeeds', async () => {
    vi.mocked(aiMemoryService.list)
      .mockRejectedValueOnce(new AIAPIError('MEMORY_UNAVAILABLE', 503))
      .mockResolvedValueOnce({ ...settings, records: [record] })
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    expect(wrapper.text()).toContain('Memory storage is unavailable')
    await wrapper.find('[data-testid="memory-refresh"]').trigger('click')
    await flushPromises()
    expect(wrapper.text()).not.toContain('Memory storage is unavailable')
    expect(wrapper.find('[data-testid="memory-record"]').exists()).toBe(true)
  })

  it('keeps a mutation error visible when the follow-up refresh succeeds', async () => {
    vi.mocked(aiMemoryService.delete).mockRejectedValue(
      new AIAPIError('MEMORY_VERSION_CONFLICT', 409)
    )
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    await wrapper.find('[data-testid="memory-delete"]').trigger('click')
    await wrapper.findAllComponents(ModalStub).at(-1)!.vm.$emit('ok')
    await flushPromises()
    expect(wrapper.text()).toContain('Memory changed elsewhere')
    expect(wrapper.find('[data-testid="memory-record"]').exists()).toBe(true)
  })

  it('displays escaped content, provenance and expiry, and deletes only after confirmation', async () => {
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    expect(wrapper.text()).toContain('<script>old</script>')
    expect(wrapper.find('script').exists()).toBe(false)
    expect(wrapper.text()).toContain('Expired')
    expect(wrapper.text()).toContain('session-source')
    await wrapper.find('[data-testid="memory-delete"]').trigger('click')
    expect(aiMemoryService.delete).not.toHaveBeenCalled()
    const confirmation = wrapper.findAllComponents(ModalStub).at(-1)!
    await confirmation.vm.$emit('cancel')
    expect(aiMemoryService.delete).not.toHaveBeenCalled()
    await wrapper.find('[data-testid="memory-delete"]').trigger('click')
    await confirmation.vm.$emit('ok')
    await flushPromises()
    expect(aiMemoryService.delete).toHaveBeenCalledWith('memory-1', settings, 4)
  })
  it('loads bounded pages and confirms user-wide disable and clear', async () => {
    vi.mocked(aiMemoryService.list).mockResolvedValueOnce({
      ...settings,
      records: [record],
      next_cursor: 'page-2'
    })
    wrapper = mount(MemoryPanel, {
      props: { open: true, options: options as never },
      global: globalOptions()
    })
    await flushPromises()
    await wrapper.find('[data-testid="memory-more"]').trigger('click')
    await flushPromises()
    expect(aiMemoryService.list).toHaveBeenLastCalledWith('page-2')
    expect(wrapper.findAll('[data-testid="memory-record"]')).toHaveLength(1)
    await wrapper.find('[data-testid="memory-toggle"]').trigger('click')
    expect(aiMemoryService.setEnabled).not.toHaveBeenCalled()
    await wrapper.findAllComponents(ModalStub).at(-1)!.vm.$emit('ok')
    await flushPromises()
    expect(aiMemoryService.setEnabled).toHaveBeenCalledWith(settings, false)
    await wrapper.find('[data-testid="memory-clear"]').trigger('click')
    await wrapper.findAllComponents(ModalStub).at(-1)!.vm.$emit('ok')
    await flushPromises()
    expect(aiMemoryService.clear).toHaveBeenCalledWith(settings)
  })
})
describe('Matching selection', () => {
  it('clears stale selection when authorized options cannot be loaded', async () => {
    vi.mocked(aiMemoryService.options).mockRejectedValue(
      new AIAPIError('AUTHENTICATION_REQUIRED', 401)
    )
    wrapper = mount(MemoryMatchSelector, {
      props: { active: true, modelValue: { projectRef: 'old-project' } },
      global: globalOptions()
    })
    await flushPromises()
    expect(wrapper.emitted('update:modelValue')).toEqual([[{}]])
    expect(wrapper.text()).toContain('Sign in again')
    expect(wrapper.findComponent(Select).props('disabled')).toBe(true)
  })
})
