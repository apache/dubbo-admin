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
                  字段说明
                  <DoubleLeftOutlined v-if="!isDrawerOpened" />
                  <DoubleRightOutlined v-else />
                </a-button>
              </a-flex>
              <a-card title="基础信息" style="width: 100%" class="_detail">
                <a-form layout="horizontal">
                  <a-row style="width: 100%">
                    <a-col :span="12">
                      <a-form-item label="规则粒度" required>
                        <a-select
                          disabled
                          v-model:value="baseInfo.ruleGranularity"
                          style="width: 120px"
                          :options="ruleGranularityOptions"
                        ></a-select>
                      </a-form-item>
                      <a-form-item
                        v-if="baseInfo.ruleGranularity === 'service'"
                        label="版本"
                        required
                      >
                        <a-input v-model:value="baseInfo.version" style="width: 300px" disabled />
                      </a-form-item>
                      <a-form-item label="容错保护">
                        <a-switch
                          v-model:checked="baseInfo.faultTolerantProtection"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                      <a-form-item label="运行时生效">
                        <a-switch
                          v-model:checked="baseInfo.runtime"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                    </a-col>
                    <a-col :span="12">
                      <a-form-item label="作用对象" required>
                        <a-input
                          disabled
                          v-model:value="baseInfo.objectOfAction"
                          style="width: 300px"
                        />
                      </a-form-item>
                      <a-form-item
                        v-if="baseInfo.ruleGranularity === 'service'"
                        label="分组"
                        required
                      >
                        <a-input v-model:value="baseInfo.group" style="width: 300px" disabled />
                      </a-form-item>
                      <a-form-item label="立即启用">
                        <a-switch
                          v-model:checked="baseInfo.enable"
                          checked-children="开"
                          un-checked-children="关"
                        />
                      </a-form-item>
                      <a-form-item label="优先级">
                        <a-input-number v-model:value="baseInfo.priority" min="1" />
                      </a-form-item>
                    </a-col>
                  </a-row>
                </a-form>
              </a-card>
            </a-row>

            <a-card title="路由列表" style="width: 100%" class="_detail">
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
            <a-descriptions title="字段说明" :column="1">
              <a-descriptions-item label="key">
                作用对象<br />
                可能的值：Dubbo应用名或者服务名
              </a-descriptions-item>
              <a-descriptions-item label="scope">
                规则粒度<br />
                可能的值：application, service
              </a-descriptions-item>
              <a-descriptions-item label="force">
                容错保护<br />
                可能的值：true, false<br />
                描述：如果为true，则路由筛选后若没有可用的地址则会直接报异常；如果为false，则会从可用地址中选择完成RPC调用
              </a-descriptions-item>
              <a-descriptions-item label="runtime">
                运行时生效<br />
                可能的值：true, false<br />
                描述：如果为true，则该rule下的所有路由将会实时生效；若为false，则只有在启动时才会生效
              </a-descriptions-item>
            </a-descriptions>
          </div>
        </a-card>
      </a-col>
    </a-flex>
    <a-card class="footer">
      <a-flex>
        <a-button type="primary" :loading="loading" @click="updateRoutingRule">确认</a-button>
        <a-button style="margin-left: 30px" @click="console.log(routeList)"> 取消</a-button>
      </a-flex>
    </a-card>
  </div>
</template>

