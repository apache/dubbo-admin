/* Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
 * http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 */
import { createApp, h, ref } from 'vue'
import { createPinia } from 'pinia'
import { createI18n } from 'vue-i18n'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import '@/style.css'
import AgentDrawer from '@/components/AgentDrawer.vue'
import en from '@/base/i18n/en'
import { loadAuthConfiguration } from '@/auth/session'

// Separate Vite development entry; absent from the production build/router.
async function bootstrap() {
  const app = createApp({
    setup() {
      const open = ref(true)
      return () =>
        h('main', [
          h(
            'button',
            {
              onClick: () => {
                open.value = true
              }
            },
            'Open assistant'
          ),
          h(AgentDrawer, {
            agentDrawerOpen: open.value,
            'onUpdate:agentDrawerOpen': (value: boolean) => {
              open.value = value
            }
          })
        ])
    }
  })
  // The Console request client reads the mesh store, so pinia must be active
  // before the auth configuration request runs.
  app.use(createPinia())
  // Mirror main.ts: discover whether Admin issues AI access tokens before any
  // AI request, so chat and Memory authenticate with a Bearer token.
  await loadAuthConfiguration()
  app
    .use(Antd)
    .use(createI18n({ legacy: false, locale: 'en', messages: { en } }))
    .mount('#app')
}

bootstrap()
