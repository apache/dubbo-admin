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
  <EventTimeline
    :events="eventList"
    :loading="loading"
    :loadingMore="loadingMore"
    :hasMore="hasMore"
    @loadMore="loadEvents()"
  />
</template>

<script lang="ts" setup>
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useMeshStore } from '@/stores/mesh'
import { listServiceEvent } from '@/api/service/service'
import EventTimeline from '@/components/EventTimeline.vue'
import type { EventItem } from '@/types/api'

const route = useRoute()
const meshStore = useMeshStore()
const defaultPageSize = 20

const eventList = ref<EventItem[]>([])
const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(false)
const pageOffset = ref(0)

const loadEvents = async (reset = false) => {
  if (reset) {
    loading.value = true
  } else if (loading.value || loadingMore.value || !hasMore.value) {
    return
  } else {
    loadingMore.value = true
  }
  try {
    const serviceName = (route.params.pathId as string) || ''
    const mesh = meshStore.mesh || 'default'
    const currentOffset = reset ? 0 : pageOffset.value
    const res = await listServiceEvent({
      serviceName,
      mesh,
      pageOffset: currentOffset,
      pageSize: defaultPageSize
    })
    const list = res?.data?.list || []
    const total = res?.data?.total || 0
    eventList.value = reset ? list : [...eventList.value, ...list]
    pageOffset.value = currentOffset + list.length
    hasMore.value = pageOffset.value < total && list.length > 0
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

onMounted(async () => {
  pageOffset.value = 0
  eventList.value = []
  hasMore.value = true
  await loadEvents(true)
})
</script>
