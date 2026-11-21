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
  <div class="__container_services_tabs_debug">
    <a-card :bordered="false" :body-style="{ padding: '24px' }">
      <div class="tabs-title">
        <a-typography-text strong>方法列表</a-typography-text>
      </div>
      <a-spin :spinning="loadingMethods">
        <a-empty
          v-if="!loadingMethods && methodList.length === 0"
          :description="emptyDescription"
        />
        <a-tabs v-else v-model:activeKey="activeKey" tab-position="left" class="debug-tabs">
          <a-tab-pane
            v-for="(method, index) in methodList"
            :key="String(index)"
            :tab="`${method.methodName}(${(method.parameterTypes || []).join(', ')})`"
          >
            <a-spin :spinning="loadingDetail">
              <div class="tab-content">
                <a-row :gutter="[24, 24]">
                  <!-- Row 1: Parameter Types -->
                  <a-col :span="12">
                    <div class="section-title">
                      <a-typography-text strong>入参类型:</a-typography-text>
                    </div>
                    <a-tree
                      v-if="enterParamType.length > 0"
                      block-node
                      :tree-data="enterParamType"
                      default-expand-all
                    />
                    <a-typography-text type="secondary" v-else class="empty-hint">
                      无入参
                    </a-typography-text>
                  </a-col>
                  <a-col :span="12">
                    <div class="section-title">
                      <a-typography-text strong>出参类型:</a-typography-text>
                    </div>
                    <a-tree
                      v-if="outputParamType.length > 0"
                      block-node
                      :tree-data="outputParamType"
                      default-expand-all
                    />
                    <a-typography-text type="secondary" v-else class="empty-hint">
                      无出参
                    </a-typography-text>
                  </a-col>

                  <!-- Row 2: Request & Response Editors -->
                  <a-col :span="12">
                    <div class="section-title">
                      <a-typography-text strong>请求:</a-typography-text>
                      <a-typography-text type="secondary">{{
                        $t('serviceDomain.integerInputHint')
                      }}</a-typography-text>
                    </div>
                    <div class="editor-wrapper">
                      <monaco-editor
                        v-model="requestValue"
                        :editor-id="`requestEditor-${index}`"
                        height="300px"
                        class="monaco-container"
                      />
                      <a-tag class="editor-tag" :bordered="false">JSON</a-tag>
                    </div>
                  </a-col>
                  <a-col :span="12">
                    <div class="section-title">
                      <a-typography-text strong>响应:</a-typography-text>
                      <a-tag :color="PRIMARY_COLOR" :bordered="false" style="margin-left: 8px">
                        <template #icon>
                          <clock-circle-outlined />
                        </template>
                        耗时: {{ elapsedMs }}ms
                      </a-tag>
                    </div>
                    <div class="editor-wrapper">
                      <monaco-editor
                        v-model="responseValue"
                        :editor-id="`responseEditor-${index}`"
                        height="300px"
                        :readonly="true"
                        class="monaco-container"
                      />
                      <a-tag class="editor-tag" :bordered="false">JSON</a-tag>
                    </div>
                  </a-col>

                  <!-- Row 3: Bottom Settings -->
                  <a-col :span="8">
                    <div class="section-title">
                      <a-typography-text strong>调用实例:</a-typography-text>
                    </div>
                    <a-select
                      v-model:value="instanceName"
                      :options="providerInstanceOptions"
                      :loading="loadingProviders"
                      :disabled="providerInstanceOptions.length === 0"
                      placeholder="请选择真实调用实例"
                      show-search
                      option-filter-prop="label"
                      allow-clear
                      style="width: 100%"
                    />
                    <a-typography-text
                      type="secondary"
                      v-if="!loadingProviders && providerInstanceOptions.length === 0"
                      class="empty-hint"
                      style="display: block; margin-top: 4px"
                    >
                      当前服务没有可调用实例
                    </a-typography-text>
                  </a-col>
                  <a-col :span="8">
                    <div class="section-title">
                      <a-typography-text strong>自定义超时时间</a-typography-text>
                    </div>
                    <div class="setting-item">
                      <a-input-number v-model:value="timeout" :min="0" style="width: 120px" />
                      <a-typography-text type="secondary" class="unit">ms</a-typography-text>
                    </div>
                  </a-col>
                  <a-col :span="8">
                    <div class="section-title">
                      <a-typography-text strong>传递attachments</a-typography-text>
                    </div>
                    <div class="setting-item">
                      <a-button
                        type="link"
                        @click="attachmentsModalOpen = true"
                        class="attachment-edit-btn"
                      >
                        <template #icon><edit-outlined /></template>
                        编辑 ({{ attachmentCount }})
                      </a-button>
                    </div>
                  </a-col>

                  <!-- Row 4: Invoke Button -->
                  <a-col :span="24" class="action-row">
                    <a-button
                      type="primary"
                      size="large"
                      class="invoke-btn"
                      :disabled="!instanceName || !currentMethodDetail || loadingDetail"
                      :loading="loadingInvoke"
                      @click="handleInvoke"
                    >
                      发起请求
                    </a-button>
                  </a-col>
                </a-row>
              </div>
            </a-spin>
          </a-tab-pane>
        </a-tabs>
      </a-spin>
    </a-card>

    <!-- Attachments Modal -->
    <a-modal
      v-model:open="attachmentsModalOpen"
      title="传递 Attachments"
      @ok="attachmentsModalOpen = false"
      @cancel="attachmentsModalOpen = false"
    >
      <div class="attachments-list">
        <div v-for="(item, idx) in attachmentsList" :key="idx" class="attachment-row">
          <a-input v-model:value="item.key" placeholder="Key" style="width: 45%" />
          <span class="kv-sep">:</span>
          <a-input v-model:value="item.value" placeholder="Value" style="width: 45%" />
          <a-button type="text" danger @click="removeAttachment(idx)">
            <template #icon><minus-circle-outlined /></template>
          </a-button>
        </div>
        <a-button type="dashed" block @click="addAttachment" style="margin-top: 8px">
          <plus-outlined /> 添加
        </a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import {
  ClockCircleOutlined,
  EditOutlined,
  MinusCircleOutlined,
  PlusOutlined
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR, PRIMARY_COLOR_R } from '@/base/constants'
import {
  getServiceProviderInstancesAPI,
  getServiceMethodsAPI,
  getServiceMethodDetailAPI,
  serviceGenericInvokeAPI
} from '@/api/service/service'
import { useMeshStore } from '@/stores/mesh'

