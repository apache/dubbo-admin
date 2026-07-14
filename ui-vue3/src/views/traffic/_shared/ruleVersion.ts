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

import {
  listRuleVersionsAPI,
  type RuleVersion,
  type RuleVersionList,
  type TrafficRuleKind
} from '@/api/service/traffic'
import { HTTP_STATUS } from '@/base/http/constants'

export interface LatestRecordedState {
  versionNo?: number
  latestRecordedDeleted: boolean
}

export const latestRecordedStateFromItems = (items: RuleVersion[]): LatestRecordedState => {
  const latestRecorded = items.find((item) => item.isLatestRecorded)
  const head = items[0]
  return {
    versionNo: latestRecorded?.versionNo,
    latestRecordedDeleted: Boolean(head?.operation === 'DELETE')
  }
}

export const latestRecordedStateFromList = (list?: RuleVersionList): LatestRecordedState => {
  if (!list) {
    return { latestRecordedDeleted: false }
  }
  if (list.latestRecordedVersionNo !== undefined || list.latestRecordedDeleted !== undefined) {
    return {
      versionNo: list.latestRecordedVersionNo,
      latestRecordedDeleted: Boolean(list.latestRecordedDeleted)
    }
  }
  return latestRecordedStateFromItems(list.items || [])
}

export const versionDiffLabel = (prefix: string, versionNo?: number): string =>
  typeof versionNo === 'number' && versionNo > 0 ? `${prefix} v${versionNo}` : prefix

export const isLatestRecordedHistoryRequest = (
  requestSeq: number,
  latestSeq: number,
  disposed: boolean
) => {
  return !disposed && requestSeq === latestSeq
}

export const fetchLatestRecordedState = async (
  kind: TrafficRuleKind,
  ruleName: string
): Promise<LatestRecordedState> => {
  const res = await listRuleVersionsAPI(kind, ruleName)
  if (res.code === HTTP_STATUS.SUCCESS) {
    return latestRecordedStateFromList(res.data)
  }
  return { latestRecordedDeleted: false }
}

export const formatRuleSpec = (specJson?: string): string => {
  if (!specJson) {
    return ''
  }
  try {
    return JSON.stringify(JSON.parse(specJson), null, 2)
  } catch (e) {
    return specJson
  }
}
