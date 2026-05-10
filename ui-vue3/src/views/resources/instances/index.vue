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
  <div class="instances-container">
    <search-table :search-domain="searchDomain">
      <template #customOperation>
        <a-button
          class="refresh-button"
          :loading="searchDomain.table.loading"
          @click="searchDomain.onSearch()"
        >
          <template #icon>
            <Icon icon="material-symbols:refresh-rounded"></Icon>
          </template>
          {{ $t('refresh') }}
        </a-button>
      </template>
      <template #bodyCell="{ text, record, index, column }">
        <template v-if="column.dataIndex === 'name'">
          <a-tooltip :title="text">
            <span
              class="app-link"
              @click="
                router.push(
                  `/resources/instances/detail/${record.name}/${record.ip}/${record.appName}`
                )
              "
            >
              <b>
                <Icon
                  style="margin-bottom: -2px"
                  icon="material-symbols:attach-file-rounded"
                ></Icon>
                {{ text }}
              </b>
            </span>
          </a-tooltip>
        </template>

        <template v-if="column.dataIndex === 'ip'">
          <span>{{ text }}</span>
        </template>

        <template v-if="column.dataIndex === 'lifecycleState'">
          <a-tag :color="INSTANCE_LIFECYCLE_COLOR[(text || 'UNKNOWN').toUpperCase()] || 'default'">
            {{ text }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'deployCluster'">
          <a-tag color="grey">
            {{ text }}
          </a-tag>
        </template>

        <template v-if="column.dataIndex === 'registerClusters'">
          <a-tag v-for="t in text" color="grey">
            {{ t }}
          </a-tag>
        </template>

        <!-- <template v-if="column.dataIndex === 'registerTime'">
          {{ formattedDate(text) }}
        </template>

        <template v-if="column.dataIndex === 'labels'">
          <a-tag v-for="(value, key) in text" :key="key"> {{ key }}:{{ value }}</a-tag>
        </template> -->
      </template>
    </search-table>
  </div>
</template>

<script setup lang="ts">
import { onMounted, provide, reactive, watch } from 'vue'
import { searchInstances } from '@/api/service/instance'
import SearchTable from '@/components/SearchTable.vue'
import { SearchDomain } from '@/utils/SearchUtil'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import { INSTANCE_LIFECYCLE_COLOR, PRIMARY_COLOR } from '@/base/constants'
import router from '@/router'
import { Icon } from '@iconify/vue'
import { queryMetrics } from '@/base/http/promQuery'
import { isNumber } from 'lodash'
import { bytesToHuman } from '@/utils/ByteUtil'
import { promQueryList } from '@/utils/PromQueryUtil'
import { useRoute } from 'vue-router'
let route = useRoute()
let query = route.query['query']
let __null = PRIMARY_COLOR
let columns = [
  {
    title: 'instanceDomain.instanceName',
    key: 'name',
    dataIndex: 'name',
    // sorter: (a: any, b: any) => sortString(a.name, b.name),
    width: 140
  },
  {
    title: 'instanceDomain.instanceIP',
    key: 'ip',
    dataIndex: 'ip',
    // sorter: (a: any, b: any) => sortString(a.ip, b.ip),
    width: 200
  },
  {
    title: 'instanceDomain.lifecycleState',
    key: 'lifecycleState',
    dataIndex: 'lifecycleState',
    width: 130
  },
  {
    title: 'instanceDomain.deployCluster',
    key: 'deployCluster',
    dataIndex: 'deployCluster',
    // sorter: (a: any, b: any) => sortString(a.deployCluster, b.deployCluster),
    width: 120
  },
  {
    title: 'instanceDomain.registerCluster',
    key: 'registerClusters',
    dataIndex: 'registerClusters',
    // sorter: (a: any, b: any) => sortString(a.registerClusters, b.registerClusters),
    width: 140
  },
  {
    title: 'instanceDomain.CPU',
    key: 'cpu',
    dataIndex: 'cpu',
    // sorter: (a: any, b: any) => sortString(a.cpu, b.cpu),
    width: 140
  },
  {
    title: 'instanceDomain.memory',
    key: 'memory',
    dataIndex: 'memory',
    // sorter: (a: any, b: any) => sortString(a.memory, b.memory),
    width: 100
  },
  {
    title: 'instanceDomain.startTime_k8s',
    key: 'startTime_k8s',
    dataIndex: 'startTime',
    // sorter: (a: any, b: any) => sortString(a.startTime, b.startTime),
    width: 200
  }
  // {
  //   title: 'instanceDomain.registerTime',
  //   key: 'registerTime',
  //   dataIndex: 'registerTime',
  //   sorter: (a: any, b: any) => sortString(a.registerTime, b.registerTime),
  //   width: 200
  // },
  // {
  //   title: 'instanceDomain.labels',
  //   key: 'labels',
  //   dataIndex: 'labels',
  //   width: 800
  // }
]

function instanceInfo(params: any) {
  return searchInstances(params).then(async (res) => {
    return promQueryList(res, ['cpu', 'memory'], async (instance: any) => {
      let ip = instance.ip.split(':')[0]
      let cpu =
        await queryMetrics(`sum(node_namespace_pod_container:container_cpu_usage_seconds_total:sum_irate{container!=""}) by (pod) * on (pod) group_left(pod_ip)
        kube_pod_info{pod_ip="${ip}"}`)
      let mem = await queryMetrics(`sum(container_memory_working_set_bytes{container!=""}) by (pod)
* on (pod) group_left(pod_ip)
kube_pod_info{pod_ip="${ip}"}`)
      instance.cpu = isNumber(cpu) ? cpu.toFixed(3) + 'u' : cpu
      instance.memory = bytesToHuman(mem)
    })
  })
}

// search
const searchDomain = reactive(
  new SearchDomain(
    [
      {
        label: 'instanceDomain.instanceIP',
        param: 'keywords',
        defaultValue: query,
        placeholder: 'typeInstanceIP',
        style: {
          width: '200px'
        }
      }
    ],
    instanceInfo,
    columns
  )
)

onMounted(() => {
  searchDomain.tableStyle = {
    scrollX: '100',
    scrollY: '367px'
  }
  searchDomain.onSearch()
})

provide(PROVIDE_INJECT_KEY.SEARCH_DOMAIN, searchDomain)
watch(route, (a, b) => {
  searchDomain.queryForm['keywords'] = a.query['query']
  searchDomain.onSearch()
  console.log(a)
})
</script>
<style lang="less" scoped>
.instances-container {
  width: 100%;
  height: 100%;
  // background-color:rgb(145, 41, 41);

  .search-table-container {
    min-height: 60vh;
  }
}
</style>
