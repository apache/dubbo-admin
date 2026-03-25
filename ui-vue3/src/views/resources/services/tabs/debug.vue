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
      <div class="tabs-title">方法列表</div>
      <a-spin :spinning="loadingMethods">
        <a-empty v-if="!loadingMethods && methodList.length === 0" description="暂无方法" />
        <a-tabs
          v-else
          v-model:activeKey="activeKey"
          tab-position="left"
          class="debug-tabs"
          @change="onTabChange"
        >
          <a-tab-pane
            v-for="(method, index) in methodList"
            :key="String(index)"
            :tab="method.methodName"
          >
            <a-spin :spinning="loadingDetail">
              <div class="tab-content">
                <a-row :gutter="[24, 24]">
                  <!-- Row 1: Parameter Types -->
                  <a-col :span="12">
                    <div class="section-title">入参类型:</div>
                    <a-tree
                      v-if="enterParamType.length > 0"
                      block-node
                      :tree-data="enterParamType"
                      default-expand-all
                    />
                    <span v-else class="empty-hint">无入参</span>
                  </a-col>
                  <a-col :span="12">
                    <div class="section-title">出参类型:</div>
                    <a-tree
                      v-if="outputParamType.length > 0"
                      block-node
                      :tree-data="outputParamType"
                      default-expand-all
                    />
                    <span v-else class="empty-hint">无出参</span>
                  </a-col>

                  <!-- Row 2: Request & Response Editors -->
                  <a-col :span="12">
                    <div class="section-title">请求:</div>
                    <div class="editor-wrapper">
                      <monaco-editor
                        v-model="requestValue"
                        :editor-id="`requestEditor-${index}`"
                        height="300px"
                      />
                      <div class="editor-tag">JSON</div>
                    </div>
                  </a-col>
                  <a-col :span="12">
                    <div class="section-title">响应:</div>
                    <div class="editor-wrapper">
                      <monaco-editor
                        v-model="responseValue"
                        :editor-id="`responseEditor-${index}`"
                        height="300px"
                        :readonly="true"
                      />
                      <div class="editor-tag">JSON</div>
                    </div>
                  </a-col>

                  <!-- Row 3: Bottom Settings -->
                  <a-col :span="8">
                    <div class="section-title">调用实例:</div>
                    <a-select
                      v-model:value="instanceName"
                      :options="providerInstanceOptions"
                      :loading="loadingProviders"
                      :disabled="providerInstanceOptions.length === 0"
                      placeholder="请选择真实调用实例"
                      show-search
                      option-filter-prop="label"
                      allow-clear
                    />
                    <div
                      v-if="!loadingProviders && providerInstanceOptions.length === 0"
                      class="empty-hint"
                    >
                      当前服务没有可调用实例
                    </div>
                  </a-col>
                  <a-col :span="8">
                    <div class="section-title">自定义超时时间</div>
                    <div class="setting-item">
                      <a-input-number v-model:value="timeout" :min="0" style="width: 120px" />
                      <span class="unit">ms</span>
                    </div>
                  </a-col>
                  <a-col :span="8">
                    <div class="section-title">传递attachments</div>
                    <div class="setting-item">
                      <a-button type="link" @click="attachmentsModalOpen = true" style="padding: 0">
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
                      :disabled="!instanceName"
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
          <minus-circle-outlined class="remove-icon" @click="removeAttachment(idx)" />
        </div>
        <a-button type="dashed" block @click="addAttachment"> <plus-outlined /> 添加 </a-button>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import MonacoEditor from '@/components/editor/MonacoEditor.vue'
