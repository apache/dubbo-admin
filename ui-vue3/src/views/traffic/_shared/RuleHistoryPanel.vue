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
    :latest-recorded-version-no="latestRecordedVersionNo"
    :loading="loading"
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
    <div v-if="rollbackTarget" style="margin-bottom: 16px">
      <div>
        <strong>{{ t('ruleVersionDomain.targetVersion') }}:</strong>
        v{{ rollbackTarget.versionNo }}
      </div>
      <div>
        <strong>{{ t('ruleVersionDomain.latestRecordedVersion') }}:</strong>
        {{ latestRecordedVersionNo ? `v${latestRecordedVersionNo}` : t('ruleVersionDomain.none') }}
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
  latestRecordedStateFromList,
  formatRuleSpec,
  isLatestRecordedHistoryRequest,
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
  (e: 'latest-recorded-version-no-change', value: number | undefined): void
}>()

const { t } = useI18n()

const openProxy = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})

const items = ref<RuleVersion[]>([])
const latestRecordedVersionNo = ref<number | undefined>(undefined)
const loading = ref(false)
const versionJsonOpen = ref(false)
const versionJson = ref('')
const versionDiffOpen = ref(false)
const versionDiffLeft = ref('')
const versionDiffRight = ref('')
const versionDiffLeftLabel = ref(t('ruleVersionDomain.targetVersion'))
const versionDiffRightLabel = ref(t('ruleVersionDomain.currentRule'))
const rollbackConfirmOpen = ref(false)
const rollbackTarget = ref<RuleVersion | null>(null)
const rollbackReason = ref('')
const rollbackLoading = ref(false)
let requestSeq = 0
let operationSeq = 0
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

type OperationToken = {
  seq: number
  open: boolean
  kind: TrafficRuleKind
  ruleName: string
  targetVersionNo?: number
}

const nextOperationToken = (targetVersionNo?: number): OperationToken => ({
  seq: ++operationSeq,
  open: props.open,
  kind: props.kind,
  ruleName: props.ruleName,
  targetVersionNo
})

// Async drawer actions outlive loading flags when the drawer is closed or a
// different rule is selected. The token keeps stale responses from reopening
// modals or overwriting state for the next rule.
const isLatestRecordedOperation = (
  token: OperationToken,
  targetVersionNo = token.targetVersionNo
) =>
  !disposed &&
  token.seq === operationSeq &&
  token.open &&
  props.open &&
  token.kind === props.kind &&
  token.ruleName === props.ruleName &&
  token.targetVersionNo === targetVersionNo

async function loadHistory() {
  // Loading alone cannot distinguish an older request from a newer one. The
  // sequence guard lets the newest rule/open state own the history snapshot.
  const seq = ++requestSeq
  const kind = props.kind
  const ruleName = props.ruleName
  if (!props.open || !ruleName || ruleName === '_tmp') {
    items.value = []
    latestRecordedVersionNo.value = undefined
    emit('latest-recorded-version-no-change', undefined)
    return
  }

  loading.value = true
  try {
    const res = await listRuleVersionsAPI(kind, ruleName)
    if (!isLatestRecordedHistoryRequest(seq, requestSeq, disposed)) {
      return
    }
    if (res?.code === HTTP_STATUS.SUCCESS) {
      items.value = res.data?.items || []
      const latestRecorded = latestRecordedStateFromList(res.data)
      latestRecordedVersionNo.value = latestRecorded.versionNo
      emit('latest-recorded-version-no-change', latestRecordedVersionNo.value)
    }
  } catch (e: any) {
    if (!isLatestRecordedHistoryRequest(seq, requestSeq, disposed)) {
      return
    }
    throw e
  } finally {
    if (isLatestRecordedHistoryRequest(seq, requestSeq, disposed)) {
      loading.value = false
    }
  }
}

const openVersionJson = (item: RuleVersion) => {
  versionJson.value = formatRuleSpec(item.specJson)
  versionJsonOpen.value = true
}

const openVersionDiff = async (item: RuleVersion) => {
  const token = nextOperationToken(item.versionNo)
  try {
    const res = await diffRuleVersionAPI(token.kind, token.ruleName, item.versionNo)
    if (!isLatestRecordedOperation(token, item.versionNo)) {
      return
    }
    if (res?.code === HTTP_STATUS.SUCCESS) {
      versionDiffLeft.value = formatRuleSpec(res.data.left.specJson)
      versionDiffRight.value = formatRuleSpec(res.data.right.specJson)
      versionDiffLeftLabel.value = versionDiffLabel(
        t('ruleVersionDomain.targetVersion'),
        res.data.left?.versionNo
      )
      versionDiffRightLabel.value = versionDiffLabel(
        t('ruleVersionDomain.currentRule'),
        res.data.right?.versionNo
      )
      versionDiffOpen.value = true
    }
  } catch (e: any) {
    if (isLatestRecordedOperation(token, item.versionNo)) {
      message.error(e?.message || t('ruleVersionDomain.diffFailed'))
    }
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

  const target = rollbackTarget.value
  const token = nextOperationToken(target.versionNo)
  rollbackLoading.value = true
  try {
    const res = await rollbackRuleVersionAPI(
      token.kind,
      token.ruleName,
      target.versionNo,
      rollbackReason.value
    )
    if (
      !isLatestRecordedOperation(token, target.versionNo) ||
      rollbackTarget.value?.versionNo !== target.versionNo
    ) {
      return
    }
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
    if (
      !isLatestRecordedOperation(token, target.versionNo) ||
      rollbackTarget.value?.versionNo !== target.versionNo
    ) {
      return
    }
    message.error(e?.message || t('ruleVersionDomain.rollbackFailed'))
  } finally {
    if (isLatestRecordedOperation(token, target.versionNo)) {
      rollbackLoading.value = false
    }
  }
}

watch(
  () => [props.open, props.kind, props.ruleName] as const,
  ([open]) => {
    if (open) {
      loadHistory()
    } else {
      // Closing the drawer invalidates in-flight history, diff, and rollback
      // responses so they cannot update a later open cycle.
      requestSeq++
      operationSeq++
      loading.value = false
    }
  },
  { immediate: true }
)

onBeforeUnmount(() => {
  disposed = true
  requestSeq++
  operationSeq++
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
