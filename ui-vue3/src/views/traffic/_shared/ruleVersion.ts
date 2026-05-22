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

import { h } from 'vue'
import { Button, notification } from 'ant-design-vue'
import {
  listRuleVersionsAPI,
  type RuleVersion,
  type TrafficRuleKind,
  type VersionConflictError
} from '@/api/service/traffic'
import { HTTP_STATUS } from '@/base/http/constants'

export interface CurrentVersionState {
  id?: number
  versionNo?: number
}

export const currentVersionStateFromItems = (items: RuleVersion[]): CurrentVersionState => {
  const current = items.find((item) => item.isCurrent) || items[0]
  return {
    id: current?.id,
    versionNo: current?.versionNo
  }
}

export const fetchCurrentVersionState = async (
  kind: TrafficRuleKind,
  ruleName: string
): Promise<CurrentVersionState> => {
  try {
    const res = await listRuleVersionsAPI(kind, ruleName)
    if (res.code === HTTP_STATUS.SUCCESS) {
      return currentVersionStateFromItems(res.data?.items || [])
    }
  } catch (e: any) {
    if (e?.code !== 'FEATURE_DISABLED') {
      throw e
    }
  }
  return {}
}

export const isVersionConflict = (e: any): e is VersionConflictError => {
  return e?.code === 'VERSION_CONFLICT'
}

export const notifyVersionConflict = (
  e: any,
  options?: { reload?: () => void | Promise<void> }
): boolean => {
  if (isVersionConflict(e)) {
    notification.warning({
      key: 'rule-version-conflict',
      duration: 0,
      message: '版本冲突',
      description: '规则已被其他操作更新，请重新加载当前版本后再提交。',
      btn: options?.reload
        ? () =>
            h(
              Button,
              {
                type: 'link',
                size: 'small',
                onClick: () => {
                  notification.close('rule-version-conflict')
                  options.reload?.()
                }
              },
              { default: () => 'Reload' }
            )
        : undefined
    })
    return true
  }
  return false
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