import { EditOutlined, MinusCircleOutlined, PlusOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import {
  getServiceProviderInstancesAPI,
  getServiceMethodsAPI,
  getServiceMethodDetailAPI,
  serviceGenericInvokeAPI
} from '@/api/service/service'
import { useMeshStore } from '@/stores/mesh'

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

const timeout = ref(3000)

const attachmentsModalOpen = ref(false)
const attachmentsList = ref<{ key: string; value: string }[]>([])

const attachmentCount = computed(() => attachmentsList.value.filter((a) => a.key).length)
const providerInstanceOptions = computed(() =>
  providerInstances.value.map((instance) => ({
    label: `${instance.appName || 'UnknownApp'} | ${instance.ip || instance.name}`,
    value: instance.name
  }))
)

function shortType(type: string): string {
  const parts = type.split('.')
  return parts[parts.length - 1]
}

function buildTypeMap(types: TypeDef[]): Record<string, TypeDef> {
  const map: Record<string, TypeDef> = {}
  types?.forEach((t) => (map[t.type] = t))
  return map
}

function normalizeParameters(detail: MethodDetail): ParameterDef[] {
  const explicitParameters = (detail.parameters || [])
    .map((param, index) => ({
      name: param.name || `arg${index}`,
      type: param.type || detail.parameterTypes?.[index] || ''
    }))
    .filter((param) => Boolean(param.type))

  if (explicitParameters.length > 0) {
    return explicitParameters
  }

  return (detail.parameterTypes || []).map((type, index) => ({
    name: `arg${index}`,
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

function generateDefaultValue(type: string, typeMap: Record<string, TypeDef>, depth = 0): any {
  if (depth > 5) return null
  const primitives: Record<string, any> = {
    'java.lang.String': '',
    String: '',
    int: 0,
    'java.lang.Integer': 0,
    long: 0,
    'java.lang.Long': 0,
    double: 0.0,
    'java.lang.Double': 0.0,
    float: 0.0,
    'java.lang.Float': 0.0,
    boolean: false,
    'java.lang.Boolean': false,
    short: 0,
    'java.lang.Short': 0,
    byte: 0,
    'java.lang.Byte': 0,
    char: '',
    'java.lang.Character': ''
  }
  if (type in primitives) return primitives[type]

  const typeDef = typeMap[type]
  if (!typeDef) return null
  if (typeDef.items?.length > 0) return []
  if (typeDef.enums?.length > 0) return typeDef.enums[0]
  const obj: Record<string, any> = {}
  Object.entries(typeDef.properties || {}).forEach(([k, v]) => {
    obj[k] = generateDefaultValue(v, typeMap, depth + 1)
  })
  return obj
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

async function loadProviderInstances() {
  if (!serviceName.value) {
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
    providerInstances.value = Array.isArray(res.data) ? res.data : []
    syncSelectedInstance()
  } finally {
    loadingProviders.value = false
  }
}

async function loadMethods() {
  if (!serviceName.value) {
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
    methodList.value = res.data || []
    if (methodList.value.length > 0) {
      activeKey.value = '0'
      await loadMethodDetail(methodList.value[0])
    } else {
      currentMethodDetail.value = null
      requestValue.value = '[]'
    }
  } finally {
    loadingMethods.value = false
  }
}

async function loadMethodDetail(method: MethodSummary) {
  loadingDetail.value = true
  responseValue.value = ''
  try {
    const res = await getServiceMethodDetailAPI({
      serviceName: serviceName.value,
      methodName: method.methodName,
      group: group.value || undefined,
      version: version.value || undefined,
      signature: method.signature || undefined
    })
    currentMethodDetail.value = res.data
    requestValue.value = generateRequestTemplate(res.data)
  } finally {
    loadingDetail.value = false
  }
}

async function onTabChange(key: string) {
  const index = Number(key)
  const method = methodList.value[index]
  if (method) {
    await loadMethodDetail(method)
  }
}

async function handleInvoke() {
  if (!currentMethodDetail.value) return
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
  let args: any[]
  try {
    args = JSON.parse(requestValue.value)
    if (!Array.isArray(args)) {
      args = [args]
    }
  } catch (error: any) {
    message.error('请求参数不是有效 JSON')
    responseValue.value = JSON.stringify(
      {
        error: 'invalid_request_json',
        message: error?.message || '请求参数不是有效 JSON'
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
  try {
    const res = await serviceGenericInvokeAPI({
      mesh: meshStore.mesh,
      instanceName: instanceName.value,
      serviceName: serviceName.value,
      methodName: currentMethodDetail.value.methodName,
      signature: currentMethodDetail.value.signature,
      args,
      group: group.value || undefined,
      version: version.value || undefined,
      timeoutMs: timeout.value > 0 ? timeout.value : undefined,
      attachments: Object.keys(attachments).length > 0 ? attachments : undefined
    })
    responseValue.value = JSON.stringify(res.data, null, 2)
  } catch (e: any) {
    responseValue.value = JSON.stringify(e || { error: '请求失败' }, null, 2)
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

async function loadPageData() {
  responseValue.value = ''
  await Promise.all([loadProviderInstances(), loadMethods()])
}

watch(
  [serviceName, group, version, () => meshStore.mesh],
  () => {
    methodList.value = []
    providerInstances.value = []
    currentMethodDetail.value = null
    instanceName.value = ''
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
    font-weight: bold;
    font-size: 16px;
    margin-bottom: 8px;
    text-align: center;
    color: rgba(0, 0, 0, 0.85);
  }

  :deep(.debug-tabs) {
    .ant-tabs-nav {
      width: 200px;
      background: #fafafa;
      border-radius: 4px;

      .ant-tabs-tab {
        margin: 0;
        padding: 12px 16px;
        transition: all 0.3s;

        &:hover {
          color: v-bind('PRIMARY_COLOR');
        }
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
    font-weight: 500;
    margin-bottom: 12px;
    font-size: 14px;
    color: rgba(0, 0, 0, 0.85);
  }

  .empty-hint {
    color: rgba(0, 0, 0, 0.45);
    font-size: 13px;
  }

  .editor-wrapper {
    position: relative;
    border: 1px solid #d9d9d9;
    border-radius: 2px;
    overflow: hidden;

    .editor-tag {
      position: absolute;
      top: 4px;
      right: 8px;
      background: #f5f5f5;
      border: 1px solid #d9d9d9;
      border-radius: 2px;
      padding: 0 4px;
      font-size: 12px;
      color: #666;
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

    .remove-icon {
      color: #ff4d4f;
      cursor: pointer;
      flex-shrink: 0;
    }
  }
}
</style>
