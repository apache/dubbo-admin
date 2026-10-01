import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getTraceStartTime } from './trace'

const { request } = vi.hoisted(() => ({ request: vi.fn() }))
vi.mock('@/base/http/request', () => ({ default: request }))

describe('getTraceStartTime', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('classifies a transport failure with no response as a network error', async () => {
    request.mockRejectedValueOnce(undefined)

    await expect(getTraceStartTime('766b9604d84bd8e6b17b2292dfd9bdd0')).rejects.toMatchObject({
      code: 'NetWorkError'
    })
  })

  it('preserves the backend error code', async () => {
    request.mockRejectedValueOnce({ code: 'NotFoundError' })

    await expect(getTraceStartTime('766b9604d84bd8e6b17b2292dfd9bdd0')).rejects.toMatchObject({
      code: 'NotFoundError'
    })
  })
})