<script lang="ts" setup>
import {
  ComponentInternalInstance,
  getCurrentInstance,
  onMounted,
  reactive,
  ref,
  inject,
  watch
} from 'vue'
import { DoubleLeftOutlined, DoubleRightOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { useRoute } from 'vue-router'
import { getConditionRuleDetailAPI, updateConditionRuleAPI } from '@/api/service/traffic'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { isNil } from 'lodash'
import { HTTP_STATUS } from '@/base/http/constants'
import useRoutingRule from '../composables/useRoutingRule'
import RoutingRuleList from '../components/RoutingRuleList.vue'

const TAB_STATE = inject(PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE)
const loading = ref(false)

const routingRuleLogic = useRoutingRule()
const {
  routeList,
  mergeConditions,
  parseConditionMatchStringToArray,
  parseConditionToStringToArray
} = routingRuleLogic

onMounted(async () => {
  if (!isNil(TAB_STATE.conditionRule)) {
    const { enabled = true, key, scope, runtime = true, conditions } = TAB_STATE.conditionRule
    baseInfo.enable = enabled
    baseInfo.objectOfAction = key
    baseInfo.ruleGranularity = scope
    baseInfo.runtime = runtime

    // Clear and rebuild routeList based on conditions
    if (conditions && conditions.length > 0) {
      routeList.value = []
      conditions.forEach((item, index) => {
        // Add new route item for each condition
        routeList.value.push({
          selectedMatchConditionTypes: [],
          requestMatch: [],
          selectedRouteDistributeMatchTypes: [],
          routeDistribute: []
        })

        const conditionArr = item.split(' => ')
        const match = conditionArr[0]?.trim()
        const to = conditionArr[1]?.trim()
        routeList.value[index].requestMatch = parseConditionMatchStringToArray(match, index)
        routeList.value[index].routeDistribute = parseConditionToStringToArray(to, index)
      })
    }
  } else {
    await getRoutingRuleDetail()
  }
  getVersionAndGroup()
})
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()
const route = useRoute()

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

watch(baseInfo, (newVal) => {
  const { ruleGranularity, enable = true, runtime = true, objectOfAction } = newVal
  TAB_STATE.conditionRule = {
    ...TAB_STATE.conditionRule,
    enabled: enable,
    key: objectOfAction,
    runtime: runtime,
    scope: ruleGranularity
  }
})

// rule granularity options
const ruleGranularityOptions = ref([
  {
    label: '应用',
    value: 'application'
  },
  {
    label: '服务',
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
    TAB_STATE.conditionRule = {
      ...TAB_STATE.conditionRule,
      conditions: mergeConditions()
    }
  },
  {
    deep: true
  }
)

// Get condition routing details
async function getRoutingRuleDetail() {
  let res = await getConditionRuleDetailAPI(<string>route.params?.ruleName)
  // console.log(res)
  if (res?.code === HTTP_STATUS.SUCCESS) {
    console.log('res', res.data)
    const { conditions, configVersion, enabled, force, key, runtime, scope } = res?.data
    baseInfo.ruleGranularity = scope
    baseInfo.objectOfAction = key
    baseInfo.enable = enabled
    baseInfo.faultTolerantProtection = force
    baseInfo.runtime = runtime
    baseInfo.configVersion = configVersion

    //   format conditions data
    if (configVersion == 'v3.0' && conditions && conditions.length > 0) {
      // Clear and rebuild routeList based on conditions
      routeList.value = []
      conditions.forEach((item, index) => {
        // Add new route item for each condition
        routeList.value.push({
          selectedMatchConditionTypes: [],
          requestMatch: [],
          selectedRouteDistributeMatchTypes: [],
          routeDistribute: []
        })

        const conditionArr = item.split(' => ')
        const match = conditionArr[0]?.trim()
        const to = conditionArr[1]?.trim()
        // console.log('to', to)
        routeList.value[index].requestMatch = parseConditionMatchStringToArray(match, index)
        routeList.value[index].routeDistribute = parseConditionToStringToArray(to, index)
      })
    }
  }
}

const updateRoutingRule = async () => {
  loading.value = true
  try {
    const { ruleName } = route.params
    const { version, ruleGranularity, objectOfAction, enable, faultTolerantProtection, runtime } =
      baseInfo
    const data = {
      configVersion: 'v3.0',
      scope: ruleGranularity,
      key: objectOfAction,
      enabled: enable,
      force: faultTolerantProtection,
      runtime,
      conditions: mergeConditions()
    }
    const res = await updateConditionRuleAPI(<string>ruleName, data)
    if (res?.code === HTTP_STATUS.SUCCESS) {
      message.success('update success')
      // 延迟 2 秒后再获取数据，确保数据库已更新
      await new Promise((resolve) => setTimeout(resolve, 2000))
      TAB_STATE.conditionRule = null
      await getRoutingRuleDetail()
    }
  } finally {
    loading.value = false
  }
}

const getVersionAndGroup = () => {
  const conditionName = route.params?.ruleName
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
