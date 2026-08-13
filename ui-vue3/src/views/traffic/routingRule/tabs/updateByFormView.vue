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
  <div class="__container_routingRule_detail">
    <a-flex style="width: 100%">
      <a-col :span="isDrawerOpened ? 24 - sliderSpan : 24" class="left">
        <a-card>
          <a-space style="width: 100%" direction="vertical" size="middle">
            <a-row>
              <a-flex justify="end" style="width: 100%">
                <a-button
                  type="text"
                  style="color: #0a90d5"
                  @click="isDrawerOpened = !isDrawerOpened"
                >
                  {{ t('routingRuleDomain.fieldDesc') }}
                  <DoubleLeftOutlined v-if="!isDrawerOpened" />
                  <DoubleRightOutlined v-else />
                </a-button>
              </a-flex>
              <a-card :title="t('basicInfo')" style="width: 100%" class="_detail">
                <a-form layout="horizontal">
                  <a-row style="width: 100%">
                    <a-col :span="12">
                      <a-form-item :label="t('routingRuleDomain.ruleGranularity')" required>
                        <a-select
                          disabled
                          v-model:value="baseInfo.ruleGranularity"
                          style="width: 120px"
                          :options="ruleGranularityOptions"
                        ></a-select>
                      </a-form-item>
                      <a-form-item
                        v-if="baseInfo.ruleGranularity === 'service'"
                        :label="t('routingRuleDomain.version')"
                        required
                      >
                        <a-input v-model:value="baseInfo.version" style="width: 300px" disabled />
                      </a-form-item>
                      <a-form-item :label="t('routingRuleDomain.force')">
                        <a-switch
                          v-model:checked="baseInfo.faultTolerantProtection"
                          :checked-children="t('flowControlDomain.on')"
                          :un-checked-children="t('flowControlDomain.off')"
                        />
                      </a-form-item>
                      <a-form-item :label="t('routingRuleDomain.runtime')">
                        <a-switch
                          v-model:checked="baseInfo.runtime"
                          :checked-children="t('flowControlDomain.on')"
                          :un-checked-children="t('flowControlDomain.off')"
                        />
                      </a-form-item>
                      <a-form-item :label="t('routingRuleDomain.configVersion')">
                        <a-input :value="baseInfo.configVersion" disabled style="width: 120px" />
                      </a-form-item>
                    </a-col>
                    <a-col :span="12">
                      <a-form-item :label="t('routingRuleDomain.objectOfAction')" required>
                        <a-input
                          disabled
                          v-model:value="baseInfo.objectOfAction"
                          style="width: 300px"
                        />
                      </a-form-item>
                      <a-form-item
                        v-if="baseInfo.ruleGranularity === 'service'"
                        :label="t('routingRuleDomain.group')"
                        required
                      >
                        <a-input v-model:value="baseInfo.group" style="width: 300px" disabled />
                      </a-form-item>
                      <a-form-item :label="t('routingRuleDomain.enabled')">
                        <a-switch
                          v-model:checked="baseInfo.enable"
                          :checked-children="t('flowControlDomain.on')"
                          :un-checked-children="t('flowControlDomain.off')"
                        />
                      </a-form-item>
                      <a-form-item :label="t('routingRuleDomain.priority')">
                        <a-input-number v-model:value="baseInfo.priority" min="1" />
                      </a-form-item>
                    </a-col>
                  </a-row>
                </a-form>
              </a-card>
            </a-row>

            <a-card :title="t('routingRuleDomain.routeList')" style="width: 100%" class="_detail">
              <StructuredConditionRuleList
                v-if="baseInfo.configVersion === 'v3.1'"
                v-model="structuredConditions"
              />
              <RoutingRuleList
                v-else
                :routeList="routeList"
                :baseInfo="baseInfo"
                :routingRuleLogic="routingRuleLogic"
              />
            </a-card>
          </a-space>
        </a-card>
      </a-col>

      <a-col :span="isDrawerOpened ? sliderSpan : 0" class="right">
        <a-card v-if="isDrawerOpened" class="sliderBox">
          <div>
            <a-descriptions :title="t('routingRuleDomain.fieldDesc')" :column="1">
              <a-descriptions-item label="key">
                <span v-html="t('routingRuleDomain.desc.objectOfAction')"></span>
              </a-descriptions-item>
              <a-descriptions-item label="scope">
                <span v-html="t('routingRuleDomain.desc.ruleGranularity')"></span>
              </a-descriptions-item>
              <a-descriptions-item label="force">
                <span v-html="t('routingRuleDomain.desc.force')"></span>
              </a-descriptions-item>
              <a-descriptions-item label="runtime">
                <span v-html="t('routingRuleDomain.desc.runtime')"></span>
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </a-card>
      </a-col>
    </a-flex>
    <a-card class="footer">
      <a-flex>
        <a-button type="primary" :loading="loading" @click="updateRoutingRule">{{
          t('confirm')
        }}</a-button>
        <a-button style="margin-left: 30px" @click="console.log(routeList)">
          {{ t('cancel') }}</a-button
        >
      </a-flex>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, inject, watch } from 'vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useRoute } from 'vue-router'
