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

export interface ApiResponse<T = any> {
  code: string
  message: string
  data: T
}

export interface PageInfo {
  Total: number
  NextOffset: string
}

export interface PaginatedData<T = any> {
  pageInfo: PageInfo
  list: T[]
}

export interface VersionGroup {
  version: string | null
  group: string | null
}

export interface ServiceSearchItem {
  serviceName: string
  versionGroups: VersionGroup[]
  avgQPS: number
  avgRT: string
  requestTotal: number
}

export interface ServiceDetail {
  serviceName: string
  versionGroup: string[]
  protocol: string
  delay: string
  timeOut: string
  retry: number
  requestTotal: number
  avgRT: string
  avgQPS: number
  obsolete: boolean
}

export interface ServiceDistributionItem {
  applicationName: string
  instanceNum: number
  instanceName: string
  rpcPort: string
  timeout: string
  retryNum: string
  label: string
}

export interface ApplicationSearchItem {
  appName: string
  deployClusters: string[]
  instanceCount: number
  registryClusters: string[]
}

export interface ApplicationDetail {
  appName: string
  appTypes: string[]
  deployClusters: string[]
  dubboPorts: number[]
  dubboVersions: string[]
  images: string[]
  registerClusters: string[]
  registerModes: string[]
  rpcProtocols: string[]
  serialProtocols: string[]
  workloads: string[]
}

export interface ApplicationInstanceStatistics {
  instanceTotal: number
  versionTotal: number
  cpuTotal: string
  memoryTotal: string
}

export interface ApplicationInstanceInfoItem {
  ip: string
  name: string
  deployState: string
  deployCluster: string
  registerState: string
  registerClusters: string[]
  cpu: string
  memory: string
  startTime: string
  registerTime: string
  labels: Record<string, string>
}

export interface ApplicationEventItem {
  desc: string
  time: string
  type: string
}

export interface InstanceSearchItem {
  ip: string
  name: string
  deployState: string
  deployCluster: string
  registerState: string
  registerClusters: string[]
  cpu: string
  memory: string
  startTime_k8s: string
  registerTime: string
  labels: Record<string, string>
}

export interface InstanceDetail {
  deployState: string
  registerStates: string
  ip: string
  rpcPort: string
  appName: string
  workloadName: string
  labels: Record<string, string>
  createTime: string
  readyTime: string
  registerTime: string
  registerClusters: string[]
  deployCluster: string
  node: string
  image: string
  probes: {
    startupProbe: { type: string; open: boolean }
    readinessProbe: { type: string; open: boolean }
    livenessProbe: { type: string; open: boolean }
  }
}

export interface ClusterMetrics {
  all: number
  application: number
  consumers: number
  providers: number
  services: number
}

export interface MeshItem {
  id: string
  name: string
  type: string
  version: string
  status: string
}

export interface VersionInfo {
  gitVersion: string
  gitCommit: string
  gitTreeState: string
  buildDate: string
  goVersion: string
  compiler: string
  platform: string
}

export interface ConfiguratorRule {
  ruleName: string
  ruleGranularity: boolean
  enable: boolean
  createTime: string
}

export interface ConfiguratorDetail {
  name: string
  configs: {
    side: string
    timeout: number
    retries: number
    loadbalance: string
  }[]
}

export interface RoutingRule {
  ruleName: string
  ruleGranularity: boolean
  enable: boolean
  createTime: string
}

export interface RoutingRuleDetail {
  name: string
  serviceName: string
  enable: boolean
  priority: number
  conditions: string[]
  force: boolean
  enabled: boolean
}

export interface TagRule {
  ruleName: string
  enable: boolean
  createTime: string
}

export interface TagRuleDetail {
  name: string
  serviceName: string
  enable: boolean
  tags: {
    name: string
    addresses: string[]
  }[]
}

export interface DestinationRuleItem {
  ruleName: string
  createTime: string
}

export interface VirtualServiceItem {
  ruleName: string
  createTime: string
  lastModifiedTime: string
}

export interface MetadataConfig {
  registry: string
  metadata: string
  config: string
  prometheus: string
  grafana: string
  tracing: string
}

export interface OverviewData {
  appCount: number
  serviceCount: number
  insCount: number
  protocols: Record<string, number>
  releases: Record<string, number>
  discoveries: Record<string, number>
}

export interface ProviderInstance {
  name: string
  appName: string
  ip: string
}

export interface ServiceMethod {
  methodName: string
  parameterTypes: string[]
  signature: string
}

export interface ServiceMethodDetail {
  methodName: string
  signature: string
  parameterTypes: string[]
  parameters: { name: string; type: string }[]
  returnType: string
  types: {
    type: string
    properties: Record<string, string>
    items: any[]
    enums: any[]
  }[]
}

export interface ArgumentRouteConfig {
  args: {
    type: string
    key: string
    operator: string
    value: string
    serviceName: string
  }[]
}

export interface FlowWeightItem {
  version: string
  weight: number
}

export interface GrayItem {
  tag: string
  weight: number
}
