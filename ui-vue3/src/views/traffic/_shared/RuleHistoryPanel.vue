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
    @rollback="openVersionRollback"
  />

  <a-modal v-model:open="versionJsonOpen" title="Version JSON" width="900px" :footer="null">
    <MonacoEditor
      v-model:modelValue="versionJson"
      language="json"
      theme="vs-dark"
      height="500px"
      :readonly="true"
    />
  </a-modal>

  <a-modal v-model:open="versionDiffOpen" title="Version Diff" width="1100px" :footer="null">
    <RuleDiffEditor :original="versionDiffLeft" :modified="versionDiffRight" height="520px" />
  </a-modal>

  <a-modal v-model:open="rollbackOpen" title="Rollback" ok-text="Rollback" @ok="confirmRollback">
    <a-form layout="vertical">
      <a-form-item label="Reason" required>
        <a-textarea v-model:value="rollbackReason" :rows="4" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { message } from 'ant-design-vue'
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
import { currentVersionStateFromItems, formatRuleSpec, notifyVersionConflict } from './ruleVersion'

const props = defineProps<{
  open: boolean
  kind: TrafficRuleKind
  ruleName: string
  title: string
}>()

const emit = defineEmits<{
  (e: 'update:open', value: boolean): void
  (e: 'rollback-success', item: RuleVersion): void
  (e: 'current-version-change', value: number | undefined): void
  (e: 'current-version-no-change', value: number | undefined): void
}>()

const openProxy = computed({
  get: () => props.open,
  set: (value: boolean) => emit('update:open', value)
})

const items = ref<RuleVersion[]>([])
const currentVersionId = ref<number | undefined>(undefined)
const currentVersionNo = ref<number | undefined>(undefined)
const loading = ref(false)
const disabled = ref(false)
const versionJsonOpen = ref(false)
const versionJson = ref('')
const versionDiffOpen = ref(false)
const versionDiffLeft = ref('')
const versionDiffRight = ref('')
const rollbackOpen = ref(false)
const rollbackReason = ref('')
const rollbackTarget = ref<RuleVersion | null>(null)

async function loadHistory() {
  if (!props.ruleName || props.ruleName === '_tmp') {
    items.value = []
    currentVersionId.value = undefined
    currentVersionNo.value = undefined
    emit('current-version-change', undefined)
    emit('current-version-no-change', undefined)
    return
  }

  loading.value = true
  disabled.value = false
  try {
    const res = await listRuleVersionsAPI(props.kind, props.ruleName)
    if (res?.code === HTTP_STATUS.SUCCESS) {
      items.value = res.data?.items || []
      const current = currentVersionStateFromItems(items.value)
      currentVersionId.value = current.id
      currentVersionNo.value = current.versionNo
      emit('current-version-change', currentVersionId.value)
      emit('current-version-no-change', currentVersionNo.value)
    }
  } catch (e: any) {
    if (e?.code === 'FEATURE_DISABLED') {
      disabled.value = true
      items.value = []
      currentVersionId.value = undefined
      currentVersionNo.value = undefined
      emit('current-version-change', undefined)
      emit('current-version-no-change', undefined)
      return
    }
    throw e
  } finally {
    loading.value = false
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
    versionDiffOpen.value = true
  }
}

const openVersionRollback = (item: RuleVersion) => {
  rollbackTarget.value = item
  rollbackReason.value = ''
  rollbackOpen.value = true
}

const confirmRollback = async () => {
  if (!rollbackTarget.value) {
    return
  }
  if (!rollbackReason.value.trim()) {
    message.warning('请填写回滚原因')
    return
  }
  try {
    const res = await rollbackRuleVersionAPI(props.kind, props.ruleName, rollbackTarget.value.id, {
      reason: rollbackReason.value,
      expectedVersionId: currentVersionId.value
    })
    if (res?.code === HTTP_STATUS.SUCCESS) {
      message.success('rollback success')
      rollbackOpen.value = false
      await loadHistory()
      emit('rollback-success', res.data)
    }
  } catch (e: any) {
    notifyVersionConflict(e, { reload: loadHistory })
  }
}

watch(
  () => props.ruleName,
  () => loadHistory(),
  { immediate: true }
)

watch(
  () => props.open,
  (open) => {
    if (open) {
      loadHistory()
    }
  }
)
</script>
