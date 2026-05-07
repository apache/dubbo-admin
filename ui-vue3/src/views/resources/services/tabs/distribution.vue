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
  <div class="__container_services_tabs_distribution">
    <search-table :search-domain="searchDomain">
      <template #bodyCell="{ column, text }">
        <template v-if="column.dataIndex === 'appName'">
          <span class="link" @click="router.push('/resources/applications/detail/' + text)">
            <b>
              <Icon style="margin-bottom: -2px" icon="material-symbols:attach-file-rounded"></Icon>
              {{ text }}
            </b>
          </span>
        </template>

        <template v-if="column.dataIndex === 'instanceName'">
          <span class="link" @click="router.push('/resources/instances/detail/' + text)">
            <b>
              <Icon style="margin-bottom: -2px" icon="material-symbols:attach-file-rounded"></Icon>
              {{ text }}
            </b>
          </span>
        </template>

        <template v-if="column.dataIndex === 'deployClusters'">
          <a-tag v-for="t in text" :key="t" :color="PRIMARY_COLOR">
            {{ t }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'registryClusters'">
          <a-tag v-for="t in text" :key="t" :color="PRIMARY_COLOR">
            {{ t }}
          </a-tag>
        </template>
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { reactive, provide } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getServiceDistribution } from '@/api/service/service'
import { PRIMARY_COLOR } from '@/base/constants'
import { Icon } from '@iconify/vue'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'

let __null = PRIMARY_COLOR
const router = useRouter()
const route = useRoute()

const tableColumns = [
  {
    title: 'servicesDomain.appName',
    key: 'appName',
    dataIndex: 'appName',
    width: 140,
    ellipsis: true
  },
  {
    title: 'servicesDomain.instanceCount',
    key: 'instanceCount',
    dataIndex: 'instanceCount',
    width: 100
  },
  {
    title: 'servicesDomain.deployClusters',
    key: 'deployClusters',
    dataIndex: 'deployClusters',
    width: 120
  },
  {
    title: 'servicesDomain.registryClusters',
    key: 'registryClusters',
    dataIndex: 'registryClusters',
    width: 200
  }
]

function getDistribution(params: any) {
  return getServiceDistribution({
    serviceName: route.params?.pathId,
    side: 'consumer',
    version: route.params?.version || '',
    group: route.params?.group || '',
    providerAppName: route.query?.providerAppName || '',
    ...params
  })
}

const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'servicesDomain.consumerAppName',
        param: 'keywords',
        placeholder: 'consumerAppName',
        style: {
          width: '300px'
        }
      }
    ],
    getDistribution,
    tableColumns
  )
)

searchDomain.onSearch()
searchDomain.tableStyle = {
  scrollX: '100',
  scrollY: '367px'
}

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
</script>

<style lang="less" scoped>
.__container_services_tabs_distribution {
  .service-filter {
    margin-bottom: 20px;

    .service-filter-select {
      margin-left: 10px;
      width: 250px;
    }

    .service-filter-input {
      margin-left: 30px;
      width: 300px;
    }
  }

  .link {
    padding: 4px 10px 4px 4px;
    border-radius: 4px;
    color: v-bind('PRIMARY_COLOR');

    &:hover {
      cursor: pointer;
      background: rgba(133, 131, 131, 0.13);
    }
  }
}
</style>
