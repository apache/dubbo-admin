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

import request from '@/base/http/request'

export type TrafficRuleKind =
  | 'condition-rule'
  | 'tag-rule'
  | 'configurator'
  | 'affinity-rule'
  | 'script-rule'

export type RouterRuleKind = 'affinity-rule' | 'script-rule'

export interface RuleVersion {
  ruleKind: string
  mesh: string
  resourceKey: string
  ruleName: string
  versionNo: number
  contentHash: string
  specJson: string
  source: 'ADMIN' | 'UPSTREAM' | 'BOOTSTRAP' | 'ROLLBACK' | string
  operation: 'CREATE' | 'UPDATE' | 'DELETE' | string
  author: string
  reason?: string
  rolledBackFromVersionNo?: number
  createdAt: string
  recordedAt?: string
  isLatestRecorded: boolean
}

export interface RuleVersionList {
  items: RuleVersion[]
  total: number
  latestRecordedVersionNo?: number
  latestRecordedDeleted?: boolean
}

export interface RuleVersionDiffSide {
  versionNo: number
  specJson: string
}

export interface RuleVersionDiff {
  left: RuleVersionDiffSide
  right: RuleVersionDiffSide
}

export interface RollbackRuleVersionResult {
  rolledBackFromVersionNo: number
  versionNo: number
  source: 'ROLLBACK' | string
}

const ruleNameForPath = (kind: TrafficRuleKind, ruleName: string): string => {
  return ['configurator', 'affinity-rule', 'script-rule'].includes(kind)
    ? encodeURIComponent(ruleName)
    : ruleName
}

export const listRuleVersionsAPI = (
  kind: TrafficRuleKind,
  ruleName: string
): Promise<{ code: string; data: RuleVersionList }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions`,
    method: 'get'
  })
}

export const getRuleVersionAPI = (
  kind: TrafficRuleKind,
  ruleName: string,
  versionNo: number
): Promise<{ code: string; data: RuleVersion }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionNo}`,
    method: 'get'
  })
}

export const diffRuleVersionAPI = (
  kind: TrafficRuleKind,
  ruleName: string,
  versionNo: number,
  against: 'current' | 'previous' | number = 'current'
): Promise<{ code: string; data: RuleVersionDiff }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionNo}/diff`,
    method: 'get',
    params: { against }
  })
}

export const rollbackRuleVersionAPI = (
  kind: TrafficRuleKind,
  ruleName: string,
  versionNo: number,
  reason: string
): Promise<{ code: string; data: RollbackRuleVersionResult }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionNo}/rollback`,
    method: 'post',
    data: { reason }
  })
}

export const searchRoutingRule = (params: any): Promise<any> => {
  return request({
    url: '/condition-rule/search',
    method: 'get',
    params
  })
}

// Get condition routing details
export const getConditionRuleDetailAPI = (ruleName: string): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'get'
  })
}

// Delete condition routing.
export const deleteConditionRuleAPI = (ruleName: string): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'delete'
  })
}

// update condition routing.
export const updateConditionRuleAPI = (ruleName: string, data: any): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'put',
    data
  })
}

// add condition routing.
export const addConditionRuleAPI = (ruleName: string, data: any): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'post',
    data
  })
}

export const searchTagRule = (params: any): Promise<any> => {
  return request({
    url: '/tag-rule/search',
    method: 'get',
    params
  })
}

// Delete tag routing.
export const deleteTagRuleAPI = (ruleName: string): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'delete'
  })
}

// Get tag routing details.
export const getTagRuleDetailAPI = (ruleName: string): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'get'
  })
}

export const updateTagRuleAPI = (ruleName: string, data: any): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'put',
    data
  })
}

export const addTagRuleAPI = (ruleName: string, data: any): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'post',
    data
  })
}

export const searchRouterRuleAPI = (kind: RouterRuleKind, params: any): Promise<any> => {
  return request({ url: `/${kind}/search`, method: 'get', params })
}

export const getRouterRuleAPI = (kind: RouterRuleKind, ruleName: string): Promise<any> => {
  return request({ url: `/${kind}/${encodeURIComponent(ruleName)}`, method: 'get' })
}

export const addRouterRuleAPI = (
  kind: RouterRuleKind,
  ruleName: string,
  data: any
): Promise<any> => {
  return request({ url: `/${kind}/${encodeURIComponent(ruleName)}`, method: 'post', data })
}

export const updateRouterRuleAPI = (
  kind: RouterRuleKind,
  ruleName: string,
  data: any
): Promise<any> => {
  return request({ url: `/${kind}/${encodeURIComponent(ruleName)}`, method: 'put', data })
}

export const deleteRouterRuleAPI = (kind: RouterRuleKind, ruleName: string): Promise<any> => {
  return request({ url: `/${kind}/${encodeURIComponent(ruleName)}`, method: 'delete' })
}

export const searchDynamicConfig = (params: any): Promise<any> => {
  return request({
    url: '/configurator/search',
    method: 'get',
    params
  })
}

export const searchVirtualService = (params: any): Promise<any> => {
  return request({
    url: '/virtualService/search',
    method: 'get',
    params
  })
}

export const searchDestinationRule = (params: any): Promise<any> => {
  return request({
    url: '/configurator/search',
    method: 'get',
    params
  })
}

export const getConfiguratorDetail = (params: any): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'get'
  })
}
export const saveConfiguratorDetail = (params: any, data: any): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'put',
    data
  })
}
export const addConfiguratorDetail = (params: any, data: any): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'post',
    data
  })
}
export const delConfiguratorDetail = (params: any): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'delete'
  })
}