defineOptions({
  name: 'ServiceDebugTab'
})

interface MethodSummary {
  methodName: string
  parameterTypes: string[]
  signature?: string
}

interface TypeDef {
  type: string
  properties: Record<string, string>
  items: string[]
  enums: string[]
}

interface ParameterDef {
  name: string
  type: string
}

interface MethodDetail {
  methodName: string
  signature?: string
  parameterTypes: string[]
  parameters: ParameterDef[]
  returnType: string
  types: TypeDef[]
}

interface ProviderInstance {
  name: string
  appName: string
  ip: string
}

const route = useRoute()
const meshStore = useMeshStore()
const serviceName = computed(() => route.params.pathId as string)
const group = computed(() => (route.params.group as string) || '')
const version = computed(() => (route.params.version as string) || '')

const methodList = ref<MethodSummary[]>([])
const loadingMethods = ref(false)
const providerInstances = ref<ProviderInstance[]>([])
const loadingProviders = ref(false)
const activeKey = ref('0')
const instanceName = ref('')

const currentMethodDetail = ref<MethodDetail | null>(null)
const loadingDetail = ref(false)
const loadingInvoke = ref(false)

const requestValue = ref('[]')
const responseValue = ref('')
const elapsedMs = ref(0)

const timeout = ref(3000)

const attachmentsModalOpen = ref(false)
const attachmentsList = ref<{ key: string; value: string }[]>([])

const isMeshSelected = computed(() => Boolean(meshStore.mesh))
const emptyDescription = computed(() => {
  if (!serviceName.value) {
    return '暂无方法'
  }
  if (!isMeshSelected.value) {
    return '请先选择 mesh'
  }
  return '暂无可调试方法'
})
const attachmentCount = computed(() => attachmentsList.value.filter((a) => a.key).length)
const providerInstanceOptions = computed(() =>
  providerInstances.value.map((instance) => ({
    label: instance.name,
    value: instance.name
  }))
)

