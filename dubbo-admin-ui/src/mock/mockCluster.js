
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

// 1、引入mockjs
const Mock = require('mockjs')

// 2、获取 mock.Random 对象
const random = Mock.Random
console.log(random) // 简单使用就不操作了，需要操作的可以去看文档

// 3、基本用法 Mock.mock(url, type, data) // 参数文档 https://github.com/nuysoft/Mock/wiki
Mock.mock('/mock/metrics/cluster', 'get', {
  code: 200,
  message: '成功',
  data: {
    all:0,
    application:0,
    consumers:0,
    providers:0,
    services:0
  }
})
