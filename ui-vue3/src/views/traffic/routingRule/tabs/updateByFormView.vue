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
              <a-card v-for="(routeItem, routeItemIndex) in routeList">
                <template #title>
                  <a-flex justify="space-between">
                    <a-space align="center">
                      <div>路由【{{ routeItemIndex + 1 }}】</div>
                      <a-tooltip>
                        <template #title>{{ routeItemDes(routeItemIndex) }}</template>
                        <div
                          style="
                            max-width: 400px;
                            overflow: hidden;
                            text-overflow: ellipsis;
                            white-space: nowrap;
                          "
                        >
                          {{ routeItemDes(routeItemIndex) }}
                        </div>
                      </a-tooltip>
                    </a-space>
                    <Icon
                      @click="deleteRoute(routeItemIndex)"
                      class="action-icon"
                      icon="tdesign:delete"
                    />
                  </a-flex>
                </template>

                <a-form layout="horizontal">
                  <a-space style="width: 100%" direction="vertical" size="large">
                    <a-form-item label="请求匹配">
                      <a-card v-if="routeItem.requestMatch.length > 0">
                        <a-space style="width: 100%" direction="vertical" size="small">
                          <a-flex align="center" justify="space-between">
                            <a-form-item label="匹配条件类型">
                              <a-select
                                v-model:value="routeItem.selectedMatchConditionTypes"
                                :options="matchConditionTypeOptions"
                                mode="multiple"
                                style="min-width: 200px"
                              />
                            </a-form-item>
                            <Icon
                              @click="deleteRequestMatch(routeItemIndex)"
                              class="action-icon"
                              icon="tdesign:delete"
                            />
                          </a-flex>
                          <template
                            v-for="(conditionItem, conditionItemIndex) in routeItem.requestMatch"
                          >
                            <!--                        host-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('host') &&
                                conditionItem.type === 'host'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input
                                v-model:value="conditionItem.value"
                                placeholder="请求来源ip"
                              />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--application-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('application') &&
                                conditionItem.type === 'application'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input
                                v-model:value="conditionItem.value"
                                placeholder="请求来源应用名"
                              />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--                      method-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('method') &&
                                conditionItem.type === 'method'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input v-model:value="conditionItem.value" placeholder="方法值" />

                              <Icon
                                @click="
                                  deleteMatchConditionTypeItem(conditionItem?.type, routeItemIndex)
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>
                            <!--                      arguments-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('arguments') &&
                                conditionItem.type === 'arguments'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-space direction="vertical">
                                <a-button
                                  type="primary"
                                  @click="addArgumentsItem(routeItemIndex, conditionItemIndex)"
                                >
                                  添加argument
                                </a-button>
                                <a-table
                                  :pagination="false"
                                  :columns="argumentsColumns"
                                  :data-source="routeItem.requestMatch[conditionItemIndex].list"
                                >
                                  <template
                                    #bodyCell="{ column, record, text, index: argumentIndex }"
                                  >
                                    <template v-if="column.key === 'index'">
                                      <a-input v-model:value="record.index" placeholder="index" />
                                    </template>
                                    <template v-else-if="column.key === 'condition'">
                                      <a-select
                                        v-model:value="record.condition"
                                        :options="conditionOptions"
                                      />
                                    </template>
                                    <template v-else-if="column.key === 'value'">
                                      <a-input v-model:value="record.value" placeholder="value" />
                                    </template>
                                    <template v-else-if="column.key === 'operation'">
                                      <a-space align="center">
                                        <Icon
                                          @click="
                                            deleteArgumentsItem(
                                              routeItemIndex,
                                              conditionItemIndex,
                                              argumentIndex
                                            )
                                          "
                                          icon="tdesign:remove"
                                          class="action-icon"
                                        />
                                      </a-space>
                                    </template>
                                  </template>
                                </a-table>
                              </a-space>
                            </a-space>
                            <!--                      attachments-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('attachments') &&
                                conditionItem.type === 'attachments'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-space direction="vertical">
                                <a-button
                                  type="primary"
                                  @click="addAttachmentsItem(routeItemIndex, conditionItemIndex)"
                                >
                                  添加attachment
                                </a-button>
                                <a-table
                                  :pagination="false"
                                  :columns="attachmentsColumns"
                                  :data-source="routeItem.requestMatch[conditionItemIndex].list"
                                >
                                  <template
                                    #bodyCell="{ column, record, text, index: attachmentsIndex }"
                                  >
                                    <template v-if="column.key === 'myKey'">
                                      <a-input v-model:value="record.myKey" placeholder="key" />
                                    </template>
                                    <template v-else-if="column.key === 'condition'">
                                      <a-select
                                        v-model:value="record.condition"
                                        :options="conditionOptions"
                                      />
                                    </template>
                                    <template v-else-if="column.key === 'value'">
                                      <a-input v-model:value="record.value" placeholder="value" />
                                    </template>
                                    <template v-else-if="column.key === 'operation'">
                                      <a-space align="center">
                                        <Icon
                                          @click="
                                            deleteAttachmentsItem(
                                              routeItemIndex,
                                              conditionItemIndex,
                                              attachmentsIndex
                                            )
                                          "
                                          icon="tdesign:remove"
                                          class="action-icon"
                                        />
                                      </a-space>
                                    </template>
                                  </template>
                                </a-table>
                              </a-space>
                            </a-space>
                            <!--                      other-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedMatchConditionTypes.includes('other') &&
                                conditionItem.type === 'other'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                              </a-tag>
                              <a-space direction="vertical">
                                <a-button
                                  type="primary"
                                  @click="addOtherItem(routeItemIndex, conditionItemIndex)"
                                >
                                  添加other
                                </a-button>
                                <a-table
                                  :pagination="false"
                                  :columns="otherColumns"
                                  :data-source="routeItem.requestMatch[conditionItemIndex].list"
                                >
                                  <template #bodyCell="{ column, record, text }">
                                    <template v-if="column.key === 'myKey'">
                                      <a-input v-model:value="record.myKey" placeholder="key" />
                                    </template>
                                    <template v-else-if="column.key === 'condition'">
                                      <a-select
                                        v-model:value="record.condition"
                                        :options="conditionOptions"
                                      />
                                    </template>
                                    <template v-else-if="column.key === 'value'">
                                      <a-input v-model:value="record.value" placeholder="value" />
                                    </template>
                                    <template v-else-if="column.key === 'operation'">
                                      <a-space align="center">
                                        <Icon
                                          @click="
                                            deleteOtherItem(
                                              routeItemIndex,
                                              conditionItemIndex,
                                              record.index
                                            )
                                          "
                                          icon="tdesign:remove"
                                          class="action-icon"
                                        />
                                        <!--                                      <Icon-->
                                        <!--                                        @click="addOtherItem(routeItemIndex, conditionItemIndex)"-->
                                        <!--                                        icon="tdesign:add"-->
                                        <!--                                        class="action-icon"-->
                                        <!--                                      />-->
                                      </a-space>
                                    </template>
                                  </template>
                                </a-table>
                              </a-space>
                            </a-space>
                          </template>
                        </a-space>
                      </a-card>
                      <a-button
                        @click="addRequestMatch(routeItemIndex)"
                        v-else
                        type="dashed"
                        size="large"
                      >
                        <template #icon>
                          <Icon icon="tdesign:add" />
                        </template>
                        增加匹配条件
                      </a-button>
                    </a-form-item>
                    <a-form-item label="路由分发" required>
                      <a-card>
                        <a-space style="width: 100%" direction="vertical" size="small">
                          <a-flex>
                            <a-form-item label="匹配条件类型">
                              <a-select
                                v-model:value="routeItem.selectedRouteDistributeMatchTypes"
                                :options="routeDistributionTypeOptions"
                                mode="multiple"
                                style="min-width: 200px"
                              />
                            </a-form-item>
                          </a-flex>
                          <template
                            v-for="(conditionItem, conditionItemIndex) in routeItem.routeDistribute"
                            :key="conditionItemIndex"
                          >
                            <!--                        host-->
                            <a-space
                              size="large"
                              align="center"
                              v-if="
                                routeItem.selectedRouteDistributeMatchTypes.includes('host') &&
                                conditionItem.type === 'host'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type }}
                              </a-tag>
                              <a-select
                                v-model:value="conditionItem.condition"
                                style="min-width: 120px"
                                :options="conditionOptions"
                              />
                              <a-input
                                v-model:value="conditionItem.value"
                                placeholder="请求来源ip"
                              />

                              <Icon
                                @click="
                                  deleteRouteDistributeMatchTypeItem(
                                    conditionItem?.type,
                                    routeItemIndex
                                  )
                                "
                                class="action-icon"
                                icon="tdesign:delete"
                              />
                            </a-space>

                            <!--                      other-->
                            <a-space
                              style="width: 100%"
                              size="large"
                              align="start"
                              v-if="
                                routeItem.selectedRouteDistributeMatchTypes.includes('other') &&
                                conditionItem.type === 'other'
                              "
                            >
                              <a-tag
                                class="match-condition-type-label"
                                :bordered="false"
                                color="processing"
                              >
                                {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                              </a-tag>
                              <a-space direction="vertical">
                                <a-button
                                  type="primary"
                                  @click="
                                    addRouteDistributeOtherItem(routeItemIndex, conditionItemIndex)
                                  "
                                >
                                  添加其他
                                </a-button>
                                <a-table
                                  :pagination="false"
                                  :columns="otherColumns"
                                  :data-source="routeItem.routeDistribute[conditionItemIndex].list"
                                >
                                  <template #bodyCell="{ column, record, text, index: otherIndex }">
                                    <template v-if="column.key === 'myKey'">
                                      <a-input v-model:value="record.myKey" placeholder="key" />
                                    </template>
                                    <template v-else-if="column.key === 'condition'">
                                      <a-select
                                        v-model:value="record.condition"
                                        :options="conditionOptions"
                                      />
                                    </template>
                                    <template v-else-if="column.key === 'value'">
                                      <a-input v-model:value="record.value" placeholder="value" />
                                    </template>
                                    <template v-else-if="column.key === 'operation'">
                                      <a-space align="center">
                                        <Icon
                                          @click="
                                            deleteRouteDistributeOtherItem(
                                              routeItemIndex,
                                              conditionItemIndex,
                                              otherIndex
                                            )
                                          "
                                          icon="tdesign:remove"
                                          class="action-icon"
                                        />
                                        <!--                                     <Icon-->
                                        <!--                                       @click="-->
                                        <!--                                          addRouteDistributeOtherItem(-->
                                        <!--                                            routeItemIndex,-->
                                        <!--                                            conditionItemIndex-->
                                        <!--                                          )-->
                                        <!--                                        "-->
                                        <!--                                       icon="tdesign:add"-->
                                        <!--                                       class="action-icon"-->
                                        <!--                                     />-->
                                      </a-space>
                                    </template>
                                  </template>
                                </a-table>
                              </a-space>
                            </a-space>
                          </template>
                        </a-space>
                      </a-card>
                    </a-form-item>
                  </a-space>
                </a-form>
              </a-card>
            </a-card>
            <a-button @click="addRoute" type="primary"> 增加路由</a-button>
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
import { Icon } from '@iconify/vue'
import { getConditionRuleDetailAPI, updateConditionRuleAPI } from '@/api/service/traffic'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { isNil } from 'lodash'
import { HTTP_STATUS } from '@/base/http/constants'
const TAB_STATE = inject(PROVIDE_INJECT_KEY.PROVIDE_INJECT_KEY)
const loading = ref(false)

onMounted(async () => {
  if (!isNil(TAB_STATE.conditionRule)) {
    const { enabled = true, key, scope, runtime = true, conditions } = TAB_STATE.conditionRule
    console.log('[ TAB_STATE.conditionRule ] >', TAB_STATE.conditionRule)
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

        const conditionArr = item.split('=>')
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

const matchConditionTypeOptions = ref([
  {
    label: 'host',
    value: 'host'
  },
  {
    label: 'application',
    value: 'application'
  },
  {
    label: 'method',
    value: 'method'
  },
  {
    label: 'arguments',
    value: 'arguments'
  },
  {
    label: 'attachments',
    value: 'attachments'
  },
  {
    label: '其他',
    value: 'other'
  }
])

const routeDistributionTypeOptions = ref([
  {
    label: 'host',
    value: 'host'
  },
  {
    label: '其他',
    value: 'other'
  }
])

const conditionOptions = ref([
  {
    label: '=',
    value: '='
  },
  {
    label: '!=',
    value: '!='
  }
])

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

// route list
const routeList: any = ref([
  {
    selectedMatchConditionTypes: [],
    requestMatch: [],
    selectedRouteDistributeMatchTypes: [],
    routeDistribute: []
  }
])

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

const addRoute = () => {
  routeList.value.push({
    selectedMatchConditionTypes: [],
    requestMatch: [],
    selectedRouteDistributeMatchTypes: [],
    routeDistribute: []
  })
}

const deleteRoute = (index: number) => {
  routeList.value.splice(index, 1)
}

const deleteRequestMatch = (index: number) => {
  routeList.value[index].requestMatch = []
  routeList.value[index].selectedMatchConditionTypes = []
}

const addRequestMatch = (index: number) => {
  routeList.value[index].requestMatch = [
    {
      type: 'host',
      condition: '',
      value: ''
    },
    {
      type: 'application',
      condition: '',
      value: ''
    },
    {
      type: 'method',
      condition: '',
      value: ''
    },
    {
      type: 'arguments',
      list: []
    },
    {
      type: 'attachments',
      list: []
    },
    {
      type: 'other',
      list: []
    }
  ]
}

const deleteMatchConditionTypeItem = (type: string, index: number) => {
  // console.log(type, index)
  routeList.value[index].selectedMatchConditionTypes = routeList.value[
    index
  ].selectedMatchConditionTypes.filter((item) => item !== type)
}

const deleteRouteDistributeMatchTypeItem = (type: string, index: number) => {
  routeList.value[index].selectedRouteDistributeMatchTypes = routeList.value[
    index
  ].selectedRouteDistributeMatchTypes.filter((item) => item !== type)
}

const argumentsColumns = [
  {
    dataIndex: 'index',
    key: 'index',
    title: '参数索引'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

// add argumentsItem
const addArgumentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    index: 0,
    condition: '=',
    value: ''
  })
}

// deleteArgumentsItem
const deleteArgumentsItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  argumentsIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'arguments')
  }

  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(argumentsIndex, 1)
}

