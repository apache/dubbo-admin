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
      <a-col :span="24" class="left">
        <a-row>
          <a-flex justify="space-between" style="width: 100%">
            <a-typography-title :level="3"> 基础信息</a-typography-title>
            <a-space :size="8">
              <a-tag v-if="latestRecordedVersionNo !== undefined">
                {{
                  $t('ruleVersionDomain.latestRecordedVersionBadge', {
                    versionNo: latestRecordedVersionNo
                  })
                }}
              </a-tag>
              <a-button type="text" style="color: #0a90d5" @click="isHistoryOpen = true">
                {{ $t('flowControlDomain.versionRecords') }}
              </a-button>
            </a-space>
          </a-flex>
          <a-card class="_detail">
            <a-descriptions :column="2" layout="vertical" title="">
              <!-- ruleName -->
              <a-descriptions-item
                :label="$t('flowControlDomain.ruleName')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <p
                  class="description-item-content with-card"
                  @click="copyIt(conditionRuleDetail.key)"
                >
                  {{ conditionRuleDetail.key }}
                  <CopyOutlined />
                </p>
              </a-descriptions-item>

              <!-- ruleGranularity -->
              <a-descriptions-item
                :label="$t('flowControlDomain.ruleGranularity')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{ conditionRuleDetail.scope }}
                </a-typography-paragraph>
              </a-descriptions-item>
              <a-descriptions-item
                label="版本"
                :labelStyle="{ fontWeight: 'bold' }"
                v-if="conditionRuleDetail.scope == 'service'"
              >
                <p
                  class="description-item-content with-card"
                  @click="copyIt(conditionRuleDetail.version)"
                >
                  {{ conditionRuleDetail.version }}
                  <CopyOutlined v-if="conditionRuleDetail.version.length" />
                </p>
              </a-descriptions-item>

              <a-descriptions-item
                label="分组"
                :labelStyle="{ fontWeight: 'bold' }"
                v-if="conditionRuleDetail.scope == 'service'"
              >
                <p
                  class="description-item-content with-card"
                  @click="copyIt(conditionRuleDetail.group)"
                >
                  {{ conditionRuleDetail.group }}
                  <CopyOutlined v-if="conditionRuleDetail.group.length" />
                </p>
              </a-descriptions-item>

              <!-- actionObject -->
              <a-descriptions-item
                :label="$t('flowControlDomain.actionObject')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <p class="description-item-content with-card" @click="copyIt(actionObj)">
                  {{ actionObj }}
                  <CopyOutlined />
                </p>
              </a-descriptions-item>

              <!-- effectTime -->
              <!--          <a-descriptions-item-->
              <!--            :label="$t('flowControlDomain.effectTime')"-->
              <!--            :labelStyle="{ fontWeight: 'bold' }"-->
              <!--          >-->
              <!--            <a-typography-paragraph> 20230/12/19 22:09:34</a-typography-paragraph>-->
              <!--          </a-descriptions-item>-->

              <!-- faultTolerantProtection -->
              <a-descriptions-item
                :label="$t('flowControlDomain.faultTolerantProtection')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{
                    conditionRuleDetail.force
                      ? $t('flowControlDomain.opened')
                      : $t('flowControlDomain.closed')
                  }}
                </a-typography-paragraph>
              </a-descriptions-item>

              <!-- enabledState -->
              <a-descriptions-item
                :label="$t('flowControlDomain.enabledState')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{
                    conditionRuleDetail.enabled
                      ? $t('flowControlDomain.enabled')
                      : $t('flowControlDomain.disabled')
                  }}
                </a-typography-paragraph>
              </a-descriptions-item>

              <!-- runTimeEffective -->
              <a-descriptions-item
                :label="$t('flowControlDomain.runTimeEffective')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{
                    conditionRuleDetail.runtime
                      ? $t('flowControlDomain.opened')
                      : $t('flowControlDomain.closed')
                  }}
                </a-typography-paragraph>
              </a-descriptions-item>

              <!-- priority -->
              <!--          <a-descriptions-item-->
              <!--            :label="$t('flowControlDomain.priority')"-->
              <!--            :labelStyle="{ fontWeight: 'bold' }"-->
              <!--          >-->
              <!--            <a-typography-paragraph>-->
              <!--              {{ $t('flowControlDomain.notSet') }}-->
              <!--            </a-typography-paragraph>-->
              <!--          </a-descriptions-item>-->
            </a-descriptions>
          </a-card>
        </a-row>

        <StructuredConditionRuleList
          v-if="conditionRuleDetail.configVersion === 'v3.1'"
          v-model="structuredConditions"
          readonly
          style="margin-top: 10px"
        />
        <a-card v-else style="margin-top: 10px" class="_detail">
          <a-space align="start" style="width: 100%">
            <a-typography-title :level="5"
              >{{ $t('flowControlDomain.requestParameterMatching') }}:
            </a-typography-title>

            <a-space align="center" direction="horizontal" size="middle" wrap>
              <a-tag v-for="(item, index) in requestParameterMatch" :key="index" color="#2db7f5">
                {{ item }}
              </a-tag>
            </a-space>
          </a-space>

          <a-space align="start" style="width: 100%" wrap>
            <a-typography-title :level="5"
              >{{ $t('flowControlDomain.addressSubsetMatching') }}:
            </a-typography-title>
            <a-tag v-for="(item, index) in addressSubsetMatch" :key="index" color="#87d068">
              {{ item }}
            </a-tag>
          </a-space>
        </a-card>
      </a-col>
    </a-flex>

    <RuleHistoryPanel
      v-model:open="isHistoryOpen"
      :title="conditionRuleDetail.key || ruleName"
      kind="condition-rule"
      :rule-name="ruleName"
      @latest-recorded-version-no-change="latestRecordedVersionNo = $event"
    />
  </div>
