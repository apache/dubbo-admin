import { flushPromises, mount } from '@vue/test-utils'
import { reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import TraceLogSearch from './TraceLogSearch.vue'
import GrafanaPage from './GrafanaPage.vue'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'

const { getTraceStartTime, getTraceLogDashboard, showError } = vi.hoisted(() => ({
  getTraceStartTime: vi.fn(),
  getTraceLogDashboard: vi.fn(),
  showError: vi.fn()
}))

vi.mock('@/api/service/trace', () => ({ getTraceStartTime, getTraceLogDashboard }))
vi.mock('ant-design-vue', () => ({ message: { error: showError } }))

const traceId = '766b9604d84bd8e6b17b2292dfd9bdd0'
const stubs = {
  'a-form': { template: '<form><slot /></form>' },
  'a-form-item': { template: '<label><slot /></label>' },
  'a-input': {
    props: ['value'],
    emits: ['update:value'],
    template: '<input :value="value" @input="$emit(\'update:value\', $event.target.value)" />'
  },
  'a-button': { template: '<button><slot /></button>' },
  'a-alert': { template: '<div />' },
  'a-spin': { template: '<div><slot /></div>' }
}

describe('TraceLogSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getTraceStartTime.mockResolvedValue({ data: { traceId, startTime: 1790774974677 } })
    getTraceLogDashboard.mockResolvedValue({
      data: { fullURL: `http://grafana/d/application-logs?var-application=.*&var-traceId=${traceId}` }
    })
  })

  it('uses the trace-focused dashboard instead of retaining the current instance filter', async () => {
    const grafana = reactive({
      api: vi.fn(),
      params: {},
      showIframe: true,
      url: 'http://grafana/d/instance-logs?var-application=shop-user&var-instance=10.0.0.1'
    })
    const wrapper = mount(TraceLogSearch, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        mocks: { $t: (key: string) => key },
        stubs
      }
    })

    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(getTraceLogDashboard).toHaveBeenCalledWith({
      traceId,
      from: 1790774674677,
      to: 1790775274677
    })
    expect(new URL(grafana.url).searchParams.get('var-instance')).toBeNull()
  })

  it('hides the previous trace results when the next lookup fails', async () => {
    getTraceStartTime.mockRejectedValueOnce({ code: 'NotFoundError' })
    const grafana = reactive({
      api: vi.fn(),
      params: {},
      showIframe: true,
      url: `http://grafana/d/application-logs?var-traceId=${traceId}`,
      error: ''
    })
    const wrapper = mount(TraceLogSearch, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        mocks: { $t: (key: string) => key },
        stubs
      }
    })

    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(grafana.showIframe).toBe(false)
    expect(grafana.error).toContain('does not exist')
  })

  it('attributes a missing trace dashboard to dashboard configuration, not Jaeger', async () => {
    getTraceLogDashboard.mockRejectedValueOnce({ code: 'NotFoundError' })
    const grafana = reactive({ api: vi.fn(), params: {}, showIframe: true, error: '' })
    const wrapper = mount(TraceLogSearch, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        mocks: { $t: (key: string) => key },
        stubs
      }
    })

    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()

    expect(grafana.error).toContain('dashboard is not configured')
    expect(grafana.error).not.toContain('Trace does not exist')
    expect(showError).toHaveBeenCalledTimes(1)
  })

  it('does not let the initial dashboard response replace a completed manual search', async () => {
    let finishInitial!: (value: unknown) => void
    const initialApi = vi.fn(() => new Promise((resolve) => { finishInitial = resolve }))
    const grafana = reactive({ api: initialApi, params: {}, showIframe: false, url: '', error: '' })
    const wrapper = mount({
      components: { TraceLogSearch, GrafanaPage },
      template: '<div><TraceLogSearch /><GrafanaPage /></div>'
    }, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        mocks: { $t: (key: string) => key },
        stubs
      }
    })

    expect(initialApi).toHaveBeenCalledOnce()
    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    const searchedUrl = grafana.url

    finishInitial({ data: { fullURL: 'http://grafana/d/application-logs?var-application=shop-user' } })
    await flushPromises()

    expect(grafana.url).toBe(searchedUrl)
    wrapper.unmount()
  })

  it('does not let a late initial-load failure hide a completed manual search', async () => {
    let failInitial!: (reason: unknown) => void
    const initialApi = vi.fn(() => new Promise((_resolve, reject) => { failInitial = reject }))
    const grafana = reactive({ api: initialApi, params: {}, showIframe: false, url: '', error: '' })
    const wrapper = mount({
      components: { TraceLogSearch, GrafanaPage },
      template: '<div><TraceLogSearch /><GrafanaPage /></div>'
    }, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        mocks: { $t: (key: string) => key },
        stubs
      }
    })

    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    failInitial(new Error('Initial dashboard unavailable'))
    await flushPromises()

    expect(grafana.showIframe).toBe(true)
    expect(grafana.error).toBe('')
    expect(showError).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})
