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

import { createI18n } from 'vue-i18n'
import { LOCAL_STORAGE_LOCALE } from '@/base/constants'
import { messages } from '@/base/i18n/messages'
import { reactive } from 'vue'

export const localeConfig = reactive({
  // todo use system's locale
  locale: localStorage.getItem(LOCAL_STORAGE_LOCALE) || 'cn',
  opts: [
    {
      value: 'en',
      title: 'en'
    },
    {
      value: 'cn',
      title: '中文'
    }
  ]
})

export const i18n: any = createI18n({
  locale: localeConfig.locale,
  legacy: false,
  globalInjection: true,
  messages
})

export const changeLanguage = (l: any) => {
  localStorage.setItem(LOCAL_STORAGE_LOCALE, l)
  i18n.global.locale.value = l
}
