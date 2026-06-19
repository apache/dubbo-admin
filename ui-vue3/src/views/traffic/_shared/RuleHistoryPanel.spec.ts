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

const version = (
  id: string,
  versionNo: number,
  operation: RuleVersion['operation'],
  isCurrent: boolean
): RuleVersion => ({
  id,
  ruleKind: 'ConditionRoute',
  mesh: '',
  resourceKey: '/demo-rule',
  ruleName: 'demo-rule',
  versionNo,
  contentHash: `hash-${id}`,
  specJson: '{"key":"demo-rule"}',
  source: 'ADMIN',
  operation,
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
  props: ['value'],
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

const mountPanel = () =>
  mount(RuleHistoryPanel, {
    props: {
      open: true,
      kind: 'condition-rule',
      ruleName: 'demo-rule',
      title: 'History'
    },
    global: {
      plugins: [i18n],
      stubs: {
        RuleHistoryDrawer: drawerStub,
        MonacoEditor: true,
        RuleDiffEditor: true,
        AModal: modalStub,
        'a-modal': modalStub,
        AAlert: { template: '<div data-test="alert">{{ message }}</div>', props: ['message'] },
        'a-alert': { template: '<div data-test="alert">{{ message }}</div>', props: ['message'] },
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
  it('renders deleted warning from explicit list state and sends rollback CAS token 0', async () => {
    mocks.listRuleVersionsAPI.mockResolvedValue({
      code: HTTP_STATUS.SUCCESS,
      data: {
        items: [
          version('delete-marker', 3, 'DELETE', false),
          version('target-version', 2, 'UPDATE', false)
        ],
        total: 2,
        deleted: true
      }
    })
    mocks.rollbackRuleVersionAPI.mockResolvedValue({
      code: HTTP_STATUS.SUCCESS,
      data: {
        rolledBackFromId: 'target-version',
        versionId: '4',
        versionNo: 4,
        source: 'ROLLBACK',
        committed: true
      }
    })

    const wrapper = mountPanel()
    await flushPromises()

    await wrapper.get('[data-test="rollback-target-version"]').trigger('click')
    await nextTick()
    expect(wrapper.text()).toContain('当前规则已删除，回滚会重新创建该规则')

    await wrapper.get('[data-test="rollback-reason"]').setValue('restore deleted rule')
    await wrapper.get('[data-test="modal-ok"]').trigger('click')
    await flushPromises()

    expect(mocks.rollbackRuleVersionAPI).toHaveBeenCalledWith(
      'condition-rule',
      'demo-rule',
      'target-version',
      'restore deleted rule',
      '0'
    )
  })
})