// attachments
const attachmentsColumns = [
  {
    dataIndex: 'myKey',
    key: 'myKey',
    title: '键'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

const addAttachmentsItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    myKey: '',
    condition: '=',
    value: ''
  })
}

const deleteAttachmentsItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  attachmentsItemIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'attachments')
  }
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(
    attachmentsItemIndex,
    1
  )
}

// other
const otherColumns = [
  {
    dataIndex: 'myKey',
    key: 'myKey',
    title: '键'
  },
  {
    dataIndex: 'condition',
    key: 'condition',
    title: '关系'
  },
  {
    dataIndex: 'value',
    key: 'value',
    title: '值'
  },
  {
    dataIndex: 'operation',
    key: 'operation',
    title: '操作'
  }
]

const addOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.push({
    myKey: '',
    condition: '=',
    value: ''
  })
}

const deleteOtherItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  otherItemIndex: number
) => {
  if (routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedMatchConditionTypes = routeList.value[
      routeItemIndex
    ].selectedMatchConditionTypes.filter((item) => item !== 'other')
    return
  }
  routeList.value[routeItemIndex].requestMatch[conditionItemIndex].list.splice(otherItemIndex, 1)
}

const addRouteDistributeOtherItem = (routeItemIndex: number, conditionItemIndex: number) => {
  routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.push({
    myKey: '',
    condition: '=',
    value: ''
  })
}

