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
  <a-card>
    <a-alert :message="hint" type="info" show-icon style="margin-bottom: 12px" />
    <a-alert
      v-if="loadError"
      :message="loadError"
      type="error"
      show-icon
      style="margin-bottom: 12px"
    />
    <div class="editor-box">
      <a-spin :spinning="loadingDetail">
        <MonacoEditor
          v-if="!loadingDetail"
          v-model:modelValue="yamlValue"
          theme="vs-dark"
          :height="520"
          language="yaml"
        />
      </a-spin>
    </div>
    <a-space style="margin-top: 16px">
      <a-button
        type="primary"
        :loading="loading"
        :disabled="loadingDetail || !!loadError"
        @click="save"
      >
        {{
          editing
            ? t('routerRuleDomain.saveChanges')
            : t('routerRuleDomain.create', { rule: ruleLabel })
        }}
      </a-button>
      <a-button :disabled="loading || loadingDetail" @click="router.push(basePath)">
        {{ t('cancel') }}
      </a-button>
    </a-space>
  </a-card>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { onBeforeRouteLeave, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import yaml from 'js-yaml'
import { message, Modal } from 'ant-design-vue'
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import {
  addRouterRuleAPI,
  getRouterRuleAPI,
  updateRouterRuleAPI,
  type RouterRuleKind
} from '@/api/service/traffic'
import { HTTP_STATUS } from '@/base/http/constants'

const props = defineProps<{ kind: RouterRuleKind }>()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const loading = ref(false)
const loadingDetail = ref(false)
const loadError = ref('')
const ruleName = computed(() => String(route.params.ruleName || ''))
const editing = computed(() => ruleName.value.length > 0)
const ruleLabel = computed(() => t(props.kind === 'affinity-rule' ? 'affinityRule' : 'scriptRule'))
const suffix = computed(() =>
  props.kind === 'affinity-rule' ? '.affinity-router' : '.script-router'
)
const basePath = computed(
  () => `/traffic/${props.kind === 'affinity-rule' ? 'affinityRule' : 'scriptRule'}`
)
const hint = computed(() =>
  props.kind === 'affinity-rule'
    ? t('routerRuleDomain.affinityHint')
    : t('routerRuleDomain.scriptHint')
)
const defaults = computed(() =>
  props.kind === 'affinity-rule'
    ? {
        configVersion: 'v3.1',
        scope: 'application',
        key: 'demo-provider',
        enabled: true,
        runtime: true,
        affinityAware: { key: 'region', ratio: 80 }
      }
    : {
        configVersion: 'v3.0',
        scope: 'application',
        key: 'demo-provider',
        enabled: true,
        type: 'javascript',
        script:
          '(function route(invokers, invocation, context) {\n  return invokers;\n})(invokers, invocation, context)'
      }
)
const yamlValue = ref('')
const savedYamlValue = ref('')
const isDirty = computed(
  () => savedYamlValue.value !== '' && yamlValue.value !== savedYamlValue.value
)
let currentLoad = 0

function resetToDefault() {
  const value = yaml.dump(defaults.value)
  yamlValue.value = value
  savedYamlValue.value = value
}

async function loadRule() {
  const loadID = ++currentLoad
  loadError.value = ''
  if (!editing.value) {
    resetToDefault()
    return
  }

  loadingDetail.value = true
  try {
    const res = await getRouterRuleAPI(props.kind, ruleName.value)
    if (loadID !== currentLoad) return
    if (res.code !== HTTP_STATUS.SUCCESS) {
      loadError.value = t('routerRuleDomain.loadFailed')
      return
    }
    const value = yaml.dump(res.data)
    yamlValue.value = value
    savedYamlValue.value = value
  } catch (_error) {
    if (loadID === currentLoad) loadError.value = t('routerRuleDomain.loadFailed')
  } finally {
    if (loadID === currentLoad) loadingDetail.value = false
  }
}

watch(ruleName, () => void loadRule(), { immediate: true })

onBeforeRouteLeave(() => {
  if (!isDirty.value || loading.value) return true
  return new Promise<boolean>((resolve) => {
    Modal.confirm({
      title: t('routerRuleDomain.unsavedChangesTitle'),
      content: t('routerRuleDomain.unsavedChangesBody'),
      okText: t('routerRuleDomain.discardChanges'),
      cancelText: t('routerRuleDomain.keepEditing'),
      okButtonProps: { danger: true },
      onOk: () => resolve(true),
      onCancel: () => resolve(false)
    })
  })
})

// save derives ruleName from key + suffix on create, matching the backend and
// Dubbo runtime subscription contract.
async function save() {
  if (loading.value || loadingDetail.value || loadError.value) return
  loading.value = true
  try {
    const data = yaml.load(yamlValue.value)
    if (!data || typeof data !== 'object' || Array.isArray(data))
      throw new Error(t('routerRuleDomain.yamlMustBeObject'))
    const rule = data as Record<string, unknown>
    const key = String(rule.key || '').trim()
    if (!key) throw new Error(t('routerRuleDomain.keyRequired'))
    rule.key = key
    const targetName = editing.value ? ruleName.value : `${key}${suffix.value}`
    const res = editing.value
      ? await updateRouterRuleAPI(props.kind, targetName, rule)
      : await addRouterRuleAPI(props.kind, targetName, rule)
    if (res.code === HTTP_STATUS.SUCCESS) {
      savedYamlValue.value = yamlValue.value
      message.success(
        editing.value
          ? t('routerRuleDomain.updateSuccess', { rule: ruleLabel.value })
          : t('routerRuleDomain.createSuccess', { rule: ruleLabel.value })
      )
      await router.push(basePath.value)
    }
  } catch (error: any) {
    message.error(error?.message || t('routerRuleDomain.saveFailed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.editor-box {
  min-height: 520px;
  overflow: hidden;
  border-radius: 6px;
}
</style>
