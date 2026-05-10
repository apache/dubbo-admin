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

import { http, type HttpHandler } from 'msw'
import { success, base } from '../utils'
import type { ConfiguratorRule, ConfiguratorDetail, PaginatedData } from '@/types/api'

function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function randomString(min: number, max: number): string {
  const len = randomInt(min, max)
  return Array.from({ length: len }, () => String.fromCharCode(97 + randomInt(0, 25))).join('')
}

export const dynamicConfigHandlers: HttpHandler[] = [
  http.get(`${base}/configurator/search`, () => {
    const total = randomInt(8, 1000)
    const list: ConfiguratorRule[] = Array.from({ length: total }, () => ({
      ruleName: 'app_' + randomString(2, 10),
      ruleGranularity: Math.random() > 0.5,
      enable: Math.random() > 0.5,
      createTime: '2024-01-01 00:00:00'
    }))
    return success<PaginatedData<ConfiguratorRule>>({
      pageInfo: { Total: total, NextOffset: '0' },
      list
    })
  }),

  http.get(`${base}/configurator/:ruleName`, ({ params }) => {
    const detail: ConfiguratorDetail = {
      name: params.ruleName as string,
      configs: [{ side: 'provider', timeout: 3000, retries: 2, loadbalance: 'roundrobin' }]
    }
    return success(detail)
  }),

  http.delete(`${base}/configurator/:ruleName`, () => success(null)),

  http.put(`${base}/configurator/:ruleName`, () => success(null)),

  http.post(`${base}/configurator/:ruleName`, () => success(null))
]
