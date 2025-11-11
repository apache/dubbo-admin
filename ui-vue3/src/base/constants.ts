/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import type { Component } from 'vue'
import { computed, h, reactive, ref } from 'vue'
import type { RouteRecordType } from '@/router/defaultRoutes'
import type { RouteLocationNormalizedLoaded } from 'vue-router'

// 2aacb8
export const PRIMARY_COLOR_DEFAULT = '#17b392'

export const LOCAL_STORAGE_LOCALE = 'LOCAL_STORAGE_LOCALE'
export const LOCAL_STORAGE_THEME = 'LOCAL_STORAGE_THEME'

let item = localStorage.getItem(LOCAL_STORAGE_THEME)

/**
 * 根据背景色自动计算适合的文字颜色（黑或白）
 * @param {string} hex 背景色十六进制（如 '#17b392' 或 '17b392'）
 * @returns {string} 文字色十六进制（'#000000' 或 '#ffffff'）
 */
function getTextColorByBackground(hex: string) {
  // 处理十六进制格式（去掉#）
  hex = hex.replace('#', '')
  // 转换为RGB
  const r = parseInt(hex.substring(0, 2), 16)
  const g = parseInt(hex.substring(2, 4), 16)
  const b = parseInt(hex.substring(4, 6), 16)

  // 计算相对亮度（WCAG标准）
  const [rNorm, gNorm, bNorm] = [r, g, b].map((c) => {
    const val = c / 255
    return val <= 0.03928 ? val / 12.92 : Math.pow((val + 0.055) / 1.055, 2.4)
  })
  const L = 0.2126 * rNorm + 0.7152 * gNorm + 0.0722 * bNorm

  // 根据亮度返回文字色
  return L > 0.5 ? '#131313' : '#e3e1e1'
}

export const PRIMARY_COLOR = ref(item || PRIMARY_COLOR_DEFAULT)
export const PRIMARY_COLOR_T = (percent: string) => computed(() => PRIMARY_COLOR.value + percent)
export const PRIMARY_COLOR_R = computed(() => getTextColorByBackground(PRIMARY_COLOR.value))

export const INSTANCE_REGISTER_COLOR: { [key: string]: string } = {
  HEALTHY: 'green',
  REGISTED: 'green'
}

export const TAB_HEADER_TITLE: Component = {
  functional: true,
  props: ['route'],
  render: (
    a: any,
    b: any,
    c: { [key: string]: RouteRecordType & RouteLocationNormalizedLoaded }
  ) => {
    let route = c.route
    let header: any = route.meta?.slots?.header
    return h(header) || h('div', route.params?.pathId)
    // console.log(h)
    // return h("div", "foo")
  }
}

/**
 * 'Running','Pending', 'Terminating', 'Crashing'
 */
export const INSTANCE_DEPLOY_COLOR: { [key: string]: string } = {
  RUNNING: 'green',
  PENDING: 'yellow',
  TERMINATING: 'red',
  CRASHING: 'darkRed'
}
