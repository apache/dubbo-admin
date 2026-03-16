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

Mock.mock(devTool.mockUrl('/mock/service/search'), 'get', {
  code: 'Success',
  msg: 'success',
  data: {
    pageInfo: {
      Total: 8,
      NextOffset: '0'
    },
    list: [
      {
        serviceName: 'org.apache.dubbo.samples.UserService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 6,
        avgRT: '194ms',
        requestTotal: 200
      },
      {
        serviceName: 'org.apache.dubbo.samples.OrderService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 13,
        avgRT: '189ms',
        requestTotal: 164
      },
      {
        serviceName: 'org.apache.dubbo.samples.DetailService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 0.5,
        avgRT: '268ms',
        requestTotal: 1324
      },
      {
        serviceName: 'org.apache.dubbo.samples.PayService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 9,
        avgRT: '346ms',
        requestTotal: 189
      },
      {
        serviceName: 'org.apache.dubbo.samples.CommentService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 8,
        avgRT: '936ms',
        requestTotal: 200
      },
      {
        serviceName: 'org.apache.dubbo.samples.RepayService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 17,
        avgRT: '240ms',
        requestTotal: 146
      },
      {
        serviceName: 'org.apche.dubbo.samples.TransportService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 43,
        avgRT: '89ms',
        requestTotal: 367
      },
      {
        serviceName: 'org.apche.dubbo.samples.DistributionService',
        versionGroups: [
          {
            version: '1.0.0',
            group: 'group1'
          },
          {
            version: '1.0.0',
            group: null
          },
          {
            version: null,
            group: 'group1'
          },
          {
            version: null,
            group: null
          }
        ],
        avgQPS: 4,
        avgRT: '78ms',
        requestTotal: 145
      }
    ]
  }
})

Mock.mock(devTool.mockUrl('/mock/service/distribution'), 'get', () => {
  return {
    code: 'Success',
    msg: 'success',
    data: {
      pageInfo: {
        Total: 8,
        NextOffset: '0'
      },
      list: []
    }
  }
})

// 服务监控和追踪
Mock.mock(devTool.mockUrl('/mock/service/metric-dashboard'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: 'http://8.147.104.101:3000/d/a0b114ca-edf7-4dfe-ac2c-34a4fc545fed/service?orgId=1&refresh=1m'
  }
})

Mock.mock(devTool.mockUrl('/mock/service/trace-dashboard'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: 'http://8.147.104.101:3000/d/e968a89b-f03d-42e3-8ad3-930ae815cb0f/service?orgId=1&refresh=1m'
  }
})

// 服务超时配置
Mock.mock(devTool.mockUrl('/mock/service/config/timeout'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: {
      timeout: 3000
    }
  }
})

Mock.mock(devTool.mockUrl('/mock/service/config/timeout'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 服务重试配置
Mock.mock(devTool.mockUrl('/mock/service/config/retry'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: {
      retry: 3
    }
  }
})

Mock.mock(devTool.mockUrl('/mock/service/config/retry'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 区域优先配置
Mock.mock(devTool.mockUrl('/mock/service/config/regionPriority'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: {
      enable: true
    }
  }
})

Mock.mock(devTool.mockUrl('/mock/service/config/regionPriority'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})

// 参数路由
Mock.mock(devTool.mockUrl('/mock/service/config/argumentRoute'), 'get', () => {
  return {
    code: 'Success',
    message: 'success',
    data: {
      args: [
        {
          type: 'header',
          key: 'X-User-Id',
          operator: '=',
          value: '123',
          serviceName: 'org.apache.dubbo.samples.UserService'
        }
      ]
    }
  }
})

Mock.mock(devTool.mockUrl('/mock/service/config/argumentRoute'), 'put', () => {
  return {
    code: 'Success',
    message: 'success',
    data: null
  }
})
