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
  <div class="__container_router_tab_index">
    <div :key="key">
      <div v-if="tabRoute.meta.tab" class="header">
        <a-row>
          <a-col :span="1">
            <span @click="router.replace(tabRoute.meta.back || '../')" style="float: left">
              <Icon icon="material-symbols:keyboard-backspace-rounded" class="back" />
            </span>
          </a-col>
          <a-col :span="18">
            <TAB_HEADER_TITLE :route="tabRoute" />
          </a-col>
        </a-row>
        <a-tabs @change="router.push({ name: activeKey || '' })" v-model:activeKey="activeKey">
          <a-tab-pane :key="v.name" v-for="v in tabRouters.filter((x: any) => !x.meta.hidden)">
            <template #tab>
              <span>
                <Icon style="margin-bottom: -2px" :icon="v.meta.icon"></Icon>
                {{ $t(v.name) }}
              </span>
            </template>
          </a-tab-pane>
        </a-tabs>
      </div>

      <a-spin class="tab-spin" :spinning="transitionFlag">
        <div
          id="layout-tab-body"
          style="
            transition: scroll-top 0.5s ease;
            overflow: auto;
            height: calc(100vh - 300px);
            padding-bottom: 20px;
          "
        >
          <router-view :key="routerKey" v-if="!transitionFlag" />
        </div>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, provide, reactive, ref } from 'vue'
import { Icon } from '@iconify/vue'
import { useRoute, useRouter } from 'vue-router'
import _ from 'lodash'

import { PRIMARY_COLOR, TAB_HEADER_TITLE } from '@/base/constants'
import { PROVIDE_INJECT_KEY } from '@/base/enums/ProvideInject'
const TAB_STATE = reactive({})
provide(PROVIDE_INJECT_KEY.PROVIDE_INJECT_KEY, TAB_STATE)
const router = useRouter()
const tabRoute = useRoute()
let __ = PRIMARY_COLOR

let meta: any = tabRoute.meta
const tabRouters = computed(() => {
  let meta: any = tabRoute.meta
  return meta?.parent?.children?.filter((x: any): any => x.meta.tab)
})
const routerKey = computed(() => {
  return tabRoute.name + '_' + _.uniqueId()
})
let activeKey = ref(tabRoute.name)
let transitionFlag = ref(false)
let key = _.uniqueId('__tab_page')
router.beforeEach((to, from, next) => {
  key = _.uniqueId('__tab_page')
  transitionFlag.value = true
  activeKey.value = <string>to.name
  next()
  setTimeout(() => {
    transitionFlag.value = false
  }, 500)
})
</script>
<style lang="less" scoped>
.__container_router_tab_index {
  :deep(.tab-spin) {
    margin-top: 20vh;
  }

  :deep(.ant-tabs-nav) {
    margin: 0;
  }

  .header {
    background: #fafafa;
    padding: 20px 20px 0 20px;
    border-radius: 10px;
    margin-bottom: 20px;
  }

  .back {
    font-size: 24px;
    margin-bottom: -2px;
    color: v-bind('PRIMARY_COLOR');
  }
}
</style>
