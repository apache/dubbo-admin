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
  <div class="__container_instance_detail">
    <a-empty v-if="instanceMissing" description="实例不存在或已下线" />
    <a-flex>
      <template v-if="!instanceMissing">
        <a-card-grid>
          <a-row :gutter="10">
            <a-col :span="12">
              <a-card class="_detail" style="height: 100%">
                <a-descriptions class="description-column" :column="1">
                  <!-- registerState -->
                  <a-descriptions-item
                    :label="$t('instanceDomain.registerState')"
                    :labelStyle="{ fontWeight: 'bold' }"
                  >
                    <a-typography-paragraph
                      :type="instanceDetail?.registerState === 'Registered' ? 'success' : 'danger'"
                    >
                      {{ instanceDetail?.registerState }}
                    </a-typography-paragraph>
                  </a-descriptions-item>

                  <!-- Register Time -->
                  <a-descriptions-item
                    :label="$t('instanceDomain.registerTime')"
                    :labelStyle="{ fontWeight: 'bold' }"
                  >
                    <a-typography-paragraph>
                      {{ formattedDate(instanceDetail?.registerTime) }}
                    </a-typography-paragraph>
                  </a-descriptions-item>
                </a-descriptions>
              </a-card>
            </a-col>

            <a-col :span="12">
              <a-card class="_detail" style="height: 100%">
                <a-descriptions class="description-column" :column="1">
                  <a-descriptions-item
                    :label="$t('instanceDomain.deployState')"
                    :labelStyle="{ fontWeight: 'bold' }"
                  >
                    <a-tag :color="deployColor(instanceDetail?.deployState)">
                      {{ instanceDetail?.deployState }}
                    </a-tag>
                  </a-descriptions-item>

                  <!-- Start time -->
                  <a-descriptions-item
                    :label="$t('instanceDomain.startTime_k8s')"
                    :labelStyle="{ fontWeight: 'bold' }"
                  >
                    <a-typography-paragraph>
                      {{ formattedDate(instanceDetail?.startTime) }}
                    </a-typography-paragraph>
                  </a-descriptions-item>

                  <!-- Ready time -->
                  <!-- <a-descriptions-item :label="$t('instanceDomain.readyTime_k8s')" :labelStyle="{ fontWeight: 'bold' }">
                  <a-typography-paragraph>
                    {{ formattedDate(instanceDetail?.readyTime) }}
                  </a-typography-paragraph>
                </a-descriptions-item> -->
                </a-descriptions>
              </a-card>
            </a-col>
          </a-row>

          <a-card style="margin-top: 10px" class="_detail">
            <a-descriptions class="description-column" :column="1">
              <!-- instanceIP -->
              <a-descriptions-item
                :label="$t('instanceDomain.instanceIP')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <p @click="copyIt(instanceDetail?.ip)" class="description-item-content with-card">
                  {{ instanceDetail?.ip }}
                  <CopyOutlined />
                </p>
              </a-descriptions-item>

              <!-- deploy cluster -->
              <a-descriptions-item
                :label="$t('instanceDomain.deployCluster')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{ instanceDetail?.deployCluster }}
                </a-typography-paragraph>
              </a-descriptions-item>

              <!-- Dubbo Port -->
              <a-descriptions-item
                :label="$t('instanceDomain.dubboPort')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <p
                  v-if="instanceDetail?.rpcPort"
                  @click="copyIt(instanceDetail?.rpcPort)"
                  class="description-item-content with-card"
                >
                  {{ instanceDetail?.rpcPort }}
                  <CopyOutlined />
                </p>
              </a-descriptions-item>

              <!-- Register cluster -->
              <a-descriptions-item
                :label="$t('instanceDomain.registerCluster')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-space>
                  <a-typography-link v-for="cluster in instanceDetail?.registerClusters">
                    {{ cluster }}
                  </a-typography-link>
                </a-space>
              </a-descriptions-item>

              <!-- whichApplication -->
              <a-descriptions-item
                :label="$t('instanceDomain.whichApplication')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-link @click="checkApplication(instanceDetail?.appName)">
                  {{ instanceDetail?.appName }}
                </a-typography-link>
              </a-descriptions-item>

              <!-- Node IP -->
              <a-descriptions-item
                :label="$t('instanceDomain.node')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <p
                  v-if="instanceDetail?.node"
                  @click="copyIt(instanceDetail?.node)"
                  class="description-item-content with-card"
                >
                  {{ instanceDetail?.node }}
                  <CopyOutlined />
                </p>
              </a-descriptions-item>

              <!-- Owning workload(k8s) -->
              <a-descriptions-item
                :label="$t('instanceDomain.owningWorkload_k8s')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-typography-paragraph>
                  {{ instanceDetail?.workloadName }}
                </a-typography-paragraph>
              </a-descriptions-item>

              <!-- image -->
              <a-descriptions-item
                v-if="instanceDetail?.image"
                :label="$t('instanceDomain.instanceImage_k8s')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-card class="description-item-card">
                  <p
                    @click="copyIt(instanceDetail?.image)"
                    class="description-item-content with-card"
                  >
                    {{ instanceDetail?.image }}
                    <CopyOutlined />
                  </p>
                </a-card>
              </a-descriptions-item>

              <!-- instanceLabel -->
              <a-descriptions-item
                v-if="instanceDetail?.labels && Object.keys(instanceDetail?.labels).length > 0"
                :label="$t('instanceDomain.instanceLabel')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-card class="description-item-card">
                  <a-tag v-for="(value, key) in instanceDetail?.labels">
                    {{ key }} : {{ value }}
                  </a-tag>
                </a-card>
              </a-descriptions-item>

              <!-- health examination -->
              <a-descriptions-item
                v-if="instanceDetail?.probes"
                :label="$t('instanceDomain.healthExamination_k8s')"
                :labelStyle="{ fontWeight: 'bold' }"
              >
                <a-card class="description-item-card">
                  <p class="white_space">
                    启动探针(StartupProbe):{{
                      isProbeOpen(instanceDetail?.probes?.startupProbe.open)
                    }}
                    类型: {{ instanceDetail?.probes?.startupProbe.type }} 端口:{{
                      instanceDetail?.probes?.startupProbe.port
                    }}
                  </p>
                  <p class="white_space">
                    就绪探针(ReadinessProbe):{{
                      isProbeOpen(instanceDetail?.probes?.readinessProbe.open)
                    }}
                    类型: {{ instanceDetail?.probes?.readinessProbe.type }} 端口:{{
                      instanceDetail?.probes?.readinessProbe.port
                    }}
                  </p>
                  <p class="white_space">
                    存活探针(LivenessProbe):{{
                      isProbeOpen(instanceDetail?.probes?.livenessProbe.open)
                    }}
                    类型: {{ instanceDetail?.probes?.livenessProbe.type }} 端口:{{
                      instanceDetail?.probes?.livenessProbe.port
                    }}
                  </p>
                </a-card>
              </a-descriptions-item>
            </a-descriptions>
          </a-card>
        </a-card-grid>
      </template>
    </a-flex>
  </div>