const deleteRouteDistributeOtherItem = (
  routeItemIndex: number,
  conditionItemIndex: number,
  otherItemIndex: number
) => {
  if (routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.length === 1) {
    routeList.value[routeItemIndex].selectedRouteDistributeMatchTypes = routeList.value[
      routeItemIndex
    ].selectedRouteDistributeMatchTypes.filter((item) => item !== 'other')
    return
  }

  routeList.value[routeItemIndex].routeDistribute[conditionItemIndex].list.splice(otherItemIndex, 1)
}

const routeItemDes = (routeIndex: number) => {
  const routeItem = routeList.value[routeIndex]
  const { ruleGranularity, objectOfAction } = baseInfo

  const typeText = ruleGranularity === 'service' ? '服务' : '应用'
  let baseDescription = `对于${typeText}【${objectOfAction || '未指定'}】`

  // 构建匹配条件描述 (when)
  let whenConditions: string[] = []
  routeItem.selectedMatchConditionTypes?.forEach((type) => {
    const matchItem = routeItem.requestMatch?.find((item) => item.type === type)
    if (!matchItem) return

    let conditionStr = ''
    const conditionSymbol =
      matchItem.condition === '='
        ? '等于'
        : matchItem.condition === '!='
          ? '不等于'
          : matchItem.condition || ''
    const valueStr = matchItem.value || '未指定'

    switch (type) {
      case 'host':
        conditionStr = `请求来源主机 ${conditionSymbol} ${valueStr}`
        break
      case 'application':
        conditionStr = `请求来源应用 ${conditionSymbol} ${valueStr}`
        break
      case 'method':
        conditionStr = `请求方法 ${conditionSymbol} ${valueStr}`
        break
      case 'arguments':
        const argConditions = matchItem.list
          ?.map((arg) => {
            const argConditionSymbol =
              arg.condition === '='
                ? '等于'
                : arg.condition === '!='
                  ? '不等于'
                  : arg.condition || ''
            const argValueStr = arg.value !== undefined && arg.value !== '' ? arg.value : '未指定'
            return `参数[${arg.index}] ${argConditionSymbol} ${argValueStr}`
          })
          .filter(Boolean)
        if (argConditions?.length > 0) conditionStr = argConditions.join(' 且 ')
        break
      case 'attachments':
        const attachConditions = matchItem.list
          ?.map((attach) => {
            const attachConditionSymbol =
              attach.condition === '='
                ? '等于'
                : attach.condition === '!='
                  ? '不等于'
                  : attach.condition || ''
            const attachValueStr =
              attach.value !== undefined && attach.value !== '' ? attach.value : '未指定'
            return `附件[${attach.myKey || '未指定'}] ${attachConditionSymbol} ${attachValueStr}`
          })
          .filter(Boolean)
        if (attachConditions?.length > 0) conditionStr = attachConditions.join(' 且 ')
        break
      case 'other':
        const otherConditions = matchItem.list
          ?.map((other) => {
            const otherConditionSymbol =
              other.condition === '='
                ? '等于'
                : other.condition === '!='
                  ? '不等于'
                  : other.condition || ''
            const otherValueStr =
              other.value !== undefined && other.value !== '' ? other.value : '未指定'
            return `自定义匹配[${other.myKey || '未指定'}] ${otherConditionSymbol} ${otherValueStr}`
          })
          .filter(Boolean)
        if (otherConditions?.length > 0) conditionStr = otherConditions.join(' 且 ')
        break
    }
    if (conditionStr) {
      // Check for empty mandatory fields
      if ((type === 'host' || type === 'application' || type === 'method') && !matchItem.value) {
        whenConditions.push(
          `${type === 'host' ? '请求来源主机' : type === 'application' ? '请求来源应用' : '请求方法'} 未填写`
        )
      } else {
        whenConditions.push(conditionStr)
      }
    }
  })

  const whenConditionStr = whenConditions.length > 0 ? whenConditions.join(' 且 ') : '任意请求'

  // 构建转发条件描述 (then)
  let thenConditions: string[] = []
  routeItem.selectedRouteDistributeMatchTypes?.forEach((type) => {
    const distributeItem = routeItem.routeDistribute?.find((item) => item.type === type)
    if (!distributeItem) return

    let conditionStr = ''
    const conditionSymbol =
      distributeItem.condition === '='
        ? '等于'
        : distributeItem.condition === '!='
          ? '不等于'
          : distributeItem.condition || ''
    const valueStr = distributeItem.value || '未指定'

    switch (type) {
      case 'host':
        conditionStr = `目标主机 ${conditionSymbol} ${valueStr}`
        break
      case 'other':
        const otherConditions = distributeItem.list
          ?.map((other) => {
            const otherConditionSymbol =
              other.condition === '='
                ? '等于'
                : other.condition === '!='
                  ? '不等于'
                  : other.condition || ''
            const otherValueStr =
              other.value !== undefined && other.value !== '' ? other.value : '未指定'
            return `目标标签[${other.myKey || '未指定'}] ${otherConditionSymbol} ${otherValueStr}`
          })
          .filter(Boolean)
        if (otherConditions?.length > 0) conditionStr = otherConditions.join(' 且 ')
        break
    }
    if (conditionStr) {
      if (type === 'host' && !distributeItem.value) {
        thenConditions.push(`目标主机 未填写`)
      } else {
        thenConditions.push(conditionStr)
      }
    }
  })

  const thenConditionStr =
    thenConditions.length > 0 ? `满足 【${thenConditions.join(' 且 ')}】` : '默认路由规则'

  return `${baseDescription}，将满足 【${whenConditionStr}】 条件的请求，转发到 ${thenConditionStr} 的实例。`
}

