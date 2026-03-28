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
import type { TagRule, TagRuleDetail, PaginatedData } from '@/types/api'

function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

function randomString(min: number, max: number): string {
  const len = randomInt(min, max)
  return Array.from({ length: len }, () => String.fromCharCode(97 + randomInt(0, 25))).join('')
}

export const tagRuleHandlers: HttpHandler[] = [
  http.get(`${base}/tag-rule/search`, () => {
    const total = randomInt(8, 1000)
    const list: TagRule[] = Array.from({ length: total }, () => ({
      ruleName: 'app_' + randomString(2, 10),
      enable: Math.random() > 0.5,
      createTime: '2024-01-01 00:00:00'
    }))
    return success<PaginatedData<TagRule>>({
      pageInfo: { Total: total, NextOffset: '0' },
      list
    })
  }),

  http.get(`${base}/tag-rule/:ruleName`, ({ params }) => {
    const detail: TagRuleDetail = {
      name: params.ruleName as string,
      serviceName: 'org.apache.dubbo.samples.UserService',
      enable: true,
      tags: [
        { name: 'v1', addresses: ['192.168.1.1:20880', '192.168.1.2:20880'] },
        { name: 'v2', addresses: ['192.168.1.3:20880'] }
      ]
    }
    return success(detail)
  }),

  http.delete(`${base}/tag-rule/:ruleName`, () => success(null)),

  http.put(`${base}/tag-rule/:ruleName`, () => success(null)),

  http.post(`${base}/tag-rule/:ruleName`, () => success(null))
]
