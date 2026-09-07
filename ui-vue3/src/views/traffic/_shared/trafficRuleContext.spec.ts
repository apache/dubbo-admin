import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { AIContextProvider } from '@/ai-context/types'
import { HTTP_STATUS } from '@/base/http/constants'
import ConditionFormView from '../routingRule/tabs/formView.vue'
import TagFormView from '../tagRule/tabs/formView.vue'

const mocks = vi.hoisted(() => ({
  register: vi.fn(),
  unregister: vi.fn(),
  getConditionRuleDetailAPI: vi.fn(),
  getTagRuleDetailAPI: vi.fn()
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

vi.mock('@/ai-context/instance', () => ({
  aiContextManager: { register: mocks.register }
}))

vi.mock('@/api/service/traffic', () => ({
  getConditionRuleDetailAPI: mocks.getConditionRuleDetailAPI,
  getTagRuleDetailAPI: mocks.getTagRuleDetailAPI
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ params: { ruleName: 'demo-rule' }, fullPath: '/traffic/demo-rule' })
}))

vi.mock('vue-clipboard3', () => ({
  default: () => ({ toClipboard: vi.fn() })
}))

vi.mock('./RuleHistoryPanel.vue', () => ({
  default: {
    name: 'RuleHistoryPanel',
    props: ['open', 'kind', 'ruleName', 'title'],
    template: '<div />'
  }
}))

beforeEach(() => {
  vi.clearAllMocks()
  mocks.register.mockReturnValue(mocks.unregister)
})

describe('traffic history and AI context integration', () => {
  it.each([
    {
      kind: 'condition-rule',
      component: ConditionFormView,
      getDetail: mocks.getConditionRuleDetailAPI,
      entries: { conditions: ['host=1.1.1.1 => host=2.2.2.2'] }
    },
    {
      kind: 'tag-rule',
      component: TagFormView,
      getDetail: mocks.getTagRuleDetailAPI,
      entries: { tags: [{ name: 'gray', match: [{ key: 'env', value: { exact: 'gray' } }] }] }
    }
  ])('keeps history and loaded $kind content available together', async (testCase) => {
    const detail = {
      configVersion: 'v3.0',
      scope: 'application',
      key: 'shop-user',
      enabled: false,
      runtime: false,
      ...testCase.entries
    }
    testCase.getDetail.mockResolvedValue({ code: HTTP_STATUS.SUCCESS, data: detail })

    const wrapper = shallowMount(testCase.component, {
      global: {
        mocks: { $t: (key: string) => key },
        stubs: Object.fromEntries(
          [
            'a-col',
            'a-flex',
            'a-row',
            'a-card',
            'a-typography-title',
            'a-tag',
            'a-button',
            'a-space',
            'a-descriptions-item',
            'a-typography-paragraph',
            'a-descriptions',
            'a-typography-text'
          ].map((name) => [name, true])
        )
      }
    })
    await flushPromises()

    const history = wrapper.findComponent({ name: 'RuleHistoryPanel' })
    expect(history.props()).toMatchObject({
      kind: testCase.kind,
      ruleName: 'demo-rule',
      title: 'shop-user'
    })
    const provider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(provider.collect()).toMatchObject({
      evidence: { source: 'traffic-rule-page', data: { kind: testCase.kind, content: detail } }
    })

    wrapper.unmount()
    expect(mocks.unregister).toHaveBeenCalledOnce()
  })
})