// Condition type configuration
const CONDITION_TYPE_CONFIG = {
  // Single value types: type=value or type!=value
  single: ['host', 'application', 'method'],
  // Array types: type[key]=value or type[key]!=value
  array: ['arguments', 'attachments'],
  // Custom types: key=value (without type prefix)
  custom: ['other']
} as const

// Type definitions
interface ConditionItem {
  type: string
  condition?: string
  value?: string
  list?: Array<Record<string, any>>
}

interface ParseOptions {
  availableTypes: readonly string[]
  isMatchCondition: boolean
}

/**
 * Parse a condition part and return parsed data
 */
function parseConditionPart(part: string, resultArray: ConditionItem[], type: string): boolean {
  part = part.trim()

  // Handle single value types (host, application, method)
  if (CONDITION_TYPE_CONFIG.single.includes(type)) {
    const match = part.match(new RegExp(`^${type}(!=|=)(.+)`))
    if (match) {
      resultArray.push({ type, condition: match[1], value: match[2].trim() })
      return true
    }
  }

  // Handle arguments: arguments[index]=value
  if (type === 'arguments') {
    const match = part.match(/^arguments\[(\d+)\](!=|=)(.+)/)
    if (match) {
      let argObj = resultArray.find((item) => item.type === 'arguments')
      if (!argObj) {
        argObj = { type: 'arguments', list: [] }
        resultArray.push(argObj)
      }
      argObj.list!.push({
        index: parseInt(match[1], 10),
        condition: match[2],
        value: match[3].trim()
      })
      return true
    }
  }

  // Handle attachments: attachments[key]=value
  if (type === 'attachments') {
    const match = part.match(/^attachments\[(.+)\](!=|=)(.+)/)
    if (match) {
      let attachObj = resultArray.find((item) => item.type === 'attachments')
      if (!attachObj) {
        attachObj = { type: 'attachments', list: [] }
        resultArray.push(attachObj)
      }
      attachObj.list!.push({ myKey: match[1].trim(), condition: match[2], value: match[3].trim() })
      return true
    }
  }

  return false
}

