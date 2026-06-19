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
import { Button, Input, Modal, notification, Space } from 'ant-design-vue'
import {
  abandonRuleVersionIntentAPI,
  listRuleVersionsAPI,
  repairRuleVersionIntentAPI,
  type RuleVersion,
  type RuleVersionList,
  type TrafficRuleKind,
  type VersionConflictError,
  type VersionLedgerPendingError
} from '@/api/service/traffic'
import { HTTP_STATUS } from '@/base/http/constants'
import { i18n } from '@/base/i18n'

export interface CurrentVersionState {
  id?: string
  versionNo?: number
  deleted: boolean
}

export const currentVersionStateFromItems = (items: RuleVersion[]): CurrentVersionState => {
  const current = items.find((item) => item.isCurrent)
  const head = items[0]
  return {
    id: current?.id,
    versionNo: current?.versionNo,
    deleted: !current && head?.operation === 'DELETE'
  }
}

export const currentVersionStateFromList = (list?: RuleVersionList): CurrentVersionState => {
  if (!list) {
    return { deleted: false }
  }
  if (list.currentVersionId !== undefined || list.deleted !== undefined) {
    return {
      id: list.currentVersionId,
      versionNo: list.currentVersionNo,
      deleted: Boolean(list.deleted)
    }
  }
  return currentVersionStateFromItems(list.items || [])
}

export const rollbackExpectedVersionId = (state: CurrentVersionState): string | undefined => {
  if (state.id !== undefined) {
    return state.id
  }
  return state.deleted ? '0' : undefined
}

export const versionDiffLabel = (prefix: string, versionNo?: number): string =>
  typeof versionNo === 'number' ? `${prefix} v${versionNo}` : prefix

export const normalizeIntentReason = (reason: string): string => reason.trim()

export const isCurrentHistoryRequest = (
  requestSeq: number,
  latestSeq: number,
  disposed: boolean
) => {
  return !disposed && requestSeq === latestSeq
}

export const fetchCurrentVersionState = async (
  kind: TrafficRuleKind,
  ruleName: string
): Promise<CurrentVersionState> => {
  try {
    const res = await listRuleVersionsAPI(kind, ruleName)
    if (res.code === HTTP_STATUS.SUCCESS) {
      return currentVersionStateFromList(res.data)
    }
  } catch (e: any) {
    if (e?.code !== 'FEATURE_DISABLED') {
      throw e
    }
  }
  return { deleted: false }
}

export const isVersionConflict = (e: any): e is VersionConflictError => {
  return e?.code === 'VERSION_CONFLICT'
}

export const isVersionLedgerPending = (e: any): e is VersionLedgerPendingError => {
  return e?.code === 'VERSION_LEDGER_PENDING'
}

export const isFeatureDisabled = (e: any): boolean => {
  return e?.code === 'FEATURE_DISABLED'
}

const t = (key: string, params?: Record<string, unknown>) => i18n.global.t(key, params)

export const ruleVersionErrorMessage = (e: any): string => e?.message || String(e)

const repairingIntentIds = new Set<string>()

const openAbandonReasonModal = (
  intentId: string,
  options?: { reload?: () => void | Promise<void> }
) => {
  let reason = ''
  let submitting = false
  const modal = Modal.confirm({
    title: t('ruleVersionDomain.abandonIntentTitle'),
    content: () =>
      h(Input.TextArea, {
        rows: 3,
        maxlength: 1024,
        placeholder: t('ruleVersionDomain.abandonReasonPlaceholder'),
        onChange: (event: Event) => {
          reason = (event.target as HTMLTextAreaElement).value
        }
      }),
    okText: t('ruleVersionDomain.abandon'),
    cancelText: t('ruleVersionDomain.cancel'),
    okButtonProps: { danger: true },
    async onOk() {
      if (submitting) {
        return Promise.reject()
      }
      const trimmed = normalizeIntentReason(reason)
      if (!trimmed) {
        notification.warning({
          key: 'rule-version-abandon-reason-required',
          message: t('ruleVersionDomain.abandonReasonRequired')
        })
        return Promise.reject()
      }
      submitting = true
      modal.update({ okButtonProps: { danger: true, loading: true } })
      try {
        await abandonRuleVersionIntentAPI(intentId, trimmed)
        notification.close('rule-version-ledger-pending')
        notification.close('rule-version-abandon-reason-required')
        await options?.reload?.()
      } catch (e: any) {
        notification.error({
          key: 'rule-version-abandon-error',
          message: t('ruleVersionDomain.abandonFailed'),
          description: e?.message || String(e)
        })
        submitting = false
        modal.update({ okButtonProps: { danger: true, loading: false } })
        return Promise.reject()
      }
    }
  })
}

export const notifyVersionConflict = (
  e: any,
  options?: { reload?: () => void | Promise<void> }
): boolean => {
  if (isVersionConflict(e)) {
    notification.warning({
      key: 'rule-version-conflict',
      duration: 0,
      message: t('ruleVersionDomain.versionConflict'),
      description: t('ruleVersionDomain.versionConflictDescription'),
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
              { default: () => t('ruleVersionDomain.reload') }
            )
        : undefined
    })
    return true
  }
  return false
}

export const notifyVersionLedgerPending = (
  e: any,
  options?: { reload?: () => void | Promise<void> }
): boolean => {
  if (!isVersionLedgerPending(e)) {
    return false
  }
  const intentId = e.intentId
  notification.warning({
    key: 'rule-version-ledger-pending',
    duration: 0,
    message: t('ruleVersionDomain.ledgerPending'),
    description: intentId
      ? t('ruleVersionDomain.ledgerPendingWithIntent', { intentId })
      : t('ruleVersionDomain.ledgerPendingDescription'),
    btn: intentId
      ? () =>
          h(
            Space,
            {},
            {
              default: () => [
                h(
                  Button,
                  {
                    type: 'link',
                    size: 'small',
                    onClick: async () => {
                      if (repairingIntentIds.has(intentId)) {
                        return
                      }
                      repairingIntentIds.add(intentId)
                      try {
                        await repairRuleVersionIntentAPI(intentId)
                        notification.close('rule-version-ledger-pending')
                        await options?.reload?.()
                      } catch (e: any) {
                        notification.error({
                          key: 'rule-version-repair-error',
                          message: t('ruleVersionDomain.repairFailed'),
                          description: e?.message || String(e)
                        })
                      } finally {
                        repairingIntentIds.delete(intentId)
                      }
                    }
                  },
                  { default: () => t('ruleVersionDomain.repair') }
                ),
                h(
                  Button,
                  {
                    type: 'link',
                    size: 'small',
                    danger: true,
                    onClick: () => {
                      openAbandonReasonModal(intentId, options)
                    }
                  },
                  { default: () => t('ruleVersionDomain.abandon') }
                )
              ]
            }
          )
      : options?.reload
        ? () =>
            h(
              Button,
              {
                type: 'link',
                size: 'small',
                onClick: () => {
                  notification.close('rule-version-ledger-pending')
                  options.reload?.()
                }
              },
              { default: () => t('ruleVersionDomain.reload') }
            )
        : undefined
  })
  return true
}

export const notifyRuleVersionError = (
  e: any,
  options?: { reload?: () => void | Promise<void> }
): boolean => {
  if (notifyVersionLedgerPending(e, options)) {
    return true
  }
  return notifyVersionConflict(e, options)
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
