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
import { HTTP_STATUS } from '@/base/http/constants'
import type { RuleVersion } from '@/api/service/traffic'
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

const version = (id: string, versionNo: number, isCurrent: boolean): RuleVersion => ({
  id,
  ruleKind: 'ConditionRoute',
  mesh: '',
  resourceKey: '/demo-rule',
  ruleName: 'demo-rule',
  versionNo,
  contentHash: `hash-${id}`,
  specJson: '{"key":"demo-rule"}',
  source: 'ADMIN',
  operation: 'UPDATE',
  author: 'admin',
  createdAt: '2026-06-19T00:00:00Z',
  isCurrent
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
              'data-test': `rollback-${item.id}`,
              onClick: () => emit('rollback', item)
            },
            `rollback-${item.id}`
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
let RuleHistoryPanel: typeof RuleHistoryPanelType

beforeAll(async () => {
  i18n = (await import('@/base/i18n')).i18n
  RuleHistoryPanel = (await import('./RuleHistoryPanel.vue')).default
})

beforeEach(() => {
  mocks.listRuleVersionsAPI.mockReset()
  mocks.rollbackRuleVersionAPI.mockReset()
  mocks.diffRuleVersionAPI.mockReset()
})

describe('RuleHistoryPanel', () => {
  it('ignores stale history responses after ruleName changes', async () => {
    let resolveFirst: (value: unknown) => void = () => undefined
    mocks.listRuleVersionsAPI
      .mockReturnValueOnce(new Promise((resolve) => (resolveFirst = resolve)))
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version('new-current', 7, true)],
          total: 1,
          currentVersionId: 'new-current',
          currentVersionNo: 7,
          deleted: false
        }
      })

    const wrapper = mountPanel({ ruleName: 'old-rule' })
    await wrapper.setProps({ ruleName: 'new-rule' })
    await flushPromises()

    resolveFirst({
      code: HTTP_STATUS.SUCCESS,
      data: {
        items: [version('old-current', 3, true)],
        total: 1,
        currentVersionId: 'old-current',
        currentVersionNo: 3,
        deleted: false
      }
    })
    await flushPromises()

    expect(wrapper.emitted('current-version-change')?.at(-1)).toEqual(['new-current'])
    expect(wrapper.emitted('current-version-no-change')?.at(-1)).toEqual([7])
    expect(wrapper.text()).toContain('rollback-new-current')
    expect(wrapper.text()).not.toContain('rollback-old-current')
  })

  it('ignores stale rollback success after ruleName changes', async () => {
    mocks.listRuleVersionsAPI
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version('old-target', 1, false)],
          total: 1,
          currentVersionId: 'old-current',
          currentVersionNo: 2,
          deleted: false
        }
      })
      .mockResolvedValueOnce({
        code: HTTP_STATUS.SUCCESS,
        data: {
          items: [version('new-target', 3, false)],
          total: 1,
          currentVersionId: 'new-current',
          currentVersionNo: 4,
          deleted: false
        }
      })
    let resolveRollback: (value: unknown) => void = () => undefined
    mocks.rollbackRuleVersionAPI.mockReturnValueOnce(
      new Promise((resolve) => (resolveRollback = resolve))
    )

    const wrapper = mountPanel({ ruleName: 'old-rule' })
    await flushPromises()
    await wrapper.get('[data-test="rollback-old-target"]').trigger('click')
    await nextTick()
    await wrapper.get('[data-test="rollback-reason"]').setValue('restore old')
    await wrapper.get('[data-test="modal-ok"]').trigger('click')

    await wrapper.setProps({ ruleName: 'new-rule' })
    await flushPromises()
    await wrapper.get('[data-test="rollback-new-target"]').trigger('click')
    await nextTick()

    resolveRollback({
      code: HTTP_STATUS.SUCCESS,
      data: {
        rolledBackFromId: 'old-target',
        versionId: 'old-rollback',
        versionNo: 5,
        source: 'ROLLBACK',
        committed: true
      }
    })
    await flushPromises()

    expect(wrapper.text()).toContain('rollback-new-target')
    expect(wrapper.find('[data-test="modal"]').exists()).toBe(true)
    expect(mocks.listRuleVersionsAPI).toHaveBeenCalledTimes(2)
  })
})