import { getConditionRuleDetailAPI, updateConditionRuleAPI } from '@/api/service/traffic'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { HTTP_STATUS } from '@/base/http/constants'
import useRoutingRule from '../composables/useRoutingRule'
import RoutingRuleList from '../components/RoutingRuleList.vue'
import StructuredConditionRuleList from '../components/StructuredConditionRuleList.vue'
import {
  isCompleteConditionRule,
  normalizeStructuredConditions,
  type StructuredConditionRule
} from '../model/ConditionRuleModel'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const TAB_STATE = inject(PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE) as any
const loading = ref(false)

const routingRuleLogic = useRoutingRule()
const {
  routeList,
  mergeConditions,
  parseConditionMatchStringToArray,
  parseConditionToStringToArray
} = routingRuleLogic

const structuredConditions = ref<StructuredConditionRule[]>([])
const initialized = ref(false)

onMounted(async () => {
  if (isCompleteConditionRule(TAB_STATE.conditionRule)) {
    loadConditionRule(TAB_STATE.conditionRule)
  } else {
    await getRoutingRuleDetail()
  }
  getVersionAndGroup()
  initialized.value = true
  syncConditionRuleDraft()
})

const route = useRoute()

const isDrawerOpened = ref(false)

const sliderSpan = ref(8)

// base info
const baseInfo = reactive({
  version: '',
  ruleGranularity: '',
  objectOfAction: '',
  enable: true,
  faultTolerantProtection: false,
  runtime: true,
  priority: null,
  configVersion: '',
  group: ''
})

const loadConditionRule = (data: Record<string, any>) => {
  const {
    conditions = [],
    configVersion = 'v3.0',
    priority,
    enabled = true,
    force = false,
    key = '',
    runtime = true,
    scope = ''
  } = data
  const keyParts = String(key).split(':')
  baseInfo.configVersion = configVersion
  baseInfo.priority = priority
  baseInfo.enable = enabled
  baseInfo.faultTolerantProtection = force
  baseInfo.objectOfAction = scope === 'service' ? keyParts[0] : key
  baseInfo.ruleGranularity = scope
  baseInfo.runtime = runtime
  if (scope === 'service') {
    baseInfo.version = keyParts[1] || ''
    baseInfo.group = keyParts[2] || ''
  }

  if (configVersion === 'v3.1') {
    structuredConditions.value = normalizeStructuredConditions(conditions)
    return
  }

  routeList.value = []
  conditions.forEach((item: string, index: number) => {
    routeList.value.push({
      selectedMatchConditionTypes: [],
      requestMatch: [],
      selectedRouteDistributeMatchTypes: [],
      routeDistribute: []
    })
    const [match, to] = item.split(' => ')
    routeList.value[index].requestMatch = parseConditionMatchStringToArray(match?.trim(), index)
    routeList.value[index].routeDistribute = parseConditionToStringToArray(to?.trim(), index)
  })
}