/**
 * Generic condition parser
 */
function parseConditionString(
  conditionStr: string,
  routeItemIndex: number,
  options: ParseOptions
): ConditionItem[] {
  const { availableTypes, isMatchCondition } = options
  const resultArray: ConditionItem[] = []
  const selectedTypesKey = isMatchCondition
    ? 'selectedMatchConditionTypes'
    : 'selectedRouteDistributeMatchTypes'

  // Clear selected types for this route item
  routeList.value[routeItemIndex][selectedTypesKey] = []

  if (!conditionStr) {
    // Return default empty structure
    return availableTypes.map((type) => {
      if (
        CONDITION_TYPE_CONFIG.array.includes(type) ||
        CONDITION_TYPE_CONFIG.custom.includes(type)
      ) {
        return { type, list: [] }
      }
      return { type, condition: '', value: '' }
    })
  }

  const parts = conditionStr.split(' & ')

  parts.forEach((part) => {
    if (!part.trim()) return

    let parsed = false

    // Try to parse with known types
    for (const type of availableTypes) {
      if (part.startsWith(type)) {
        if (parseConditionPart(part, resultArray, type)) {
          // Add to selected types if not already present
          const selectedTypes = routeList.value[routeItemIndex][selectedTypesKey]
          if (!selectedTypes.includes(type)) {
            selectedTypes.push(type)
          }
          parsed = true
          break
        }
      }
    }

    // Handle custom key=value pairs (other type)
    // This applies to both match conditions and route distribute conditions
    if (!parsed) {
      const match = part.match(/^([^!=]+)(!?=)(.+)$/)
      if (match) {
        const type = 'other'
        let otherObj = resultArray.find((item) => item.type === type)
        if (!otherObj) {
          otherObj = { type, list: [] }
          resultArray.push(otherObj)
        }
        otherObj.list!.push({
          myKey: match[1].trim(),
          condition: match[2],
          value: match[3].trim()
        })

        // Add to selected types
        const selectedTypes = routeList.value[routeItemIndex][selectedTypesKey]
        if (!selectedTypes.includes(type)) {
          selectedTypes.push(type)
        }
      }
    }
  })

  // Add default empty structures for types that weren't found
  availableTypes.forEach((type) => {
    if (!resultArray.find((item) => item.type === type)) {
      if (
        CONDITION_TYPE_CONFIG.array.includes(type) ||
        CONDITION_TYPE_CONFIG.custom.includes(type)
      ) {
        resultArray.push({ type, list: [] })
      } else {
        resultArray.push({ type, condition: '', value: '' })
      }
    }
  })

  return resultArray
}

