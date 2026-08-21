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
import { createApp } from 'vue'
import Antd from 'ant-design-vue'
import '@/assets/iconfont/iconfont.css'
import '@/style.css'
import router from './router'
import App from './App.vue'
import 'ant-design-vue/dist/reset.css'
import { i18n } from '@/base/i18n'

import Vue3ColorPicker from 'vue3-colorpicker'

import 'vue3-colorpicker/style.css'
import 'nprogress/nprogress.css'
import piniaPluginPersistedstate from 'pinia-plugin-persistedstate'

import { getAuthState, updateAuthState } from '@/utils/AuthUtil'
import { createPinia } from 'pinia'

async function bootstrap() {
  if (import.meta.env.VITE_MOCK_ENABLED === 'true') {
    const { worker, workerStartOptions } = await import('./mocks/browser')
    await worker.start(workerStartOptions)
    updateAuthState(true, 'admin')
    console.info('[Mock Mode] MSW enabled, auto-logged in as admin')
  }

  const app = createApp(App)

  const pinia = createPinia()

  pinia.use(piniaPluginPersistedstate)

  app.use(Antd).use(Vue3ColorPicker).use(pinia).use(i18n).use(router).mount('#app')

  router.beforeEach((to, from, next) => {
    const authState = getAuthState()
    if (authState?.state || to.path.startsWith('/login')) {
      next()
    } else {
      next({ path: `/login?redirect=${to.path}` })
    }
  })
}

bootstrap()
