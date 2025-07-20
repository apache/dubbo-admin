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
  <div class="__container_menu">
    <a-menu
      mode="inline"
      :selectedKeys="selectedKeys"
      :open-keys="openKeys"
      @select="selectMenu"
      :items="items"
      @click="handleClick"
    >
    </a-menu>
  </div>
</template>

<script setup lang="ts">
import type { RouteRecordType } from '@/router/defaultRoutes'
import { routes as defaultRoutes } from '@/router/defaultRoutes'
import type { ItemType, MenuProps } from 'ant-design-vue'
import type { ComponentInternalInstance } from 'vue'
import { computed, getCurrentInstance, h, reactive } from 'vue'
import { Icon } from '@iconify/vue'
import { useRoute, useRouter } from 'vue-router'
import type { RouterMeta } from '@/router/RouterMeta'

const {
  appContext: {
    config: { globalProperties }
  }
} = <ComponentInternalInstance>getCurrentInstance()

const routesForMenu = defaultRoutes

const nowRoute = useRoute()

// load active menu
let selectedKeys = computed(() => [getLoadSelectedKeys(nowRoute.meta)])
let openKeys: any = reactive([])
function getLoadSelectedKeys(meta: RouterMeta): string {
  return meta.tab || meta.hidden ? getLoadSelectedKeys(meta.parent?.meta!) : meta._router_key!
}
function loadOpenedKeys() {
  let p: any = nowRoute.meta.parent
  while (p) {
    openKeys.push(p.meta?._router_key)
    p = p.meta?.parent
  }
}

loadOpenedKeys()

function selectMenu(e: any) {
  selectedKeys[0] = e.key
}

function getItem(
  label: any,
  title: any,
  key?: string,
  icon?: any,
  children?: ItemType[],
  type?: 'group'
): ItemType {
  if (icon) {
    icon = h(Icon, { icon: icon })
  }

  return {
    key,
    title,
    icon,
    children,
    label: computed(() => globalProperties.$t(label)),
    type
  } as ItemType
}

const items: ItemType[] = reactive([])

/**
 * pre handle routes
 * 1. if a route doesn't have any child, skip loop
 * 2. if a route has only one child, replace
 *
 * @param arr
 * @param arr2
 */
function prepareRoutes(
  arr: readonly RouteRecordType[] | undefined,
  arr2: ItemType[],
  parent = 'root'
) {
  if (!arr || arr.length === 0) return
  for (let r of arr) {
    if (r.meta?.skip) {
      prepareRoutes(r.children, arr2, r.name)
      continue
    }
    if (!r.meta?.hidden) {
      if (!r.children || r.children.length === 0 || r.meta?.tab_parent) {
        arr2.push(getItem(r.name, r.path, r.meta?._router_key, r.meta?.icon))
      } else if (r.children.length === 1) {
        arr2.push(
          getItem(r.children[0].name, r.path, r.meta?._router_key, r.children[0].meta?.icon)
        )
      } else {
        const tmp: ItemType[] = reactive([])
        prepareRoutes(r.children, tmp, r.name)
        arr2.push(getItem(r.name, r.path, r.meta?._router_key, r.meta?.icon, tmp))
      }
    }
  }
}

prepareRoutes(routesForMenu, items)

const router = useRouter()

const handleClick: MenuProps['onClick'] = (e) => {
  // console.log(e.item?.title)
  router.push(<string>e.item?.title)
}
</script>
<style lang="less" scoped>
.__container_menu {
  .icon-wrapper {
    .icon {
      font-size: 20px;
      margin-right: 5px;
      margin-bottom: -4px;
      font-weight: 700;
    }
  }
}
</style>
