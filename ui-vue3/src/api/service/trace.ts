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