function shortType(type: string): string {
  const parts = type.split('.')
  return parts[parts.length - 1]
}

function buildTypeMap(types: TypeDef[]): Record<string, TypeDef> {
  const map: Record<string, TypeDef> = Object.create(null)
  types?.forEach((t) => (map[t.type] = t))
  return map
}

function normalizeParameters(detail: MethodDetail): ParameterDef[] {
  return (detail.parameterTypes || []).map((type, index) => ({
    name: detail.parameters?.[index]?.name || `arg${index}`,
    type
  }))
}

function buildParamNodes(
  params: ParameterDef[],
  typeMap: Record<string, TypeDef>,
  prefix = ''
): any[] {
  return (params || []).map((param, i) => {
    const key = `${prefix}${i}`
    const typeDef = typeMap[param.type]
    const label = `${param.name}: ${shortType(param.type)}`
    if (typeDef) {
      if (typeDef.enums?.length > 0) {
        return {
          title: `${label} [${typeDef.enums.join(' | ')}]`,
          key
        }
      }
      const propEntries = Object.entries(typeDef.properties || {})
      if (propEntries.length > 0) {
        return {
          title: label,
          key,
          children: propEntries.map(([k, v], ci) => ({
            title: `${k}: ${shortType(v)}`,
            key: `${key}-${ci}`
          }))
        }
      }
    }
    return { title: label, key }
  })
}

function buildReturnTypeNodes(returnType: string, typeMap: Record<string, TypeDef>): any[] {
  if (!returnType || returnType === 'void') return []
  const typeDef = typeMap[returnType]
  const label = shortType(returnType)
  if (typeDef) {
    if (typeDef.enums?.length > 0) {
      return [{ title: `${label} [${typeDef.enums.join(' | ')}]`, key: '0' }]
    }
    const propEntries = Object.entries(typeDef.properties || {})
    if (propEntries.length > 0) {
      return [
        {
          title: label,
          key: '0',
          children: propEntries.map(([k, v], ci) => ({
            title: `${k}: ${shortType(v)}`,
            key: `0-${ci}`
          }))
        }
      ]
    }
  }
  return [{ title: label, key: '0' }]
}

const enterParamType = computed(() => {
  if (!currentMethodDetail.value) return []
  const typeMap = buildTypeMap(currentMethodDetail.value.types)
  return buildParamNodes(normalizeParameters(currentMethodDetail.value), typeMap)
})

const outputParamType = computed(() => {
  if (!currentMethodDetail.value) return []
  const typeMap = buildTypeMap(currentMethodDetail.value.types)
  return buildReturnTypeNodes(currentMethodDetail.value.returnType, typeMap)
})

function generateDefaultValue(
  type: string,
  typeMap: Record<string, TypeDef>,
  parents: string[] = []
): unknown {
  if (parents.includes(type)) return null
  if (type.startsWith('*') && !typeMap[type])
    return generateDefaultValue(type.slice(1), typeMap, parents)
  const name = type.replace(/^java\.lang\./, '')
  if (['long', 'Long', 'int64', 'uint64', 'uint', 'java.math.BigInteger'].includes(name)) return '0'
  if (['boolean', 'Boolean', 'bool'].includes(name)) return false
  if (['char', 'Character'].includes(name)) return 'a'
  if (['String', 'string'].includes(name)) return ''
  if (
    /^(u?int(8|16|32)?|rune|Integer|[Bb]yte|[Ss]hort|[Dd]ouble|[Ff]loat|float32|float64)$/.test(
      name
    )
  )
    return 0
  const def = typeMap[type]
  if (def?.enums?.length) return def.enums[0]
  if (
    type.startsWith('map[') ||
    type === 'java.util.Map' ||
    type === 'java.util.HashMap' ||
    def?.items?.length === 2
  )
    return {}
  if (
    type.startsWith('[') ||
    type.endsWith('[]') ||
    def?.items?.length === 1 ||
    ['java.util.List', 'java.util.Collection'].includes(type)
  )
    return []
  if (!def) return null
  return Object.fromEntries(
    Object.entries(def.properties || {}).map(([field, fieldType]) => [
      field,
      generateDefaultValue(fieldType, typeMap, [...parents, type])
    ])
  )
}