/**
 * Parse match condition string (when part)
 */
function parseConditionMatchStringToArray(matchStr: string, routeItemIndex: number) {
  return parseConditionString(matchStr, routeItemIndex, {
    availableTypes: [
      ...CONDITION_TYPE_CONFIG.single,
      ...CONDITION_TYPE_CONFIG.array,
      ...CONDITION_TYPE_CONFIG.custom
    ],
    isMatchCondition: true
  })
}

/**
 * Parse route distribute condition string (then part)
 */
function parseConditionToStringToArray(toStr: string, routeItemIndex: number) {
  return parseConditionString(toStr || '', routeItemIndex, {
    availableTypes: ['host', 'other'],
    isMatchCondition: false
  })
}

// Test case
// const str = 'host=example.com & application=myApp & method=getItem & arguments[1]!=dubbo & arguments[2]=dubbo2 & attachments[myKey]=myValue & other[myKey2]=myValue2';
// const test = parseConditionsStringToArray(str, 0);
// routeList.value[0].requestMatch = test
// console.log('test', test)

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

        const conditionArr = item.split('=>')
        const match = conditionArr[0]?.trim()
        const to = conditionArr[1]?.trim()
        // console.log('to', to)
        routeList.value[index].requestMatch = parseConditionMatchStringToArray(match, index)
        routeList.value[index].routeDistribute = parseConditionToStringToArray(to, index)
      })
    }
  }
}

