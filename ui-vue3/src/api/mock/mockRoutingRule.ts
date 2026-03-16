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

import devTool from '@/utils/DevToolUtil'
import Mock from 'mockjs'

Mock.mock(devTool.mockUrl('/mock/condition-rule/search'), 'get', () => {
  const total = Mock.mock('@integer(8, 1000)')
  const list = []
  for (let i = 0; i < total; i++) {
    list.push({
      ruleName: 'app_' + Mock.mock('@string(2,10)'),
      ruleGranularity: Mock.mock('@boolean'),
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

// 条件路由详情
Mock.mock(devTool.mockUrl('/mock/condition-rule/'), 'get', (options: any) => {
  const url = options.url
  const ruleName = url.split('/').pop()
  return {
    code: 'Success',
    message: 'success',
    data: {
      name: ruleName,
      serviceName: 'org.apache.dubbo.samples.UserService',
      enable: true,
      priority: 1,
      conditions: [
        'method != "find*" => address = 192.168.1.1:20880',
        'method = "sayHello" => address = 192.168.1.2:20880'
      ],
      force: false,
      enabled: true
    }
  }
})

// 删除条件路由
Mock.mock(devTool.mockUrl('/mock/condition-rule/'), 'delete', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 更新条件路由
Mock.mock(devTool.mockUrl('/mock/condition-rule/'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 新增条件路由
Mock.mock(devTool.mockUrl('/mock/condition-rule/'), 'post', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})
