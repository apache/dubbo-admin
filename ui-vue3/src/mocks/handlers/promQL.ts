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

export const promQLHandlers: HttpHandler[] = [
  http.get(`${base}/promQL/query`, () =>
    success({
      status: 'success',
      data: {
        resultType: 'vector',
        result: [
          {
            metric: {
              __name__: 'dubbo_requests_total',
              service: 'org.apache.dubbo.samples.UserService'
            },
            value: [1710644821.532, '1234']
          }
        ]
      }
    })
  )
]
