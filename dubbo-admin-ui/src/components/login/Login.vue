<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  - the License.  You may obtain a copy of the License at
  -
  -     http://www.apache.org/licenses/LICENSE-2.0
  -
  - Unless required by applicable law or agreed to in writing, software
  - distributed under the License is distributed on an "AS IS" BASIS,
  - WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  - See the License for the specific language governing permissions and
  - limitations under the License.
  -->

<template>
  <v-container fluid fill-height class="login-overlay">
    <v-layout flex align-center justify-center>
      <v-flex xs12 sm8 lg5 elevation-6>
        <v-toolbar class="pt-5 blue darken-4">
          <v-toolbar-items>
            <v-toolbar-title class="white--text">
              <h4>Log in to Dubbo OPS</h4>
            </v-toolbar-title>
          </v-toolbar-items>
        </v-toolbar>
        <v-card>
          <v-card-text class="pt-4">
            <div>
              <v-form v-model="valid" ref="form">
                <v-text-field
                  label="Enter your username"
                  v-model="username"
                  :rules="usernameRules"
                  required
                ></v-text-field>
                <v-text-field
                  label="Enter your password"
                  v-model="password"
                  :append-icon="!passwordVisible ? 'visibility' : 'visibility_off'"
                  @click:append="() => (passwordVisible = !passwordVisible)"
                  :type="!passwordVisible ? 'password' : 'text'"
                  :rules="passwordRules"
                  counter
                  required
                ></v-text-field>
                <v-layout justify-space-between>
                  <v-btn @click="submit" :class=" { 'blue darken-4 white--text': valid, disabled: !valid }">Login
                  </v-btn>
                </v-layout>
              </v-form>
            </div>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  export default {
    name: 'Login',
    data () {
      return {
        valid: false,
        username: '',
        password: '',
        passwordVisible: false,
        passwordRules: [
          (v) => !!v || 'Password is required'
        ],
        usernameRules: [
          (v) => !!v || 'User name is required'
        ]
      }
    },
    methods: {
      submit () {}
    }
  }
</script>

<style scoped>
  .login-overlay {
  }
</style>
