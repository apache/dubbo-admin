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
import { defineComponent, h, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { HTTP_STATUS } from '@/base/http/constants'
import type { RuleVersion } from '@/api/service/traffic'
import type RuleHistoryDrawerType from './RuleHistoryDrawer.vue'
import type RuleHistoryPanelType from './RuleHistoryPanel.vue'

const mocks = vi.hoisted(() => {
  Object.defineProperty(globalThis, 'localStorage', {
    value: {
      getItem: () => null,
      setItem: () => undefined,
      removeItem: () => undefined
    },
    configurable: true
  })

  return {
    listRuleVersionsAPI: vi.fn(),
    rollbackRuleVersionAPI: vi.fn(),
    diffRuleVersionAPI: vi.fn()
  }
})

vi.mock('@/api/service/traffic', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/api/service/traffic')>()
  return {
    ...actual,
    listRuleVersionsAPI: mocks.listRuleVersionsAPI,
    rollbackRuleVersionAPI: mocks.rollbackRuleVersionAPI,
    diffRuleVersionAPI: mocks.diffRuleVersionAPI
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

vi.mock('@/components/editor/MonacoEditor.vue', () => ({
  default: {
    name: 'MonacoEditor',
    template: '<div data-test="monaco-editor" />'
  }
}))

vi.mock('./RuleDiffEditor.vue', () => ({
  default: {
    name: 'RuleDiffEditor',
    template: '<div data-test="rule-diff-editor" />'
  }
}))

const version = (
  versionNo: number,
  isLatestRecorded: boolean,
  overrides: Partial<RuleVersion> = {}
): RuleVersion => ({
  ruleKind: 'ConditionRoute',
  mesh: '',
  resourceKey: '/demo-rule',
  ruleName: 'demo-rule',
  versionNo,
  contentHash: `hash-${versionNo}`,
  specJson: '{"key":"demo-rule"}',
  source: 'ADMIN',
  operation: 'UPDATE',
  author: 'admin',
  createdAt: '2026-06-19T00:00:00Z',
  isLatestRecorded,
  ...overrides
})

const drawerStub = defineComponent({
  props: ['items'],
  emits: ['rollback'],
  setup(props, { emit }) {
    return () =>
      h(
        'div',
        { 'data-test': 'history-drawer' },
        (props.items as RuleVersion[]).map((item) =>
          h(
            'button',
            {
              type: 'button',
              'data-test': `rollback-${item.versionNo}`,
              onClick: () => emit('rollback', item)
            },
            `rollback-${item.versionNo}`
          )
        )
      )
  }
})

const modalStub = defineComponent({
  props: ['open'],
  emits: ['ok'],
  setup(props, { emit, slots }) {
    return () =>
      props.open
        ? h('div', { 'data-test': 'modal' }, [
            slots.default?.(),
            h(
              'button',
              { type: 'button', 'data-test': 'modal-ok', onClick: () => emit('ok') },
              'ok'
            )
          ])
        : null
  }
})

const textAreaStub = defineComponent({
  emits: ['update:value'],
  setup(_props, { emit }) {
    return () =>
      h('textarea', {
        'data-test': 'rollback-reason',
        onInput: (event: Event) => {
          emit('update:value', (event.target as HTMLTextAreaElement).value)
        }
      })
  }
})

const mountPanel = (props: Partial<InstanceType<typeof RuleHistoryPanelType>['$props']> = {}) =>
  mount(RuleHistoryPanel, {
    props: {
      open: true,
      kind: 'condition-rule',
      ruleName: 'demo-rule',
      title: 'History',
      ...props
    },
    global: {
      plugins: [i18n],
      stubs: {
        RuleHistoryDrawer: drawerStub,
        MonacoEditor: true,
        RuleDiffEditor: true,
        AModal: modalStub,
        'a-modal': modalStub,
        AAlert: { template: '<div />' },
        'a-alert': { template: '<div />' },
        ATypographyText: { template: '<span><slot /></span>' },
        'a-typography-text': { template: '<span><slot /></span>' },
        AForm: { template: '<form><slot /></form>' },
        'a-form': { template: '<form><slot /></form>' },
        AFormItem: { template: '<label><slot /></label>' },
        'a-form-item': { template: '<label><slot /></label>' },
        ATextarea: textAreaStub,
        'a-textarea': textAreaStub
      }
    }
  })

let i18n: typeof import('@/base/i18n').i18n
let RuleHistoryDrawer: typeof RuleHistoryDrawerType
let RuleHistoryPanel: typeof RuleHistoryPanelType

beforeAll(async () => {
  i18n = (await import('@/base/i18n')).i18n
  RuleHistoryDrawer = (await import('./RuleHistoryDrawer.vue')).default
  RuleHistoryPanel = (await import('./RuleHistoryPanel.vue')).default
})

beforeEach(() => {
  mocks.listRuleVersionsAPI.mockReset()
  mocks.rollbackRuleVersionAPI.mockReset()
  mocks.diffRuleVersionAPI.mockReset()
  vi.mocked(message.error).mockClear()
  vi.mocked(message.success).mockClear()
  vi.mocked(message.warning).mockClear()
})

const mountDrawer = (items: RuleVersion[]) =>
  mount(RuleHistoryDrawer, {
    props: {
      open: true,
      title: 'History',
      items
    },
    global: {
      plugins: [i18n],
      stubs: {
        ADrawer: {
          props: ['open'],
          template: '<div v-if="open"><slot name="title" /><slot /></div>'
        },
        'a-drawer': {
          props: ['open'],
          template: '<div v-if="open"><slot name="title" /><slot /></div>'
        },
        ASpin: { template: '<div><slot /></div>' },
        'a-spin': { template: '<div><slot /></div>' },
        AEmpty: { template: '<div data-test="empty" />' },
        'a-empty': { template: '<div data-test="empty" />' },
        ASpace: { template: '<div><slot /></div>' },
        'a-space': { template: '<div><slot /></div>' },
        ATag: { template: '<span><slot /></span>' },
        'a-tag': { template: '<span><slot /></span>' },
        ATooltip: { template: '<div><slot /></div>' },
        'a-tooltip': { template: '<div><slot /></div>' },
        AButton: {
          props: {
            disabled: {
              type: Boolean,
              default: false
            }
          },
          emits: ['click'],
          template:
            '<button :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>'
        },
        'a-button': {
          props: {
            disabled: {
              type: Boolean,
              default: false
            }
          },
          emits: ['click'],
          template:
            '<button :disabled="disabled" @click="$emit(\'click\', $event)"><slot /></button>'
        },
        ATypographyText: { template: '<span><slot /></span>' },
        'a-typography-text': { template: '<span><slot /></span>' }
      }
    }
  })

const rollbackButton = (wrapper: ReturnType<typeof mountDrawer>) => wrapper.findAll('button').at(-1)

describe('RuleHistoryPanel', () => {
  it('allows rollback for latest recorded non-delete versions', () => {
    const wrapper = mountDrawer([version(3, true)])

    rollbackButton(wrapper)?.trigger('click')

    expect(wrapper.emitted('rollback')?.[0][0]).toMatchObject({ versionNo: 3 })
  })

  it('disables rollback for delete markers', () => {
    const wrapper = mountDrawer([version(4, true, { operation: 'DELETE', specJson: '<deleted>' })])

    rollbackButton(wrapper)?.trigger('click')

    expect(wrapper.emitted('rollback')).toBeUndefined()
  })

  it('ignores stale history responses after ruleName changes', async () => {
    let resolveFirst: (value: unknown) => void = () => undefined
    mocks.listRuleVersionsAPI
      .mockReturnValueOnce(new Promise((resolve) => (resolveFirst = resolve)))
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version(7, true)],
          total: 1,
          latestRecordedVersionNo: 7,
          latestRecordedDeleted: false
        }
      })

    const wrapper = mountPanel({ ruleName: 'old-rule' })
    await wrapper.setProps({ ruleName: 'new-rule' })
    await flushPromises()

    resolveFirst({
      code: HTTP_STATUS.SUCCESS,
      data: {
        items: [version(3, true)],
        total: 1,
        latestRecordedVersionNo: 3,
        latestRecordedDeleted: false
      }
    })
    await flushPromises()

    expect(wrapper.emitted('latest-recorded-version-no-change')?.at(-1)).toEqual([7])
    expect(wrapper.text()).toContain('rollback-7')
    expect(wrapper.text()).not.toContain('rollback-3')
  })

  it('ignores stale rollback success after ruleName changes', async () => {
    mocks.listRuleVersionsAPI
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version(1, false)],
          total: 1,
          latestRecordedVersionNo: 2,
          latestRecordedDeleted: false
        }
      })
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version(3, false)],
          total: 1,
          latestRecordedVersionNo: 4,
          latestRecordedDeleted: false
        }
      })
    let resolveRollback: (value: unknown) => void = () => undefined
    mocks.rollbackRuleVersionAPI.mockReturnValueOnce(
      new Promise((resolve) => (resolveRollback = resolve))
    )

    const wrapper = mountPanel({ ruleName: 'old-rule' })
    await flushPromises()
    await wrapper.get('[data-test="rollback-1"]').trigger('click')
    await nextTick()
    await wrapper.get('[data-test="rollback-reason"]').setValue('restore old')
    await wrapper.get('[data-test="modal-ok"]').trigger('click')

    await wrapper.setProps({ ruleName: 'new-rule' })
    await flushPromises()
    await wrapper.get('[data-test="rollback-3"]').trigger('click')
    await nextTick()

    resolveRollback({
      code: HTTP_STATUS.SUCCESS,
      data: {
        rolledBackFromVersionNo: 1,
        versionNo: 5,
        source: 'ROLLBACK'
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('rollback-3')
    expect(wrapper.find('[data-test="modal"]').exists()).toBe(true)
    expect(mocks.listRuleVersionsAPI).toHaveBeenCalledTimes(2)
  })

  it('shows backend rollback rejection errors', async () => {
    mocks.listRuleVersionsAPI.mockResolvedValue({
      code: HTTP_STATUS.SUCCESS,
      data: {
        items: [version(1, true)],
        total: 1,
        latestRecordedVersionNo: 1,
        latestRecordedDeleted: false
      }
    })
    mocks.rollbackRuleVersionAPI.mockRejectedValue({
      code: 'InvalidArgument',
      message: 'cannot roll back to a version identical to current'
    })

    const wrapper = mountPanel()
    await flushPromises()
    await wrapper.get('[data-test="rollback-1"]').trigger('click')
    await nextTick()
    await wrapper.get('[data-test="rollback-reason"]').setValue('same content')
    await wrapper.get('[data-test="modal-ok"]').trigger('click')
    await flushPromises()

    expect(message.error).toHaveBeenCalledWith('cannot roll back to a version identical to current')
  })
})
