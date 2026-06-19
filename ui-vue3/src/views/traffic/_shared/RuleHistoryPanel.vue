<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->

<template>
  <RuleHistoryDrawer
    v-model:open="openProxy"
    :title="title"
    :items="items"
    :current-version-no="currentVersionNo"
    :loading="loading"
    :disabled="disabled"
    @view-json="openVersionJson"
    @diff-current="openVersionDiff"
    @rollback="openRollbackConfirm"
  />

  <a-modal
    v-model:open="versionJsonOpen"
    :title="t('ruleVersionDomain.versionJson')"
    width="900px"
    :footer="null"
  >
    <MonacoEditor
      v-model:modelValue="versionJson"
      language="json"
      theme="vs-dark"
      height="500px"
      :readonly="true"
    />
  </a-modal>

  <a-modal
    v-model:open="versionDiffOpen"
    :title="t('ruleVersionDomain.versionDiff')"
    width="1100px"
    :footer="null"
  >
    <div class="version-diff-labels">
      <span>{{ versionDiffLeftLabel }}</span>
      <span>{{ versionDiffRightLabel }}</span>
    </div>
    <RuleDiffEditor :original="versionDiffLeft" :modified="versionDiffRight" height="520px" />
  </a-modal>

  <a-modal
    v-model:open="rollbackConfirmOpen"
    :title="t('ruleVersionDomain.rollbackConfirmTitle')"
    width="700px"
    @ok="handleRollbackConfirm"
    :confirmLoading="rollbackLoading"
  >
    <a-alert
      v-if="currentDeleted"
      :message="t('ruleVersionDomain.rollbackDeletedWarning')"
      type="warning"
      show-icon
      style="margin-bottom: 16px"
    />
    <div v-if="rollbackTarget" style="margin-bottom: 16px">
      <div>
        <strong>{{ t('ruleVersionDomain.targetVersion') }}:</strong>
        v{{ rollbackTarget.versionNo }}
      </div>
      <div>
        <strong>{{ t('ruleVersionDomain.currentVersion') }}:</strong>
        {{ currentVersionNo ? `v${currentVersionNo}` : t('ruleVersionDomain.none') }}
        <span v-if="currentVersionId">({{ currentVersionId }})</span>
      </div>
      <div>
        <strong>{{ t('ruleVersionDomain.source') }}:</strong>
        {{ sourceLabel(rollbackTarget.source) }}
      </div>
      <div>
        <strong>{{ t('ruleVersionDomain.author') }}:</strong>
        {{ authorLabel(rollbackTarget.author) }}
      </div>
      <div>
        <strong>{{ t('ruleVersionDomain.createdAt') }}:</strong>
        {{ createdAtLabel(rollbackTarget.createdAt) }}
      </div>
    </div>
    <a-alert
      :message="t('ruleVersionDomain.rollbackCasHint')"
      type="info"
      show-icon
      style="margin-bottom: 12px"
    />
    <a-typography-text type="secondary" class="rollback-hint">
      {{ t('ruleVersionDomain.rollbackAppendHint') }}
    </a-typography-text>
    <a-form layout="vertical">
      <a-form-item :label="t('ruleVersionDomain.rollbackReason')" required>
        <a-textarea
          v-model:value="rollbackReason"
          :placeholder="t('ruleVersionDomain.rollbackReasonPlaceholder')"
          :rows="3"
          :maxlength="1024"
        />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
import { useI18n } from 'vue-i18n'
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import { HTTP_STATUS } from '@/base/http/constants'
import {
  diffRuleVersionAPI,
  listRuleVersionsAPI,
  rollbackRuleVersionAPI,
  type RuleVersion,
  type TrafficRuleKind
} from '@/api/service/traffic'
import RuleHistoryDrawer from './RuleHistoryDrawer.vue'
import RuleDiffEditor from './RuleDiffEditor.vue'
import {
  currentVersionStateFromList,
  formatRuleSpec,
  isCurrentHistoryRequest,
  isVersionConflict,
  isVersionLedgerPending,
  rollbackExpectedVersionId,
  versionDiffLabel
} from './ruleVersion'
import dayjs from 'dayjs'

const props = defineProps<{
  open: boolean
  kind: TrafficRuleKind
  ruleName: string
  title: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'current-version-change', value: string | undefined): void
  (e: 'current-version-no-change', value: number | undefined): void
}>()

const { t } = useI18n()

const openProxy = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})

const items = ref<RuleVersion[]>([])
const currentVersionId = ref<string | undefined>(undefined)
const currentVersionNo = ref<number | undefined>(undefined)
const currentDeleted = ref(false)
const loading = ref(false)
const disabled = ref(false)
const versionJsonOpen = ref(false)
const versionJson = ref('')
const versionDiffOpen = ref(false)
const versionDiffLeft = ref('')
const versionDiffRight = ref('')
const versionDiffLeftLabel = ref(t('ruleVersionDomain.targetVersion'))
const versionDiffRightLabel = ref(t('ruleVersionDomain.currentVersion'))
const rollbackConfirmOpen = ref(false)
const rollbackTarget = ref<RuleVersion | null>(null)
const rollbackReason = ref('')
const rollbackLoading = ref(false)
let requestSeq = 0
let disposed = false

