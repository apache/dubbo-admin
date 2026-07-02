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
  <div class="routing-rule-container">
    <search-table :search-domain="searchDomain">
      <template #customOperation>
        <a-button type="primary" @click="router.push(`/traffic/addRoutingRule/addByFormView`)"
          >{{ t('routingRuleDomain.createNewRoutingRule') }}
        </a-button>
      </template>
      <template #bodyCell="{ text, column, record }">
        <template v-if="column.dataIndex === 'ruleName'">
          <span class="rule-link" @click="router.push(`formview/${record[column.key]}`)">
            <b>
              <Icon style="margin-bottom: -2px" icon="material-symbols:attach-file-rounded"></Icon>
              {{ text }}
            </b>
          </span>
        </template>
        <template v-if="column.dataIndex === 'ruleGranularity'">
          {{
            record.scope === 'service'
              ? t('routingRuleDomain.service')
              : t('routingRuleDomain.application')
          }}
        </template>
        <template v-if="column.dataIndex === 'enabled'">
          {{ text ? t('flowControlDomain.enabled') : t('flowControlDomain.disabled') }}
        </template>
        <!-- 时间 -->
        <template v-if="column.dataIndex === 'createTime'">
          {{ formattedDate(record.createTime) }}
        </template>
        <template v-if="column.dataIndex === 'operation'">
          <a-button type="link" @click="router.push(`formview/${record.ruleName}`)">
            {{ t('view') }}
          </a-button>
          <a-button
            type="link"
            @click="router.push(`/traffic/updateRoutingRule/updateByFormView/${record.ruleName}`)"
          >
            {{ t('edit') }}
          </a-button>
          <a-popconfirm
            :title="t('routingRuleDomain.warnDeleteRouteRule')"
            ok-text="Yes"
            cancel-text="No"
            @confirm="confirm(record.ruleName)"
          >
            <a-button type="link"> {{ t('delete') }}</a-button>
          </a-popconfirm>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, provide, reactive, inject } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
import { deleteConditionRuleAPI, searchRoutingRule } from '@/api/service/traffic'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain, sortString } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import router from '@/router'
import { Icon } from '@iconify/vue'
import { PRIMARY_COLOR } from '@/base/constants'
import { formattedDate } from '@/utils/DateUtil'
import { HTTP_STATUS } from '@/base/http/constants'
import { message } from 'ant-design-vue'
const TAB_STATE = inject(PROVIDE_INJECT_KEY.TAB_LAYOUT_STATE) as any
let columns = [
  {
    title: 'ruleName',
    key: 'ruleName',
    dataIndex: 'ruleName',
    // sorter: (a: any, b: any) => sortString(a.appName, b.appName),
    width: 140
  },
  {
    title: 'ruleGranularity',
    key: 'ruleGranularity',
    dataIndex: 'ruleGranularity',
    render: (_text: any, record: any) =>
      record.isService ? t('routingRuleDomain.service') : t('routingRuleDomain.application'),
    width: 100
    // sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },
  {
    title: 'createTime',
    key: 'createTime',
    dataIndex: 'createTime',
    width: 120
    // sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },
  {
    title: 'enabled',
    key: 'enabled',
    dataIndex: 'enabled',
    width: 120
    // sorter: (a: any, b: any) => sortString(a.instanceNum, b.instanceNum)
  },
  {
    title: 'operation',
    key: 'operation',
    dataIndex: 'operation',
    width: 200
  }
]
const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'serviceGovernance',
        param: 'keywords',
        placeholder: 'typeRoutingRules',
        style: {
          width: '200px'
        }
      }
    ],
    searchRoutingRule,
    columns
  )
)

//Delete conditional routing
const deleteRule = async (ruleName: string) => {
  try {
    const res = await deleteConditionRuleAPI(ruleName)
    if (res.code === HTTP_STATUS.SUCCESS) {
      await searchDomain.onSearch()
    }
  } catch (e: any) {
    message.error(e?.message || String(e))
  }
}

onMounted(() => {
  TAB_STATE.conditionRule = null
  TAB_STATE.addConditionRuleSate = null
  searchDomain.onSearch()
  searchDomain.tableStyle = {
    scrollX: '100',
    scrollY: '367px'
  }
})

const confirm = (ruleName: string) => {
  deleteRule(ruleName)
}

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>
<style lang="less" scoped>
.routing-rule-container {
  height: 100%;

  .search-table-container {
    height: 100%;
    //min-height: 60vh;
    //max-height: 70vh; //overflow: auto;

    .rule-link {
      padding: 4px 10px 4px 4px;
      border-radius: 4px;
      color: v-bind('PRIMARY_COLOR');

      &:hover {
        cursor: pointer;
        background: rgba(133, 131, 131, 0.13);
      }
    }
  }
}
</style>