/**
 * Merge condition items into a string
 */
function mergeConditionItems(
  selectedTypes: string[],
  conditionItems: any[],
  separator: string = ' & '
): string {
  const result: string[] = []

  selectedTypes.forEach((type) => {
    const item = conditionItems.find((i) => i.type === type)
    if (!item) return

    // Handle list-based types (arguments, attachments, other)
    if (item.list && Array.isArray(item.list)) {
      item.list.forEach((listItem: any) => {
        if (listItem.value !== undefined && listItem.value !== '') {
          if (type === 'arguments') {
            result.push(`${type}[${listItem.index}]${listItem.condition}${listItem.value}`)
          } else if (type === 'attachments') {
            result.push(`${type}[${listItem.myKey}]${listItem.condition}${listItem.value}`)
          } else if (type === 'other') {
            result.push(`${listItem.myKey}${listItem.condition}${listItem.value}`)
          }
        }
      })
    }
    // Handle single value types (host, application, method)
    else if (item.value !== undefined && item.value !== '') {
      result.push(`${item.type}${item.condition}${item.value}`)
    }
  })

  return result.join(separator)
}

/**
 * Merge all route conditions into condition strings
 */
function mergeConditions() {
  const conditions: string[] = []

  routeList.value.forEach((routeItem) => {
    // Merge match conditions (when)
    const matchStr = mergeConditionItems(
      routeItem.selectedMatchConditionTypes,
      routeItem.requestMatch
    )

    // Merge distribute conditions (then)
    const toStr = mergeConditionItems(
      routeItem.selectedRouteDistributeMatchTypes,
      routeItem.routeDistribute
    )

    // Only add condition if there's actual content in match part
    if (matchStr.length > 0) {
      if (toStr.length > 0) {
        conditions.push(`${matchStr} => ${toStr}`)
      } else {
        conditions.push(matchStr)
      }
    }
  })

  return conditions
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
