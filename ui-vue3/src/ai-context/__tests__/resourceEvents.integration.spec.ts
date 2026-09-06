import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it, vi } from 'vitest'
import ApplicationEvents from '@/views/resources/applications/tabs/event.vue'
import EventTimeline from '@/components/EventTimeline.vue'
import { useMeshStore } from '@/stores/mesh'
import type { AIContextProvider } from '../types'

const mocks = vi.hoisted(() => ({
  register: vi.fn(),
  listApplicationEvent: vi.fn()
}))

vi.mock('../instance', () => ({
  aiContextManager: { register: mocks.register }
}))

vi.mock('@/api/service/app', () => ({
  listApplicationEvent: mocks.listApplicationEvent
}))

describe('application lifecycle events AI context', () => {
  it('keeps paginated events visible to the provider and unregisters on exit', async () => {
    const unregister = vi.fn()
    mocks.register.mockReturnValue(unregister)
    const firstPage = Array.from({ length: 20 }, (_, index) => ({
      type: 'normal',
      message: `Instance registered ${index}`,
      source: 'nacos',
      time: '2026-09-06T08:00:00Z'
    }))
    const lastPage = [
      {
        type: 'warning',
        message: 'Instance deregistered',
        source: 'zookeeper',
        time: '2026-09-06T07:00:00Z'
      }
    ]
    mocks.listApplicationEvent
      .mockResolvedValueOnce({ data: { list: firstPage, total: 21 } })
      .mockResolvedValueOnce({ data: { list: lastPage, total: 21 } })
    const pinia = createPinia()
    useMeshStore(pinia).mesh = 'production'
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/applications/:pathId/event', component: ApplicationEvents }]
    })
    await router.push('/applications/shop-user/event')
    await router.isReady()

    const wrapper = shallowMount(ApplicationEvents, { global: { plugins: [pinia, router] } })
    await flushPromises()

    expect(mocks.listApplicationEvent).toHaveBeenNthCalledWith(1, {
      appName: 'shop-user',
      mesh: 'production',
      pageOffset: 0,
      pageSize: 20
    })
    const timeline = wrapper.getComponent(EventTimeline)
    expect(timeline.props('events')).toEqual(firstPage)
    expect(timeline.props('hasMore')).toBe(true)
    timeline.vm.$emit('loadMore')
    await flushPromises()

    expect(mocks.listApplicationEvent).toHaveBeenNthCalledWith(2, {
      appName: 'shop-user',
      mesh: 'production',
      pageOffset: 20,
      pageSize: 20
    })
    expect(timeline.props('events')).toEqual([...firstPage, ...lastPage])
    expect(timeline.props('hasMore')).toBe(false)
    const provider = mocks.register.mock.calls[0][0] as AIContextProvider
    expect(await provider.collect()).toMatchObject({
      evidence: {
        id: 'event-list',
        data: { total: 21, includedCount: 10, truncated: true }
      }
    })
    expect(mocks.register).toHaveBeenCalledOnce()
    timeline.vm.$emit('loadMore')
    await flushPromises()
    expect(mocks.listApplicationEvent).toHaveBeenCalledTimes(2)

    wrapper.unmount()
    expect(unregister).toHaveBeenCalledOnce()
  })
})