const currentConditions = () =>
  baseInfo.configVersion === 'v3.1' ? structuredConditions.value : mergeConditions()

const currentRuleKey = () =>
  baseInfo.ruleGranularity === 'service' && baseInfo.version && baseInfo.group
    ? `${baseInfo.objectOfAction}:${baseInfo.version}:${baseInfo.group}`
    : baseInfo.objectOfAction

const syncConditionRuleDraft = () => {
  if (!initialized.value) {
    return
  }
  TAB_STATE.conditionRule = {
    configVersion: baseInfo.configVersion || 'v3.0',
    priority: baseInfo.priority,
    enabled: baseInfo.enable,
    force: baseInfo.faultTolerantProtection,
    key: currentRuleKey(),
    runtime: baseInfo.runtime,
    scope: baseInfo.ruleGranularity,
    conditions: currentConditions()
  }
}

watch(baseInfo, syncConditionRuleDraft)

// rule granularity options
// rule granularity options
import { computed } from 'vue'
const ruleGranularityOptions = computed(() => [
  {
    label: t('routingRuleDomain.application'),
    value: 'application'
  },
  {
    label: t('routingRuleDomain.service'),
    value: 'service'
  }
])

watch(routeList, syncConditionRuleDraft, {
  deep: true
})
watch(structuredConditions, syncConditionRuleDraft, { deep: true })

// Get condition routing details
async function getRoutingRuleDetail() {
  let res = await getConditionRuleDetailAPI(route.params?.ruleName as string)
  if (res?.code === HTTP_STATUS.SUCCESS) {
    loadConditionRule(res.data || {})
  }
}

const updateRoutingRule = async () => {
  loading.value = true
  try {
    const { ruleName } = route.params
    const { ruleGranularity, enable, faultTolerantProtection, runtime, priority, configVersion } =
      baseInfo
    const data = {
      configVersion: configVersion || 'v3.0',
      priority,
      scope: ruleGranularity,
      key: currentRuleKey(),
      enabled: enable,
      force: faultTolerantProtection,
      runtime,
      conditions: currentConditions()
    }
    const res = await updateConditionRuleAPI(ruleName as string, data)
    if (res?.code === HTTP_STATUS.SUCCESS) {
      message.success('update success')
      TAB_STATE.conditionRule = null
      await getRoutingRuleDetail()
    }
  } catch (e: any) {
    message.error(e?.message || String(e))
  } finally {
    loading.value = false
  }
}

const getVersionAndGroup = () => {
  const conditionName = String(route.params?.ruleName || '')
  // console.log('lll', baseInfo)
  if (conditionName && baseInfo.ruleGranularity === 'service') {
    const arr = conditionName.split(':')
    if (arr.length >= 3) {
      baseInfo.version = arr[1]
      baseInfo.group = arr[2].split('.')[0]
    } else {
      // Handle case where conditionName doesn't have expected format
      console.warn(
        `Invalid conditionName format: ${conditionName}. Expected format: 'service:version:group'`
      )
      baseInfo.version = ''
      baseInfo.group = ''
    }
  }
}
</script>

<style lang="less" scoped>
.__container_routingRule_detail {
  overflow: auto;
  max-height: calc(100vh - 200px);

  &::-webkit-scrollbar {
    display: none;
  }

  .action-icon {
    font-size: 17px;
    margin-left: 10px;
    cursor: pointer;
  }

  .match-condition-type-label {
    min-width: 100px;
    text-align: center;
  }

  .bottom-action-footer {
    width: 100%;
    background-color: white;
    height: 50px;
    display: flex;
    align-items: center;
    padding-left: 20px;
    box-shadow: 0 -2px 4px rgba(0, 0, 0, 0.1);
    /* 添加顶部阴影 */
  }

  .sliderBox {
    margin-left: 5px;
    max-height: 530px;
    overflow: auto;
  }

  &:deep(.left.ant-col) {
    transition: all 0.5s ease;
  }

  &:deep(.right.ant-col) {
    transition: all 0.5s ease;
  }
}
</style>
