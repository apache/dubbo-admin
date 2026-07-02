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

import type {
  AxiosInstance,
  AxiosInterceptorManager,
  AxiosRequestHeaders,
  AxiosResponse,
  InternalAxiosRequestConfig
} from 'axios'
import axios from 'axios'
import NProgress from 'nprogress'
import { removeAuthState } from '@/utils/AuthUtil'
import router from '@/router'
import { useMeshStore } from '@/stores/mesh'
import { message } from 'ant-design-vue'
import { HTTP_STATUS } from './constants'

// 白名单：对于这些 URL 不进行 message 错误提示 不要 URL 前面的 /
const SILENT_ERROR_URLS = ['promQL/query']

// 检查 URL 是否在静默错误白名单中
const isSilentErrorUrl = (url?: string): boolean => {
  if (!url) return false
  return SILENT_ERROR_URLS.some((silentUrl) => url.includes(silentUrl))
}

const shouldShowErrorMessage = (url?: string): boolean => {
  return !isSilentErrorUrl(url)
}

const service: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 30 * 1000
})
const request: AxiosInterceptorManager<InternalAxiosRequestConfig> = service.interceptors.request
const response: AxiosInterceptorManager<AxiosResponse> = service.interceptors.response

request.use(
  (config) => {
    if (!config.headers) {
      config.headers = <AxiosRequestHeaders>{}
    }
    if (!config.headers['Content-Type']) {
      config.headers['Content-Type'] = 'application/json'
      config.data = JSON.stringify(config.data)
    }
    if (!config.params) {
      config.params = {}
    }
    const { mesh } = useMeshStore()
    config.params['mesh'] = mesh
    // NProgress.start()
    // console.log(config)
    return config
  },
  (error) => {
    Promise.reject(error)
  }
)
const rejectState: { errorHandler: Function | null } = {
  errorHandler: null
}

response.use(
  (response) => {
    NProgress.done()
    // Success case: code is 'Success' (production) or 200 (mock mode compatibility)
    if (response.status === 200 && response.data.code === HTTP_STATUS.SUCCESS) {
      return Promise.resolve(response.data)
    }

    // Show error toast message
    const errorMsg = `${response.data.code}:${response.data.message}`
    if (shouldShowErrorMessage(response.config.url)) {
      message.error(errorMsg)
    }
    console.error(errorMsg)
    return Promise.reject(response.data)
  },
  (error) => {
    NProgress.done()
    // Handle error response with data
    const response = error.response

    // Handle 401 unauthorized
    if (response?.status === 401) {
      removeAuthState()
      try {
        const current = router.currentRoute?.value
        const redirectPath = current?.fullPath || current?.path || '/'

        if (!redirectPath.startsWith('/login')) {
          router.push({ path: `/login?redirect=${encodeURIComponent(redirectPath)}` })
        }
      } catch (e) {
        console.error('Router push failed during 401 redirect:', e)
        if (!window.location.pathname.startsWith('/login')) {
          window.location.href = `/login?redirect=${encodeURIComponent(window.location.pathname)}`
        }
      }
      return Promise.reject(error.response?.data)
    }
    // Handle success status in error response
    if (response?.data?.code === HTTP_STATUS.SUCCESS) {
      return Promise.resolve(response.data)
    }
    if (response?.status === 401) {
      return Promise.reject(error.response?.data)
    }
    if (response?.data) {
      const errorMsg = `${response.data?.code}:${response.data?.message}`
      if (shouldShowErrorMessage(error.config?.url)) {
        message.error(errorMsg)
      }
      console.error(errorMsg)
    } else {
      // Handle network or other errors
      if (!isSilentErrorUrl(error.config?.url)) {
        message.error('NetworkError:请求失败，请检查网络连接')
      }
      console.error(error)
    }
    return Promise.reject(error.response?.data)
  }
)
export default service
