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
import { reactive } from 'vue'
import { createI18n } from 'vue-i18n'
import { messages } from '@/base/i18n/messages'
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
const testI18n = (locale = 'en') => createI18n({ legacy: false, locale, messages })
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
      data: {
        fullURL: `http://grafana/d/application-logs?var-application=.*&var-traceId=${traceId}`
      }
    })
  })

  it.each([
    ['en', '', 'Trace ID is required'],
    ['cn', '', '请输入 Trace ID'],
    ['en', 'invalid', 'Trace ID must be 16 or 32 hexadecimal characters'],
    ['cn', 'invalid', 'Trace ID 必须为 16 或 32 位十六进制字符']
  ])('localizes client validation in %s for %s', async (locale, input, expected) => {
    const grafana = reactive({ api: vi.fn(), params: {}, showIframe: true, error: '' })
    const wrapper = mount(TraceLogSearch, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        plugins: [testI18n(locale)],
        stubs
      }
    })
    await wrapper.get('input').setValue(input)
    await wrapper.get('button').trigger('click')
    expect(grafana.error).toBe(expected)
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expected)
    expect(getTraceStartTime).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it.each([
    ['trace', 'InvalidArgument', 'Trace ID 必须为 16 或 32 位十六进制字符'],
    ['trace', 'NotFoundError', 'Trace 不存在或没有有效的开始时间'],
    ['trace', 'ConfigError', '未配置 Jaeger 查询地址'],
    ['trace', 'NetWorkError', 'Jaeger 查询暂不可用，请稍后重试'],
    ['trace', 'unexpected', '无法获取 Trace 时间范围'],
    ['dashboard', 'NotFoundError', '未配置 Trace 日志面板'],
    ['dashboard', 'NetWorkError', '日志面板暂不可用，请稍后重试']
  ])('localizes %s %s failures without duplicate errors', async (stage, code, expected) => {
    const api = stage === 'trace' ? getTraceStartTime : getTraceLogDashboard
    api.mockRejectedValueOnce({ code })
    const grafana = reactive({ api: vi.fn(), params: {}, showIframe: true, error: '' })
    const wrapper = mount(TraceLogSearch, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        plugins: [testI18n('cn')],
        stubs
      }
    })
    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(grafana.error).toBe(expected)
    expect(grafana.showIframe).toBe(false)
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expected)
    wrapper.unmount()
  })

  it.each([
    ['en', 'missing', 'Grafana dashboard is not configured'],
    ['cn', 'missing', '未配置 Grafana 仪表板'],
    ['en', 'network', 'Grafana dashboard is unavailable. Please try again later'],
    ['cn', 'network', 'Grafana 仪表板暂不可用，请稍后重试']
  ])('localizes the initial dashboard %s %s error', async (locale, failure, expected) => {
    const api = vi.fn()
    if (failure === 'missing') api.mockResolvedValue({ data: {} })
    else api.mockRejectedValue(new Error('Unavailable'))
    const grafana = reactive({ api, params: {}, showIframe: false, error: '' })
    const wrapper = mount(GrafanaPage, {
      global: {
        provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
        plugins: [testI18n(locale)],
        stubs
      }
    })
    await flushPromises()
    expect(grafana.error).toBe(expected)
    expect(showError).toHaveBeenCalledTimes(1)
    expect(showError).toHaveBeenCalledWith(expected)
    wrapper.unmount()
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
        plugins: [testI18n()],
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
        plugins: [testI18n()],
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
        plugins: [testI18n()],
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
    const initialApi = vi.fn(
      () =>
        new Promise((resolve) => {
          finishInitial = resolve
        })
    )
    const grafana = reactive({ api: initialApi, params: {}, showIframe: false, url: '', error: '' })
    const wrapper = mount(
      {
        components: { TraceLogSearch, GrafanaPage },
        template: '<div><TraceLogSearch /><GrafanaPage /></div>'
      },
      {
        global: {
          provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
          plugins: [testI18n()],
          stubs
        }
      }
    )

    expect(initialApi).toHaveBeenCalledOnce()
    await wrapper.get('input').setValue(traceId)
    await wrapper.get('button').trigger('click')
    await flushPromises()
    const searchedUrl = grafana.url

    finishInitial({
      data: { fullURL: 'http://grafana/d/application-logs?var-application=shop-user' }
    })
    await flushPromises()

    expect(grafana.url).toBe(searchedUrl)
    wrapper.unmount()
  })

  it('does not let a late initial-load failure hide a completed manual search', async () => {
    let failInitial!: (reason: unknown) => void
    const initialApi = vi.fn(
      () =>
        new Promise((_resolve, reject) => {
          failInitial = reject
        })
    )
    const grafana = reactive({ api: initialApi, params: {}, showIframe: false, url: '', error: '' })
    const wrapper = mount(
      {
        components: { TraceLogSearch, GrafanaPage },
        template: '<div><TraceLogSearch /><GrafanaPage /></div>'
      },
      {
        global: {
          provide: { [PROVIDE_INJECT_KEY.GRAFANA]: grafana },
          plugins: [testI18n()],
          stubs
        }
      }
    )

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
