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
  <!--      example like blow-->
  <div class="__container_AppTabHeaderSlot">
    <a-row :gutter="12" align="middle">
      <a-col flex="none">
        <span class="header-desc">{{ $t('instanceDomain.name') }}: {{ route.params?.name }}</span>
      </a-col>
      <a-col flex="none">
        <a-tag :color="lifecycleColor(instanceLifecycleState)">
          {{ instanceLifecycleState }}
        </a-tag>
      </a-col>
    </a-row>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { getInstanceDetail } from '@/api/service/instance'
import { INSTANCE_LIFECYCLE_COLOR } from '@/base/constants'

const route = useRoute()
const instanceLifecycleState = ref('Unknown')

const lifecycleColor = (state?: string) => {
  return INSTANCE_LIFECYCLE_COLOR[(state || 'UNKNOWN').toUpperCase()] || 'default'
}

const fetchInstanceLifecycleState = async () => {
  const instanceName = route.params?.name
  if (!instanceName) {
    instanceLifecycleState.value = 'Unknown'
    return
  }

  try {
    const { data } = await getInstanceDetail(
      {
        instanceName,
        instanceIP: route.params?.pathId
      },
      {
        silentError: true
      }
    )
    instanceLifecycleState.value = data?.lifecycleState || 'Unknown'
  } catch {
    instanceLifecycleState.value = 'Unknown'
  }
}

watch(() => [route.params?.name, route.params?.pathId], fetchInstanceLifecycleState, {
  immediate: true
})
</script>
<style lang="less" scoped>
.__container_AppTabHeaderSlot {
  .header-desc {
    line-height: 30px;
    vertical-align: center;
  }
}
</style>
