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
import DevToolUtil from '@/utils/DevToolUtil'

Mock.mock(DevToolUtil.mockUrl('/mock/tag-rule/search'), 'get', () => {
  const total = Mock.mock('@integer(8, 1000)')
  const list = []
  for (let i = 0; i < total; i++) {
    list.push({
      ruleName: 'app_' + Mock.mock('@string(2,10)'),
      enable: Mock.mock('@boolean'),
      createTime: Mock.mock('@datetime')
    })
  }
  return {
    code: 'Success',
    msg: 'success',
    data: {
      pageInfo: {
        Total: total,
        NextOffset: '0'
      },
      list
    }
  }
})

// 标签路由详情
Mock.mock(DevToolUtil.mockUrl('/mock/tag-rule/'), 'get', (options: any) => {
  const url = options.url
  const ruleName = url.split('/').pop()
  return {
    code: 'Success',
    message: 'success',
    data: {
      name: ruleName,
      serviceName: 'org.apache.dubbo.samples.UserService',
      enable: true,
      tags: [
        {
          name: 'v1',
          addresses: ['192.168.1.1:20880', '192.168.1.2:20880']
        },
        {
          name: 'v2',
          addresses: ['192.168.1.3:20880']
        }
      ]
    }
  }
})

// 删除标签路由
Mock.mock(DevToolUtil.mockUrl('/mock/tag-rule/'), 'delete', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 更新标签路由
Mock.mock(DevToolUtil.mockUrl('/mock/tag-rule/'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 新增标签路由
Mock.mock(DevToolUtil.mockUrl('/mock/tag-rule/'), 'post', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})
