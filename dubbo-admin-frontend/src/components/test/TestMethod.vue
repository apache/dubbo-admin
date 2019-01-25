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
  <v-container grid-list-xl fluid>
    <v-layout row wrap>
      <v-flex class="test-form" lg12 xl6>
        <v-card>
          <v-card-title class="headline">Test: {{service}}#{{method.name}}</v-card-title>
          <v-card-text>
            <json-editor id="test" v-model="method.json"/>
          </v-card-text>
          <v-card-actions>
            <v-spacer></v-spacer>
            <v-btn id="execute" mt-0 color="primary" @click="executeMethod()">EXECUTE</v-btn>
          </v-card-actions>
        </v-card>
      </v-flex>
      <v-flex class="test-result" lg12 xl6>
        <v-card>
          <v-card-title class="headline">Result</v-card-title>
          <v-card-text>
            <json-editor v-model="result" name="Result" readonly></json-editor>
          </v-card-text>
        </v-card>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'

  export default {
    name: 'TestMethod',
    components: {
      JsonEditor
    },
    data () {
      return {
        service: this.$route.query['service'],
        application: this.$route.query['application'],
        method: {
          name: null,
          parameterTypes: [],
          json: []
        },
        result: null
      }
    },
    methods: {
      executeMethod () {
        let serviceTestDTO = {
          service: this.service,
          method: this.method.name,
          parameterTypes: this.method.parameterTypes,
          params: this.method.json
        }
        this.$axios.post('/test', serviceTestDTO).then(response => {
          if (response.status === 200) {
            this.result = response.data
          }
        })
      }
    },
    mounted () {
      const query = this.$route.query
      const method = query['method']

      if (method) {
        const [methodName, parametersTypes] = method.split('~')
        this.method.name = methodName
        this.method.parameterTypes = parametersTypes.split(';')
      }

      this.$axios.get('/test/method', {
        params: {
          application: this.application,
          service: this.service,
          method: method
        }
      }).then(response => {
        this.method.json = response.data.parameterTypes
      })
    }
  }
</script>

<style scoped>
</style>
