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
export default {
  service: '服务',
  serviceSearch: '服务查询',
  serviceGovernance: '服务治理',
  routingRule: '条件路由',
  tagRule: '标签路由',
  dynamicConfig: '动态配置',
  accessControl: '黑白名单',
  weightAdjust: '权重调整',
  loadBalance: '负载均衡',
  serviceTest: '服务测试',
  serviceMock: '服务Mock',
  providers: '提供者',
  consumers: '消费者',
  metrics: '统计',
  group: '组',
  version: '版本',
  app: '应用',
  ip: 'IP地址',
  serviceInfo: '服务信息',
  port: '端口',
  timeout: '超时(毫秒)',
  serialization: '序列化',
  appName: '应用名',
  serviceName: '服务名',
  operation: '操作',
  searchResult: '查询结果',
  search: '搜索',
  methodName: '方法名',
  enabled: '开启',
  disabled: '禁用',
  method: '方法',
  weight: '权重',
  create: '创建',
  save: '保存',
  cancel: '取消',
  close: '关闭',
  confirm: '确认',
  ruleContent: '规则内容',
  createNewRoutingRule: '创建新路由规则',
  createNewTagRule: '创建新标签规则',
  createNewDynamicConfigRule: '创建新动态配置规则',
  createNewWeightRule: '新建权重规则',
  createNewLoadBalanceRule: '新建负载均衡规则',
  serviceIdHint: '服务名',
  view: '查看',
  edit: '编辑',
  delete: '删除',
  searchRoutingRule: '搜索路由规则',
  searchAccessRule: '搜索黑白名单',
  searchWeightRule: '搜索权重调整规则',
  dataIdHint: '组/服务：版本形式的服务ID，组和版本可选',
  agree: '同意',
  disagree: '不同意',
  searchDynamicConfig: '搜索动态配置',
  appNameHint: '服务所属的应用名称',
  basicInfo: '基础信息',
  metaData: '元数据',
  searchDubboService: '搜索Dubbo服务或应用',
  serviceSearchHint: '服务ID, org.apache.dubbo.demo.api.DemoService, * 代表所有服务',
  ipSearchHint: '在指定的IP地址上查找目标服务器提供的所有服务',
  appSearchHint: '输入应用名称以查找由一个特定应用提供的所有服务, * 代表所有',
  searchTagRule: '根据应用名搜索标签规则',
  searchBalanceRule: '搜索负载均衡规则',
  parameterList: '参数列表',
  returnType: '返回值',
  noMetadataHint: '无元数据信息，请升级至Dubbo2.7及以上版本，或者查看application.properties中关于config center的配置，详见',
  here: '这里',
  configAddress: 'https://github.com/apache/incubator-dubbo-admin/wiki/Dubbo-Admin%E9%85%8D%E7%BD%AE%E8%AF%B4%E6%98%8E',
  whiteList: '白名单',
  whiteListHint: '白名单IP列表, 多个地址用逗号分隔: 1.1.1.1,2.2.2.2',
  blackList: '黑名单',
  blackListHint: '黑名单IP列表, 多个地址用逗号分隔: 3.3.3.3,4.4.4.4',
  address: '地址列表',
  weightAddressHint: '此权重设置的IP地址,用逗号分隔: 1.1.1.1,2.2.2.2',
  weightHint: '权重值，默认100',
  methodHint: '负载均衡生效的方法，*代表所有方法',
  strategy: '策略',
  balanceStrategyHint: '负载均衡策略',
  goIndex: '返回首页',
  releaseLater: '在后续版本中发布，敬请期待',
  later: {
    metrics: 'Metrics会在后续版本中发布，敬请期待',
    serviceTest: '服务测试会在后续版本中发布，敬请期待',
    serviceMock: '服务Mock会在后续版本中发布，敬请期待'
  },
  by: '按',
  $vuetify: {
    dataIterator: {
      rowsPerPageText: '每页记录数：',
      rowsPerPageAll: '全部',
      pageText: '{0}-{1} 共 {2} 条',
      noResultsText: '没有找到匹配记录',
      nextPage: '下一页',
      prevPage: '上一页'
    },
    dataTable: {
      rowsPerPageText: '每页行数：'
    },
    noDataText: '无可用数据'
  },
  configManage: '配置管理',
  configCenterAddress: '配置中心地址',
  searchDubboConfig: '搜索Dubbo配置',
  createNewDubboConfig: '新建Dubbo配置',
  scope: '范围',
  name: '名称',
  warnDeleteConfig: ' 是否要删除Dubbo配置: ',
  configNameHint: '配置所属的应用名, global 表示全局配置',
  configContent: '配置内容',
  testMethod: '测试方法',
  execute: '执行',
  result: '结果: ',
  success: ' 成功',
  fail: '失败',
  detail: '详情',
  more: '更多',
  copyUrl: '复制 URL',
  copy: '复制',
  url: 'URL',
  copySuccessfully: '已复制',
  test: '测试',
  placeholders: {
    searchService: '通过服务名搜索服务'
  },
  methods: '方法列表',
  testModule: {
    searchServiceHint: '服务ID, org.apache.dubbo.demo.api.DemoService, 使用 * 进行模糊查找, 按回车键查询'
  }
}
