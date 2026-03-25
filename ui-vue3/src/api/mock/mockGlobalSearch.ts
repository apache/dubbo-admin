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

import Mock from 'mockjs'
import devTool from '@/utils/DevToolUtil'

Mock.mock(devTool.mockUrl('/mock/search'), 'get', {
  code: 'Success',
  message: '成功',
  data: {
    find: true,
    candidates: ['test1', 'test2', 'test3']
  }
})

Mock.mock(devTool.mockUrl('/mock/meshes'), 'get', {
  code: 'Success',
  message: '成功',
  data: [
    {
      name: 'dubbo-mesh',
      type: 'kubernetes',
      version: '3.1.0',
      status: 'Healthy'
    }
  ]
})