</template>

<script lang="ts" setup>
import {
  type ComponentInternalInstance,
  computed,
  getCurrentInstance,
  onMounted,
  reactive,
  ref
} from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { getConditionRuleDetailAPI } from '@/api/service/traffic'
import { useRoute } from 'vue-router'
import { HTTP_STATUS } from '@/base/http/constants'
import RuleHistoryPanel from '../../_shared/RuleHistoryPanel.vue'
import StructuredConditionRuleList from '../components/StructuredConditionRuleList.vue'
import {
  normalizeStructuredConditions,
  type StructuredConditionRule
} from '../model/ConditionRuleModel'

interface ConditionRuleDetail {
  configVersion: string
  key: string
  scope: string
  version: string
  group: string
  force?: boolean
  enabled?: boolean
  runtime?: boolean
  conditions: any[]
}

const {
  appContext: {
    config: { globalProperties }
  }
} = getCurrentInstance() as ComponentInternalInstance
const route = useRoute()
const ruleName = computed(() => String(route.params?.ruleName || ''))

const isHistoryOpen = ref(false)
const latestRecordedVersionNo = ref<number | undefined>(undefined)

const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

// Condition routing details
const conditionRuleDetail = reactive<ConditionRuleDetail>({
  configVersion: '',
  key: '',
  scope: '',
  version: '',
  group: '',
  conditions: []
})
const structuredConditions = ref<StructuredConditionRule[]>([])

const actionObj = computed(() => {
  const key = conditionRuleDetail.key || ''
  const arr = typeof key === 'string' ? key.split(':') : []
  return arr[0] || ''
})

// Request parameter matching
const requestParameterMatch = ref<string[]>([])

// Address subset matching
const addressSubsetMatch = ref<string[]>([])

// Get condition routing details
async function getRoutingRuleDetail() {
  let res = await getConditionRuleDetailAPI(ruleName.value)
  if (res?.code === HTTP_STATUS.SUCCESS) {
    Object.assign(conditionRuleDetail, res?.data || {})

    requestParameterMatch.value = []
    addressSubsetMatch.value = []
    if (conditionRuleDetail.configVersion === 'v3.1') {
      structuredConditions.value = normalizeStructuredConditions(conditionRuleDetail.conditions)
      return
    }
    conditionRuleDetail.conditions.forEach((item: any) => {
      const arr = item.split(' => ')
      const addressArr = arr[1]?.split(' & ')
      const requestMatchArr = arr[0]?.split(' & ')
      requestParameterMatch.value = requestParameterMatch.value.concat(requestMatchArr)
      addressSubsetMatch.value = addressSubsetMatch.value.concat(addressArr)
    })
  }
}

const getVersionAndGroup = () => {
  const conditionName = ruleName.value
  if (conditionName && conditionRuleDetail.scope === 'service') {
    const arr = conditionName?.split(':')
    conditionRuleDetail.version = arr[1] || ''
    conditionRuleDetail.group = arr[2]?.split('.')[0] || ''
  }
}

onMounted(async () => {
  await getRoutingRuleDetail()
  getVersionAndGroup()
})
</script>

<style lang="less" scoped>
.__container_routingRule_detail {
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
