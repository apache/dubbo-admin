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
    <div class="editor-box">
      <MonacoEditor v-model:modelValue="yamlValue" theme="vs-dark" :height="520" language="yaml" />
    </div>
    <a-space style="margin-top: 16px">
      <a-button type="primary" :loading="loading" @click="save">{{ t('confirm') }}</a-button>
      <a-button @click="router.push(basePath)">{{ t('cancel') }}</a-button>
    </a-space>
  </a-card>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import yaml from 'js-yaml'
import { message } from 'ant-design-vue'
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
const ruleName = computed(() => String(route.params.ruleName || ''))
const editing = computed(() => ruleName.value.length > 0)
const suffix = computed(() =>
  props.kind === 'affinity-rule' ? '.affinity-router' : '.script-router'
)
const basePath = computed(
  () => `/traffic/${props.kind === 'affinity-rule' ? 'affinityRule' : 'scriptRule'}`
)
const hint = computed(() =>
  props.kind === 'affinity-rule'
    ? 'Affinity supports application or service scope. Service key format: <interface>:<version>:<group>.'
    : 'Script rules support application scope and the registered javascript type only.'
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
        script: '(function route(invokers, invocation, context) {\n  return invokers;\n})()'
      }
)
const yamlValue = ref(yaml.dump(defaults.value))

onMounted(async () => {
  if (!editing.value) return
  const res = await getRouterRuleAPI(props.kind, ruleName.value)
  if (res.code === HTTP_STATUS.SUCCESS) yamlValue.value = yaml.dump(res.data)
})

// save derives ruleName from key + suffix on create, matching the backend and
// Dubbo runtime subscription contract.
async function save() {
  loading.value = true
  try {
    const data = yaml.load(yamlValue.value)
    if (!data || typeof data !== 'object' || Array.isArray(data))
      throw new Error('YAML content must be an object')
    const key = String((data as Record<string, any>).key || '')
    if (!key) throw new Error('key is required')
    const targetName = editing.value ? ruleName.value : `${key}${suffix.value}`
    const res = editing.value
      ? await updateRouterRuleAPI(props.kind, targetName, data)
      : await addRouterRuleAPI(props.kind, targetName, data)
    if (res.code === HTTP_STATUS.SUCCESS) {
      message.success(editing.value ? 'update success' : 'create success')
      await router.push(basePath.value)
    }
  } catch (error: any) {
    message.error(error?.message || String(error))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.editor-box {
  overflow: hidden;
  border-radius: 6px;
}
</style>
