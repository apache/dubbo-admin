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

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { login, type AuthProvider } from '@/api/service/login'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { i18n } from '@/base/i18n'
import { useMeshStore } from '@/stores/mesh'
import { meshesSearch } from '@/api/service/globalSearch'
import { loadAuthConfiguration, providerLoginURL, syncAuthenticatedPrincipal } from '@/auth/session'

defineOptions({ name: 'AdminLogin' })

const userinfo = reactive({ username: '', password: '' })
const methods = ref<string[]>([])
const providers = ref<AuthProvider[]>([])
const loadingConfiguration = ref(true)
const passwordEnabled = computed(() => methods.value.includes('password'))
const router = useRouter()
const route = useRoute()
const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
const meshStore = useMeshStore()

onMounted(async () => {
  try {
    const configuration = await loadAuthConfiguration()
    methods.value = configuration.methods ?? []
    providers.value = configuration.providers ?? []
    try {
      await syncAuthenticatedPrincipal()
      await router.replace(redirect)
    } catch {
      // No active Admin session: remain on the login page.
    }
  } catch {
    message.error(i18n.global.t('loginDomain.authFail'))
  } finally {
    loadingConfiguration.value = false
  }
})

async function loginHandle() {
  const formData = new FormData()
  formData.append('user', userinfo.username)
  formData.append('password', userinfo.password)
  try {
    await login(formData)
    await syncAuthenticatedPrincipal()
    const { data } = await meshesSearch()
    if (!meshStore.mesh || !data.some((item: any) => item.id === meshStore.mesh)) {
      meshStore.mesh = data[0]?.id
    }
    await router.replace(redirect)
  } catch {
    message.error(i18n.global.t('loginDomain.authFail'))
  }
}

function providerLogin(providerID: string) {
  window.location.assign(providerLoginURL(providerID))
}
</script>

<template>
  <div class="background">
    <a-card class="login" :loading="loadingConfiguration">
      <a-row class="title">
        <div>用户登录</div>
      </a-row>
      <a-row v-if="passwordEnabled" class="password-method">
        <a-form class="password-form" layout="vertical" :model="userinfo" ref="login-form-ref">
          <a-form-item
            class="item"
            :label="$t('loginDomain.username')"
            name="username"
            :rules="[{ required: true }]"
          >
            <a-input v-model:value="userinfo.username"></a-input>
          </a-form-item>
          <a-form-item
            class="item"
            :label="$t('loginDomain.password')"
            name="password"
            :rules="[{ required: true }]"
          >
            <a-input type="password" v-model:value="userinfo.password"></a-input>
          </a-form-item>
          <a-form-item class="item" label="">
            <a-button @click="loginHandle" size="large" type="primary" class="login-btn">
              {{ $t('loginDomain.login') }}
            </a-button>
          </a-form-item>
        </a-form>
      </a-row>
      <a-row v-if="providers.length" class="provider-list">
        <a-button
          v-for="provider in providers"
          :key="provider.id"
          size="large"
          class="provider-btn"
          @click="providerLogin(provider.id)"
        >
          <img
            v-if="provider.iconUrl"
            class="provider-icon"
            :src="provider.iconUrl"
            alt=""
            aria-hidden="true"
          />
          <span class="provider-label">{{ provider.displayName }}</span>
        </a-button>
      </a-row>
    </a-card>
  </div>
</template>

<style scoped lang="less">
.background {
  background: url('assets/login.jpg') no-repeat center center fixed;
  background-size: cover;
  width: 100vw;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;

  .login {
    background-color: #fff;
    padding: 20px;
    border-radius: 12px;
    box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
    min-width: 350px;
    width: 22vw;

    .title {
      width: 100%;
      justify-content: center;
      font-size: 20px;
      font-weight: 500;
      margin-bottom: 20px;
    }

    .login-btn,
    .provider-btn {
      width: 100%;
    }

    .password-method,
    .password-form,
    .provider-list {
      width: 100%;
    }

    .provider-list {
      display: grid;
      gap: 12px;
      margin-top: 12px;
    }

    .provider-btn {
      position: relative;
    }

    .provider-icon {
      position: absolute;
      top: 50%;
      left: 16px;
      width: 20px;
      height: 20px;
      transform: translateY(-50%);
    }
  }
}
</style>
