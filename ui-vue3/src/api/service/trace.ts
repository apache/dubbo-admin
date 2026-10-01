import request from '@/base/http/request'

export const getTraceStartTime = async (traceId: string): Promise<any> => {
  try {
    return await request({
      url: '/trace/start-time',
      method: 'get',
      params: { traceId },
      silentError: true
    } as any)
  } catch (error) {
    // The shared HTTP interceptor rejects undefined when no response exists.
    if (error === undefined) {
      throw Object.assign(new Error('Admin request unavailable'), { code: 'NetWorkError' })
    }
    throw error
  }
}

export const getTraceLogDashboard = (params: {
  traceId: string
  from: number
  to: number
}): Promise<any> => {
  return request({
    url: '/trace/log-dashboard',
    method: 'get',
    params,
    silentError: true
  } as any)
}
