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
                        <a-input v-model:value="baseInfo.version" style="width: 300px" />
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
                    </a-col>
                    <a-col :span="12">
                      <a-form-item :label="t('routingRuleDomain.objectOfAction')" required>
                        <a-input v-model:value="baseInfo.objectOfAction" style="width: 300px" />
                      </a-form-item>
                      <a-form-item
                        v-if="baseInfo.ruleGranularity === 'service'"
                        :label="t('routingRuleDomain.group')"
                        required
                      >
                        <a-input v-model:value="baseInfo.group" style="width: 300px" />
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
              <RoutingRuleList
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
        <a-button type="primary" @click="addRoutingRule">{{ t('confirm') }}</a-button>
      </a-flex>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import {
  type ComponentInternalInstance,
  getCurrentInstance,
  reactive,
  ref,
  inject,
  onMounted,
  watch
} from 'vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { useRouter } from 'vue-router'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { addConditionRuleAPI } from '@/api/service/traffic'
import { isNil } from 'lodash'
import { HTTP_STATUS } from '@/base/http/constants'
import useRoutingRule from '../composables/useRoutingRule'
import RoutingRuleList from '../components/RoutingRuleList.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const TAB_STATE = inject(PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE) as any

const routingRuleLogic = useRoutingRule()
const {
  routeList,
  mergeConditions,
  parseConditionMatchStringToArray,
  parseConditionToStringToArray
} = routingRuleLogic

onMounted(() => {
  if (!isNil(TAB_STATE.conditionRule)) {
    const { enabled = true, key, scope, runtime = true, conditions } = TAB_STATE.conditionRule
    baseInfo.enable = enabled
    baseInfo.objectOfAction = key
    baseInfo.ruleGranularity = scope
    baseInfo.runtime = runtime

    conditions &&
      conditions.length &&
      conditions.forEach((item: string, index: number) => {
        const conditionArr = item.split(' => ')
        const match = conditionArr[0]?.trim()
        const to = conditionArr[1]?.trim()
        routeList.value[index].requestMatch = parseConditionMatchStringToArray(match, index)
        routeList.value[index].routeDistribute = parseConditionToStringToArray(to, index)
      })
  }

  if (!isNil(TAB_STATE.addConditionRuleSate)) {
    const { version, group } = TAB_STATE.addConditionRuleSate
    baseInfo.version = version
    baseInfo.group = group
  }
})
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()
const router = useRouter()

const isDrawerOpened = ref(false)

const sliderSpan = ref(8)

let __ = PRIMARY_COLOR

const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

// base info
const baseInfo = reactive({
  version: '',
  ruleGranularity: '',
  objectOfAction: '',
  enable: true,
  faultTolerantProtection: false,
  runtime: true,
  priority: null,
  group: ''
})

watch(
  baseInfo,
  (newVal) => {
    const { ruleGranularity, enable = true, runtime = true, objectOfAction } = newVal
    TAB_STATE.conditionRule = {
      ...TAB_STATE.conditionRule,
      enabled: enable,
      key: objectOfAction,
      runtime: runtime,
      scope: ruleGranularity
    }
    TAB_STATE.addConditionRuleSate = {
      version: newVal.version,
      group: newVal.group
    }
  },
  {
    immediate: isNil(TAB_STATE.conditionRule) ? true : false
  }
)

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

enum ruleGranularityEnum {
  application = '应用',
  service = '服务'
}

watch(
  routeList,
  (newVal) => {
    if (TAB_STATE && TAB_STATE.conditionRule) {
      TAB_STATE.conditionRule = {
        ...TAB_STATE.conditionRule,
        conditions: mergeConditions()
      }
    }
  },
  {
    deep: true
  }
)

const addRoutingRule = async () => {
  // Logic for adding routing rule
  // Copied from original file, but making sure it uses mergeConditions
  const {
    version,
    ruleGranularity,
    objectOfAction,
    enable,
    faultTolerantProtection,
    runtime,
    group
  } = baseInfo

  let ruleName =
    ruleGranularity === 'service'
      ? `${objectOfAction}:${version}:${group}.condition-router`
      : `${objectOfAction}.condition-router`

  const data = {
    configVersion: 'v3.0',
    scope: ruleGranularity,
    key: objectOfAction,
    enabled: enable,
    force: faultTolerantProtection,
    runtime,
    conditions: mergeConditions()
  }
  const res = await addConditionRuleAPI(ruleName, data)
  if (res?.code === HTTP_STATUS.SUCCESS) {
    message.success('add success')
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
