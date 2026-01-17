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
  <a-card v-for="(routeItem, routeItemIndex) in routeList" :key="routeItemIndex">
    <template #title>
      <a-flex justify="space-between">
        <a-space align="center">
          <div>路由【{{ routeItemIndex + 1 }}】</div>
          <a-tooltip>
            <template #title>{{
              routingRuleLogic.routeItemDes(routeItemIndex, baseInfo)
            }}</template>
            <div
              style="
                max-width: 400px;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              "
            >
              {{ routingRuleLogic.routeItemDes(routeItemIndex, baseInfo) }}
            </div>
          </a-tooltip>
        </a-space>
        <Icon
          @click="routingRuleLogic.deleteRoute(routeItemIndex)"
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
                  @click="routingRuleLogic.deleteRequestMatch(routeItemIndex)"
                  class="action-icon"
                  icon="tdesign:delete"
                />
              </a-flex>
              <template v-for="(conditionItem, conditionItemIndex) in routeItem.requestMatch">
                <!-- host -->
                <a-space
                  size="large"
                  align="center"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('host') &&
                    conditionItem.type === 'host'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type }}
                  </a-tag>
                  <a-select
                    v-model:value="conditionItem.condition"
                    style="min-width: 120px"
                    :options="conditionOptions"
                  />
                  <a-input v-model:value="conditionItem.value" placeholder="请求来源ip" />

                  <Icon
                    @click="
                      routingRuleLogic.deleteMatchConditionTypeItem(
                        conditionItem?.type,
                        routeItemIndex
                      )
                    "
                    class="action-icon"
                    icon="tdesign:delete"
                  />
                </a-space>
                <!-- application -->
                <a-space
                  size="large"
                  align="center"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('application') &&
                    conditionItem.type === 'application'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type }}
                  </a-tag>
                  <a-select
                    v-model:value="conditionItem.condition"
                    style="min-width: 120px"
                    :options="conditionOptions"
                  />
                  <a-input v-model:value="conditionItem.value" placeholder="请求来源应用名" />

                  <Icon
                    @click="
                      routingRuleLogic.deleteMatchConditionTypeItem(
                        conditionItem?.type,
                        routeItemIndex
                      )
                    "
                    class="action-icon"
                    icon="tdesign:delete"
                  />
                </a-space>
                <!-- method -->
                <a-space
                  size="large"
                  align="center"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('method') &&
                    conditionItem.type === 'method'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
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
                      routingRuleLogic.deleteMatchConditionTypeItem(
                        conditionItem?.type,
                        routeItemIndex
                      )
                    "
                    class="action-icon"
                    icon="tdesign:delete"
                  />
                </a-space>
                <!-- arguments -->
                <a-space
                  style="width: 100%"
                  size="large"
                  align="start"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('arguments') &&
                    conditionItem.type === 'arguments'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type }}
                  </a-tag>
                  <a-space direction="vertical">
                    <a-button
                      type="primary"
                      @click="routingRuleLogic.addArgumentsItem(routeItemIndex, conditionItemIndex)"
                    >
                      添加argument
                    </a-button>
                    <a-table
                      :pagination="false"
                      :columns="argumentsColumns"
                      :data-source="routeItem.requestMatch[conditionItemIndex].list"
                    >
                      <template #bodyCell="{ column, record, text, index: argumentIndex }">
                        <template v-if="column.key === 'index'">
                          <a-input v-model:value="record.index" placeholder="index" />
                        </template>
                        <template v-else-if="column.key === 'condition'">
                          <a-select v-model:value="record.condition" :options="conditionOptions" />
                        </template>
                        <template v-else-if="column.key === 'value'">
                          <a-input v-model:value="record.value" placeholder="value" />
                        </template>
                        <template v-else-if="column.key === 'operation'">
                          <a-space align="center">
                            <Icon
                              @click="
                                routingRuleLogic.deleteArgumentsItem(
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
                <!-- attachments -->
                <a-space
                  style="width: 100%"
                  size="large"
                  align="start"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('attachments') &&
                    conditionItem.type === 'attachments'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type }}
                  </a-tag>
                  <a-space direction="vertical">
                    <a-button
                      type="primary"
                      @click="
                        routingRuleLogic.addAttachmentsItem(routeItemIndex, conditionItemIndex)
                      "
                    >
                      添加attachment
                    </a-button>
                    <a-table
                      :pagination="false"
                      :columns="attachmentsColumns"
                      :data-source="routeItem.requestMatch[conditionItemIndex].list"
                    >
                      <template #bodyCell="{ column, record, text, index: attachmentsIndex }">
                        <template v-if="column.key === 'myKey'">
                          <a-input v-model:value="record.myKey" placeholder="key" />
                        </template>
                        <template v-else-if="column.key === 'condition'">
                          <a-select v-model:value="record.condition" :options="conditionOptions" />
                        </template>
                        <template v-else-if="column.key === 'value'">
                          <a-input v-model:value="record.value" placeholder="value" />
                        </template>
                        <template v-else-if="column.key === 'operation'">
                          <a-space align="center">
                            <Icon
                              @click="
                                routingRuleLogic.deleteAttachmentsItem(
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
                <!-- other -->
                <a-space
                  style="width: 100%"
                  size="large"
                  align="start"
                  v-if="
                    routeItem.selectedMatchConditionTypes.includes('other') &&
                    conditionItem.type === 'other'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                  </a-tag>
                  <a-space direction="vertical">
                    <a-button
                      type="primary"
                      @click="routingRuleLogic.addOtherItem(routeItemIndex, conditionItemIndex)"
                    >
                      添加other
                    </a-button>
                    <a-table
                      :pagination="false"
                      :columns="otherColumns"
                      :data-source="routeItem.requestMatch[conditionItemIndex].list"
                    >
                      <template #bodyCell="{ column, record, text, index: otherIndex }">
                        <template v-if="column.key === 'myKey'">
                          <a-input v-model:value="record.myKey" placeholder="key" />
                        </template>
                        <template v-else-if="column.key === 'condition'">
                          <a-select v-model:value="record.condition" :options="conditionOptions" />
                        </template>
                        <template v-else-if="column.key === 'value'">
                          <a-input v-model:value="record.value" placeholder="value" />
                        </template>
                        <template v-else-if="column.key === 'operation'">
                          <a-space align="center">
                            <Icon
                              @click="
                                routingRuleLogic.deleteOtherItem(
                                  routeItemIndex,
                                  conditionItemIndex,
                                  otherIndex
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
              </template>
            </a-space>
          </a-card>
          <a-button
            @click="routingRuleLogic.addRequestMatch(routeItemIndex)"
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
                <!-- host -->
                <a-space
                  size="large"
                  align="center"
                  v-if="
                    routeItem.selectedRouteDistributeMatchTypes.includes('host') &&
                    conditionItem.type === 'host'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type }}
                  </a-tag>
                  <a-select
                    v-model:value="conditionItem.condition"
                    style="min-width: 120px"
                    :options="conditionOptions"
                  />
                  <a-input v-model:value="conditionItem.value" placeholder="请求来源ip" />

                  <Icon
                    @click="
                      routingRuleLogic.deleteRouteDistributeMatchTypeItem(
                        conditionItem?.type,
                        routeItemIndex
                      )
                    "
                    class="action-icon"
                    icon="tdesign:delete"
                  />
                </a-space>

                <!-- other -->
                <a-space
                  style="width: 100%"
                  size="large"
                  align="start"
                  v-if="
                    routeItem.selectedRouteDistributeMatchTypes.includes('other') &&
                    conditionItem.type === 'other'
                  "
                >
                  <a-tag class="match-condition-type-label" :bordered="false" color="processing">
                    {{ conditionItem?.type == 'other' ? '其他' : conditionItem?.type }}
                  </a-tag>
                  <a-space direction="vertical">
                    <a-button
                      type="primary"
                      @click="
                        routingRuleLogic.addRouteDistributeOtherItem(
                          routeItemIndex,
                          conditionItemIndex
                        )
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
                          <a-select v-model:value="record.condition" :options="conditionOptions" />
                        </template>
                        <template v-else-if="column.key === 'value'">
                          <a-input v-model:value="record.value" placeholder="value" />
                        </template>
                        <template v-else-if="column.key === 'operation'">
                          <a-space align="center">
                            <Icon
                              @click="
                                routingRuleLogic.deleteRouteDistributeOtherItem(
                                  routeItemIndex,
                                  conditionItemIndex,
                                  otherIndex
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
              </template>
            </a-space>
          </a-card>
        </a-form-item>
      </a-space>
    </a-form>
  </a-card>
  <a-button @click="routingRuleLogic.addRoute" type="primary"> 增加路由</a-button>
</template>

<script lang="ts" setup>
import { Icon } from '@iconify/vue'
import {
  matchConditionTypeOptions,
  conditionOptions,
  routeDistributionTypeOptions
} from '../composables/useRoutingRule'

const props = defineProps<{
  routeList: any[]
  baseInfo: any
  routingRuleLogic: any
}>()

const argumentsColumns = [
  { dataIndex: 'index', key: 'index', title: '参数索引' },
  { dataIndex: 'condition', key: 'condition', title: '关系' },
  { dataIndex: 'value', key: 'value', title: '值' },
  { dataIndex: 'operation', key: 'operation', title: '操作' }
]

const attachmentsColumns = [
  { dataIndex: 'myKey', key: 'myKey', title: '键' },
  { dataIndex: 'condition', key: 'condition', title: '关系' },
  { dataIndex: 'value', key: 'value', title: '值' },
  { dataIndex: 'operation', key: 'operation', title: '操作' }
]

const otherColumns = [
  { dataIndex: 'myKey', key: 'myKey', title: '键' },
  { dataIndex: 'condition', key: 'condition', title: '关系' },
  { dataIndex: 'value', key: 'value', title: '值' },
  { dataIndex: 'operation', key: 'operation', title: '操作' }
]
</script>

<style lang="less" scoped>
.action-icon {
  font-size: 17px;
  margin-left: 10px;
  cursor: pointer;
}

.match-condition-type-label {
  min-width: 100px;
  text-align: center;
}
</style>