const sourceLabels: Record<string, string> = {
  ADMIN: 'ruleVersionDomain.sourceAdmin',
  UPSTREAM: 'ruleVersionDomain.sourceUpstream',
  BOOTSTRAP: 'ruleVersionDomain.sourceBootstrap',
  ROLLBACK: 'ruleVersionDomain.sourceRollback'
}

const sourceLabel = (source: string) => (sourceLabels[source] ? t(sourceLabels[source]) : source)
const authorLabel = (author: string) => author.replace(/^system:/, '')
const createdAtLabel = (createdAt: string) => dayjs(createdAt).format('YYYY/M/D HH:mm:ss')

async function loadHistory() {
  const seq = ++requestSeq
  const kind = props.kind
  const ruleName = props.ruleName
  if (!props.open || !ruleName || ruleName === '_tmp') {
    items.value = []
    currentVersionId.value = undefined
    currentVersionNo.value = undefined
    currentDeleted.value = false
    emit('current-version-change', undefined)
    emit('current-version-no-change', undefined)
    return
  }

  loading.value = true
  disabled.value = false
  try {
    const res = await listRuleVersionsAPI(kind, ruleName)
    if (!isCurrentHistoryRequest(seq, requestSeq, disposed)) {
      return
    }
    if (res?.code === HTTP_STATUS.SUCCESS) {
      items.value = res.data?.items || []
      const current = currentVersionStateFromList(res.data)
      currentVersionId.value = current.id
      currentVersionNo.value = current.versionNo
      currentDeleted.value = current.deleted
      emit('current-version-change', currentVersionId.value)
      emit('current-version-no-change', currentVersionNo.value)
    }
  } catch (e: any) {
    if (!isCurrentHistoryRequest(seq, requestSeq, disposed)) {
      return
    }
    if (e?.code === 'FEATURE_DISABLED') {
      disabled.value = true
      items.value = []
      currentVersionId.value = undefined
      currentVersionNo.value = undefined
      currentDeleted.value = false
      emit('current-version-change', undefined)
      emit('current-version-no-change', undefined)
      return
    }
    throw e
  } finally {
    if (isCurrentHistoryRequest(seq, requestSeq, disposed)) {
      loading.value = false
    }
  }
}

const openVersionJson = (item: RuleVersion) => {
  versionJson.value = formatRuleSpec(item.specJson)
  versionJsonOpen.value = true
}

const openVersionDiff = async (item: RuleVersion) => {
  const res = await diffRuleVersionAPI(props.kind, props.ruleName, item.id)
  if (res?.code === HTTP_STATUS.SUCCESS) {
    versionDiffLeft.value = formatRuleSpec(res.data.left.specJson)
    versionDiffRight.value = formatRuleSpec(res.data.right.specJson)
    versionDiffLeftLabel.value = versionDiffLabel(
      t('ruleVersionDomain.targetVersion'),
      res.data.left?.versionNo
    )
    versionDiffRightLabel.value = currentDeleted.value
      ? t('ruleVersionDomain.currentDeleted')
      : versionDiffLabel(t('ruleVersionDomain.currentVersion'), res.data.right?.versionNo)
    versionDiffOpen.value = true
  }
}

const openRollbackConfirm = (item: RuleVersion) => {
  rollbackTarget.value = item
  rollbackReason.value = ''
  rollbackConfirmOpen.value = true
}

const handleRollbackConfirm = async () => {
  if (!rollbackTarget.value) return
  if (!rollbackReason.value.trim()) {
    message.warning(t('ruleVersionDomain.rollbackReasonRequired'))
    return
  }

  rollbackLoading.value = true
  try {
    // Send the current version as a weak CAS guard so rollback does not
    // overwrite a newer change made after the drawer was opened.
    const res = await rollbackRuleVersionAPI(
      props.kind,
      props.ruleName,
      rollbackTarget.value.id,
      rollbackReason.value,
      rollbackExpectedVersionId({
        id: currentVersionId.value,
        versionNo: currentVersionNo.value,
        deleted: currentDeleted.value
      })
    )
    if (res?.code === HTTP_STATUS.SUCCESS) {
      const versionNo = res.data?.versionNo
      message.success(
        versionNo
          ? t('ruleVersionDomain.rollbackSuccessWithVersion', { versionNo })
          : t('ruleVersionDomain.rollbackSuccess')
      )
      rollbackConfirmOpen.value = false
      await loadHistory()
    }
  } catch (e: any) {
    if (isVersionConflict(e)) {
      message.error(t('ruleVersionDomain.rollbackConflict'))
      await loadHistory()
    } else if (isVersionLedgerPending(e)) {
      message.error(t('ruleVersionDomain.rollbackPending'))
      await loadHistory()
    } else {
      message.error(e?.message || t('ruleVersionDomain.rollbackFailed'))
    }
  } finally {
    rollbackLoading.value = false
  }
}

watch(
  () => [props.open, props.kind, props.ruleName] as const,
  ([open]) => {
    if (open) {
      loadHistory()
    } else {
      requestSeq++
      loading.value = false
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  disposed = true
  requestSeq++
})
</script>

<style scoped>
.version-diff-labels {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 8px;
  color: var(--el-text-color-regular, rgba(0, 0, 0, 0.65));
  font-size: 13px;
  font-weight: 500;
}

.rollback-hint {
  display: block;
  margin: -4px 0 12px;
  font-size: 13px;
}
</style>
