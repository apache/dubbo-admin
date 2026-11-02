<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

<template>
  <!-- GrafanaPage owns loading, Not Configured, and error states before showing the iframe. -->
  <div class="application-log-tab">
    <TraceLogSearch />
    <GrafanaPage />
  </div>
</template>

<script setup lang="ts">
import { provide, reactive } from 'vue'
import { useRoute } from 'vue-router'
import GrafanaPage from '@/components/GrafanaPage.vue'
import TraceLogSearch from '@/components/TraceLogSearch.vue'
import { getApplicationLogDashboard } from '@/api/service/app'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
import type { GrafanaState } from '@/types/grafana'

const route = useRoute()
provide<GrafanaState>(
  PROVIDE_INJECT_KEY.GRAFANA,
  reactive({
    api: getApplicationLogDashboard,
    showIframe: false,
    params: { appName: route.params.pathId as string }
  })
)
</script>
