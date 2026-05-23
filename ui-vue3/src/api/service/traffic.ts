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

export type TrafficRuleKind = 'condition-rule' | 'tag-rule' | 'configurator'

export interface RuleVersion {
  id: number
  ruleKind: string
  mesh: string
  resourceKey: string
  ruleName: string
  versionNo: number
  contentHash: string
  specJson: string
  source: 'ADMIN' | 'UPSTREAM' | 'ROLLBACK' | 'BOOTSTRAP' | string
  operation: 'CREATE' | 'UPDATE' | 'DELETE' | string
  author: string
  reason?: string
  rolledBackFromId?: number
  createdAt: string
  isCurrent: boolean
}

export interface RuleVersionList {
  items: RuleVersion[]
  total: number
}

export interface RuleVersionDiffSide {
  id: number
  versionNo: number
  specJson: string
}

export interface RuleVersionDiff {
  left: RuleVersionDiffSide
  right: RuleVersionDiffSide
}

export interface RuleMutationOptions {
  expectedVersionId?: number
}

export interface RuleRollbackRequest extends RuleMutationOptions {
  reason: string
}

export interface VersionConflictError {
  code: 'VERSION_CONFLICT' | 'VERSION_LEDGER_PENDING'
  message: string
  currentVersionId?: number | null
}

const ruleNameForPath = (kind: TrafficRuleKind, ruleName: string): string => {
  return kind === 'configurator' ? encodeURIComponent(ruleName) : ruleName
}

const withExpectedVersion = (options?: RuleMutationOptions) => {
  return options?.expectedVersionId ? { expectedVersionId: options.expectedVersionId } : undefined
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
  versionId: number
): Promise<{ code: string; data: RuleVersion }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionId}`,
    method: 'get'
  })
}

export const diffRuleVersionAPI = (
  kind: TrafficRuleKind,
  ruleName: string,
  versionId: number,
  against = 'current'
): Promise<{ code: string; data: RuleVersionDiff }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionId}/diff`,
    method: 'get',
    params: { against }
  })
}

export const rollbackRuleVersionAPI = (
  kind: TrafficRuleKind,
  ruleName: string,
  versionId: number,
  data: RuleRollbackRequest
): Promise<{ code: string; data: RuleVersion }> => {
  return request({
    url: `/${kind}/${ruleNameForPath(kind, ruleName)}/versions/${versionId}/rollback`,
    method: 'post',
    data
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
export const deleteConditionRuleAPI = (
  ruleName: string,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'delete',
    params: withExpectedVersion(options)
  })
}

// update condition routing.
export const updateConditionRuleAPI = (
  ruleName: string,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'put',
    data,
    params: withExpectedVersion(options)
  })
}

// add condition routing.
export const addConditionRuleAPI = (
  ruleName: string,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/condition-rule/${ruleName}`,
    method: 'post',
    data,
    params: withExpectedVersion(options)
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
export const deleteTagRuleAPI = (ruleName: string, options?: RuleMutationOptions): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'delete',
    params: withExpectedVersion(options)
  })
}

// Get tag routing details.
export const getTagRuleDetailAPI = (ruleName: string): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'get'
  })
}

export const updateTagRuleAPI = (
  ruleName: string,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'put',
    data,
    params: withExpectedVersion(options)
  })
}

export const addTagRuleAPI = (
  ruleName: string,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/tag-rule/${ruleName}`,
    method: 'post',
    data,
    params: withExpectedVersion(options)
  })
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
export const saveConfiguratorDetail = (
  params: any,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'put',
    data,
    params: withExpectedVersion(options)
  })
}
export const addConfiguratorDetail = (
  params: any,
  data: any,
  options?: RuleMutationOptions
): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'post',
    data,
    params: withExpectedVersion(options)
  })
}
export const delConfiguratorDetail = (params: any, options?: RuleMutationOptions): Promise<any> => {
  return request({
    url: `/configurator/${encodeURIComponent(params.name)}`,
    method: 'delete',
    params: withExpectedVersion(options)
  })
}

// TODO Perform front-end and back-end joint debugging
