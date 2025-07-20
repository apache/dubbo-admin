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
  <div class="__container_layout_bread">
    <a-breadcrumb>
      <a-breadcrumb-item v-for="r in routes">{{ $t(r.name) }}</a-breadcrumb-item>
      <a-breadcrumb-item v-if="pathId">{{ pathId }}</a-breadcrumb-item>
    </a-breadcrumb>
  </div>
</template>

<script setup lang="ts">
import { useRoute, useRouter } from 'vue-router'
import { computed } from 'vue'

const route = useRoute()
const router = useRouter()

let pathId = computed(() => {
  return route.params?.pathId ? route.params.pathId : ''
})
const routes = computed(() => {
  return route.matched.slice(1).map((x, idx) => {
    return {
      name: <string>x.name
      // handle: router.push(x)
    }
  })
})
</script>
<style lang="less" scoped>
.__container_layout_bread {
  padding-left: 20px;
  padding-top: 10px;
}
</style>