function generateRequestTemplate(detail: MethodDetail): string {
  const typeMap = buildTypeMap(detail.types)
  const args = normalizeParameters(detail).map((p) => generateDefaultValue(p.type, typeMap))
  return JSON.stringify(args, null, 2)
}

function syncSelectedInstance() {
  if (providerInstances.value.length === 0) {
    instanceName.value = ''
    return
  }
  if (providerInstances.value.some((instance) => instance.name === instanceName.value)) {
    return
  }
  instanceName.value = providerInstances.value[0].name
}

async function loadProviderInstances(page: number) {
  if (!serviceName.value || !isMeshSelected.value) {
    providerInstances.value = []
    instanceName.value = ''
    return
  }
  loadingProviders.value = true
  try {
    const res = await getServiceProviderInstancesAPI({
      serviceName: serviceName.value,
      group: group.value || undefined,
      version: version.value || undefined
    })
    if (page !== pageVersion) return
    providerInstances.value = Array.isArray(res.data) ? res.data : []
    syncSelectedInstance()
  } finally {
    if (page === pageVersion) loadingProviders.value = false
  }
}

async function loadMethods(page: number) {
  if (!serviceName.value || !isMeshSelected.value) {
    methodList.value = []
    currentMethodDetail.value = null
    requestValue.value = '[]'
    return
  }
  loadingMethods.value = true
  try {
    const res = await getServiceMethodsAPI({
      serviceName: serviceName.value,
      group: group.value || undefined,
      version: version.value || undefined
    })
    if (page !== pageVersion) return
    methodList.value = res.data || []
    if (methodList.value.length > 0) {
      activeKey.value = '0'
    } else {
      currentMethodDetail.value = null
      requestValue.value = '[]'
    }
  } finally {
    if (page === pageVersion) loadingMethods.value = false
  }
}

const selectedMethod = computed(() => methodList.value[Number(activeKey.value)])
const providerApp = computed(
  () => providerInstances.value.find((p) => p.name === instanceName.value)?.appName
)

watch([selectedMethod, providerApp], async ([method, app], _, onCleanup) => {
  let current = true
  onCleanup(() => {
    current = false
  })
  currentMethodDetail.value = null
  requestValue.value = '[]'
  responseValue.value = ''
  elapsedMs.value = 0
  loadingDetail.value = Boolean(method)
  if (!method) return
  try {
    const res = await getServiceMethodDetailAPI({
      serviceName: serviceName.value,
      methodName: method.methodName,
      group: group.value || undefined,
      version: version.value || undefined,
      providerAppName: app,
      signature: method.signature
    })
    if (current && res.data.signature === method.signature) {
      currentMethodDetail.value = res.data
      requestValue.value = generateRequestTemplate(res.data)
    }
  } catch (error) {
    if (current) responseValue.value = JSON.stringify(error, null, 2)
  } finally {
    if (current) loadingDetail.value = false
  }
})

async function handleInvoke() {
  const detail = currentMethodDetail.value
  if (!detail) return
  if (!meshStore.mesh) {
    message.warning('请先选择 mesh')
    responseValue.value = JSON.stringify(
      {
        error: 'missing_mesh',
        message: '请先选择 mesh'
      },
      null,
      2
    )
    return
  }
  if (!instanceName.value) {
    message.warning('请选择调用实例')
    responseValue.value = JSON.stringify(
      {
        error: 'missing_instance',
        message: '请选择调用实例'
      },
      null,
      2
    )
    return
  }
  const attachments: Record<string, string> = {}
  attachmentsList.value.forEach((a) => {
    if (a.key) attachments[a.key] = a.value
  })

  loadingInvoke.value = true
  responseValue.value = ''
  elapsedMs.value = 0
  try {
    const res = await serviceGenericInvokeAPI({
      mesh: meshStore.mesh,
      instanceName: instanceName.value,
      serviceName: serviceName.value,
      methodName: detail.methodName,
      signature: detail.signature,
      args: requestValue.value,
      group: group.value || undefined,
      version: version.value || undefined,
      timeoutMs: timeout.value > 0 ? timeout.value : undefined,
      attachments: Object.keys(attachments).length > 0 ? attachments : undefined
    })
    if (currentMethodDetail.value !== detail) return
    if (res.data && typeof res.data === 'object') {
      elapsedMs.value = res.data.elapsedMs || 0
      const rawResult = res.data.rawResult
      responseValue.value = JSON.stringify(rawResult, null, 2)
    } else {
      responseValue.value = JSON.stringify(res.data, null, 2)
    }
  } catch (e: any) {
    if (currentMethodDetail.value !== detail) return
    responseValue.value = JSON.stringify(
      e instanceof SyntaxError
        ? { error: 'invalid_request_json', message: e.message }
        : e || { error: '请求失败' },
      null,
      2
    )
  } finally {
    loadingInvoke.value = false
  }
}

