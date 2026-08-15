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
  <div class="router-rule-list">
    <search-table :search-domain="searchDomain">
      <template #customOperation>
        <a-button type="primary" @click="router.push(`${basePath}/edit`)">{{
          t('create')
        }}</a-button>
      </template>
      <template #bodyCell="{ text, column, record }">
        <template v-if="column.dataIndex === 'ruleName'">
          <a @click="router.push(`${basePath}/edit/${record.ruleName}`)">{{ text }}</a>
        </template>
        <template v-else-if="column.dataIndex === 'enabled'">
          {{ text ? t('enabled') : t('disabled') }}
        </template>
        <template v-else-if="column.dataIndex === 'operation'">
          <a-button type="link" @click="router.push(`${basePath}/edit/${record.ruleName}`)">{{
            t('edit')
          }}</a-button>
          <a-popconfirm :title="t('warnDeleteRoutingRule')" @confirm="remove(record.ruleName)">
            <a-button type="link">{{ t('delete') }}</a-button>
          </a-popconfirm>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, provide, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import router from '@/router'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import {
  deleteRouterRuleAPI,
  searchRouterRuleAPI,
  type RouterRuleKind
} from '@/api/service/traffic'
import { HTTP_STATUS } from '@/base/http/constants'
import { message } from 'ant-design-vue'

const props = defineProps<{ kind: RouterRuleKind }>()
const { t } = useI18n()
const basePath = computed(
  () => `/traffic/${props.kind === 'affinity-rule' ? 'affinityRule' : 'scriptRule'}`
)
const columns = [
  { title: 'ruleName', key: 'ruleName', dataIndex: 'ruleName' },
  { title: 'scope', key: 'scope', dataIndex: 'scope', width: 140 },
  { title: 'enabled', key: 'enabled', dataIndex: 'enabled', width: 120 },
  { title: 'operation', key: 'operation', dataIndex: 'operation', width: 180 }
]
const searchDomain = reactive(
  new SearchDomain(
    [{ label: 'serviceGovernance', param: 'keywords', placeholder: 'typeRoutingRules' }],
    (params: any) => searchRouterRuleAPI(props.kind, params),
    columns
  )
)

async function remove(ruleName: string) {
  try {
    const res = await deleteRouterRuleAPI(props.kind, ruleName)
    if (res.code === HTTP_STATUS.SUCCESS) await searchDomain.onSearch()
  } catch (error: any) {
    message.error(error?.message || String(error))
  }
}

onMounted(() => searchDomain.onSearch())
provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>

<style scoped>
.router-rule-list {
  height: 100%;
}
</style>
