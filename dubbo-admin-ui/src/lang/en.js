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
  service: 'Service',
  serviceSearch: 'Search service name',
  serviceGovernance: 'Service Governance',
  routingRule: 'Condition Rule',
  tagRule: 'Tag Rule',
  dynamicConfig: 'Dynamic Config',
  accessControl: 'Black White List',
  weightAdjust: 'Weight Adjust',
  loadBalance: 'Load Balance',
  serviceTest: 'Service Test',
  serviceMock: 'Service Mock',
  metrics: 'Metrics',
  group: 'Group',
  serviceInfo: 'Service Info',
  providers: 'Providers',
  consumers: 'Consumers',
  version: 'Version',
  app: 'Application',
  ip: 'IP',
  port: 'PORT',
  timeout: 'timeout(ms)',
  serialization: 'serialization',
  appName: 'Application Name',
  serviceName: 'Service Name',
  operation: 'Operation',
  searchResult: 'Search Result',
  search: 'Search',
  methodName: 'Method Name',
  enabled: 'Enabled',
  disabled: 'Disabled',
  method: 'Method',
  weight: 'Weight',
  create: 'CREATE',
  save: 'SAVE',
  cancel: 'CANCEL',
  close: 'CLOSE',
  confirm: 'CONFIRM',
  ruleContent: 'RULE CONTENT',
  createNewRoutingRule: 'Create New Routing Rule',
  createNewTagRule: 'Create New Tag Rule',
  createNewDynamicConfigRule: 'Create New Dynamic Config Rule',
  createNewWeightRule: 'Create New Weight Rule',
  createNewLoadBalanceRule: 'Create new load balancing rule',
  serviceIdHint: 'Service ID',
  view: 'View',
  edit: 'Edit',
  delete: 'Delete',
  searchRoutingRule: 'Search Routing Rule',
  searchAccess: 'Search Access Rule',
  searchWeightRule: 'Search Weight Adjust Rule',
  dataIdHint: 'A service ID in form of group/service:version, group and version are optional',
  agree: 'Agree',
  disagree: 'Disagree',
  searchDynamicConfig: 'Search Dynamic Config',
  appNameHint: 'Application name the service belongs to',
  basicInfo: 'BasicInfo',
  metaData: 'MetaData',
  searchDubboService: 'Search Dubbo Services or applications',
  serviceSearchHint: 'Service ID, org.apache.dubbo.demo.api.DemoService, * for all services',
  ipSearchHint: 'Find all services provided by the target server on the specified IP address',
  appSearchHint: 'Input an application name to find all services provided by one particular application, * for all',
  searchTagRule: 'Search Tag Rule by application name',
  searchBalanceRule: 'Search Balancing Rule',
  noMetadataHint: 'There is no metadata available, please update to Dubbo2.7, or check your config center configuration in application.properties, please check ',
  parameterList: 'parameterList',
  returnType: 'returnType',
  here: 'here',
  configAddress: 'https://github.com/apache/incubator-dubbo-admin/wiki/Dubbo-Admin-configuration',
  whiteList: 'White List',
  whiteListHint: 'White list IP address, divided by comma: 1.1.1.1,2.2.2.2',
  blackList: 'Black List',
  blackListHint: 'Black list IP address, divided by comma: 3.3.3.3,4.4.4.4',
  address: 'Address',
  weightAddressHint: 'IP addresses to set this weight, divided by comma: 1.1.1.1,2.2.2.2',
  weightHint: 'weight value, default is 100',
  methodHint: 'choose method of load balancing, * for all methods',
  strategy: 'Strategy',
  balanceStrategyHint: 'load balancing strategy',
  goIndex: 'Go To Index',
  releaseLater: 'will release later',
  later: {
    metrics: 'Metrics will release later',
    serviceTest: 'Service Test will release later',
    serviceMock: 'Service Mock will release later'
  },
  by: 'by ',
  $vuetify: {
    dataIterator: {
      rowsPerPageText: 'Items per page:',
      rowsPerPageAll: 'All',
      pageText: '{0}-{1} of {2}',
      noResultsText: 'No matching records found',
      nextPage: 'Next page',
      prevPage: 'Previous page'
    },
    dataTable: {
      rowsPerPageText: 'Rows per page:'
    },
    noDataText: 'No data available'
  },
  configManage: 'Configuration Management',
  configCenterAddress: 'ConfigCenter Address',
  searchDubboConfig: 'Search Dubbo Config',
  createNewDubboConfig: 'Create New Dubbo Config',
  scope: 'Scope',
  name: 'Name',
  warnDeleteConfig: ' Are you sure to Delete Dubbo Config: ',
  configNameHint: "Application name the config belongs to, use 'global'(without quotes) for global config",
  configContent: 'Config Content',
  testMethod: 'Test Method',
  execute: 'EXECUTE',
  result: 'Result: ',
  success: 'SUCCESS',
  fail: 'FAIL',
  detail: 'Detail',
  more: 'More',
  copyUrl: 'Copy URL',
  copy: 'Copy',
  url: 'URL',
  copySuccessfully: 'Copied',
  test: 'Test',
  placeholders: {
    searchService: 'Search by service name'
  },
  methods: 'Methods',
  testModule: {
    searchServiceHint: 'Service ID, org.apache.dubbo.demo.api.DemoService, * for fuzzy search, press Enter to search'
  }
}