function addAttachment() {
  attachmentsList.value.push({ key: '', value: '' })
}

function removeAttachment(idx: number) {
  attachmentsList.value.splice(idx, 1)
}

let pageVersion = 0
async function loadPageData() {
  const page = ++pageVersion
  responseValue.value = ''
  if (!isMeshSelected.value) {
    loadingMethods.value = false
    loadingProviders.value = false
    return
  }
  try {
    await Promise.all([loadProviderInstances(page), loadMethods(page)])
  } catch (error) {
    console.error('load debug page data failed', error)
  }
}

watch(
  [serviceName, group, version, () => meshStore.mesh],
  () => {
    methodList.value = []
    providerInstances.value = []
    currentMethodDetail.value = null
    instanceName.value = ''
    requestValue.value = '[]'
    responseValue.value = ''
    elapsedMs.value = 0
    void loadPageData()
  },
  { immediate: true }
)
</script>

<style lang="less" scoped>
.__container_services_tabs_debug {
  padding: 0;
  background: transparent;

  .tabs-title {
    width: 200px;
    font-size: 16px;
    margin-bottom: 8px;
    text-align: center;
  }

  :deep(.debug-tabs) {
    .ant-tabs-nav {
      width: 200px;
      background: #fafafa;
      border-radius: 4px;

      .ant-tabs-tab {
        margin: 0;
        padding: 12px 16px;
        font-weight: 400;
        transition: all 0.3s;

        &:hover {
          color: v-bind('PRIMARY_COLOR');
        }
      }

      .ant-tabs-tab-btn {
        white-space: normal;
        overflow-wrap: anywhere;
        text-align: left;
      }

      .ant-tabs-tab-active {
        background: v-bind('PRIMARY_COLOR');
        border-radius: 4px;

        .ant-tabs-tab-btn {
          color: v-bind('PRIMARY_COLOR_R') !important;
          font-weight: 600;
        }
      }

      .ant-tabs-ink-bar {
        display: none;
      }
    }

    .ant-tabs-content-holder {
      padding-left: 32px;
    }
  }

  .tab-content {
    min-height: 500px;
  }

  .section-title {
    margin-bottom: 12px;
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .empty-hint {
    font-size: 13px;
  }

  .editor-wrapper {
    position: relative;
    border: 1px solid #d9d9d9;
    border-radius: 6px;
    overflow: hidden;
    background-color: #fafafa;

    .monaco-container {
      padding-top: 4px;
    }

    .editor-tag {
      position: absolute;
      top: 8px;
      right: 8px;
      z-index: 10;
      pointer-events: none;
    }
  }

  .setting-item {
    display: flex;
    align-items: center;
    gap: 8px;

    .unit {
      color: rgba(0, 0, 0, 0.45);
    }
  }

  .action-row {
    display: flex;
    justify-content: center;
    margin-top: 40px;

    .invoke-btn {
      width: 200px;
      height: 40px;
      font-size: 14px;
    }
  }

  .attachment-edit-btn {
    padding: 0;
    color: v-bind('PRIMARY_COLOR') !important;

    &:hover {
      color: v-bind('PRIMARY_COLOR') !important;
      opacity: 0.85;
    }
  }

  :deep(.ant-tree) {
    background: transparent;
    .ant-tree-treenode {
      padding: 4px 0;
    }
  }
}

.attachments-list {
  .attachment-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;

    .kv-sep {
      color: rgba(0, 0, 0, 0.45);
    }
  }
}
</style>
