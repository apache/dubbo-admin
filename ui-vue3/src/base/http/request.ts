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
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { removeAuthState } from '@/utils/AuthUtil'

const service: AxiosInstance = axios.create({
  //  change this to decide where to go
  // baseURL: '/mock',
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
    console.log(config.data)

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
    // Success case - code is 'Success'
    if (
      response.status === 200 &&
      (response.data.code === 'Success')
    ) {
      return Promise.resolve(response.data)
    }
    // Handle 401 unauthorized
    if (response.status === 401) {
      removeAuthState()
    }
    // Show error toast message
    const errorMsg = `${response.data.code}:${response.data.message}`
    message.error(errorMsg)
    console.error(errorMsg)
    return Promise.reject(response.data)
  },
  (error) => {
    NProgress.done()
    // Handle error response with data
    if (error.response?.data) {
      const errorMsg = `${error.response.data.code}:${error.response.data.message}`
      message.error(errorMsg)
      console.error(errorMsg)
    } else {
      // Handle network or other errors
      message.error('NetworkError:请求失败，请检查网络连接')
      console.error(error)
    }
    return Promise.reject(error.response?.data)
  }
)
export default service