</template>

<script lang="ts" setup>
import { type ComponentInternalInstance, getCurrentInstance, onMounted, reactive, ref } from 'vue'
import { CopyOutlined } from '@ant-design/icons-vue'
import useClipboard from 'vue-clipboard3'
import { message } from 'ant-design-vue'
import { INSTANCE_DEPLOY_COLOR, PRIMARY_COLOR, PRIMARY_COLOR_T } from '@/base/constants'
import { getInstanceDetail } from '@/api/service/instance'
import { useRoute, useRouter } from 'vue-router'
import { formattedDate } from '@/utils/DateUtil'

const route = useRoute()
const router = useRouter()
const apiData: any = reactive({})
const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

let __ = PRIMARY_COLOR
let PRIMARY_COLOR_20 = PRIMARY_COLOR_T('20')

// instance detail information
const instanceDetail = <any>reactive({})
const instanceMissing = ref(false)

onMounted(async () => {
  const { name, pathId } = route.params
  let params = {
    instanceName: name,
    instanceIP: pathId
  }
  try {
    apiData.detail = await getInstanceDetail(params, { silentError: true })
    Object.assign(instanceDetail, apiData.detail.data)
  } catch {
    instanceMissing.value = true
  }
})

// Click on the application name to view the application
const checkApplication = (appName: string) => {
  router.push({
    path: '/resources/applications/detail/' + appName
  })
}

const toClipboard = useClipboard().toClipboard

function copyIt(v: string) {
  message.success(globalProperties.$t('messageDomain.success.copy'))
  toClipboard(v)
}

const isProbeOpen = (status: boolean) => {
  return status ? '开启' : '关闭'
}

const deployColor = (state?: string) => {
  return INSTANCE_DEPLOY_COLOR[(state || 'UNKNOWN').toUpperCase()] || 'default'
}
</script>

<style lang="less" scoped>
.__container_instance_detail {
}
</style>
