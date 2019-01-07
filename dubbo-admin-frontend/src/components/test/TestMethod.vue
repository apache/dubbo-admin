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
      <v-flex lg12>
      </v-flex>
      <v-flex lg12>
        <v-card>
          <v-flex lg8>
            <v-card-title>
              <span>Service: {{service}}<br> Method: {{method.name}}</span>
            </v-card-title>
            <v-card-text>
              <json-editor id="test" v-model="method.json"/>
            </v-card-text>
            <v-card-actions>
              <v-spacer></v-spacer>
              <v-btn id="execute" mt-0 color="primary" @click="executeMethod()">EXECUTE</v-btn>
            </v-card-actions>
          </v-flex>
          <v-flex lg8>
            <v-card-text>
              <h2>Test Result</h2>
              <json-editor id="result" v-model="result" v-if="showResult"></json-editor>
            </v-card-text>
          </v-flex>
        </v-card>
      </v-flex>

    </v-layout>

  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  export default {
    name: 'TestMethod',
    data () {
      return {
        basic: [],
        service: null,
        application: null,
        method: {
          name: null,
          parameterTypes: [],
          json: []
        },
        showResult: false,
        result: {}
      }
    },

    methods: {
      executeMethod: function () {
        let serviceTestDTO = {}
        serviceTestDTO.service = this.service
        serviceTestDTO.method = this.method.name
        serviceTestDTO.parameterType = this.method.parameterTypes
        serviceTestDTO.params = this.method.json
        this.$axios.post('/test', serviceTestDTO)
          .then(response => {
            if (response.status === 200) {
              this.result = response.data
              this.showResult = true
            }
          })
      }
    },

    mounted: function () {
      let query = this.$route.query
      let vm = this
      let method = null
      Object.keys(query).forEach(function (key) {
        if (key === 'service') {
          let item = {}
          item.name = 'service'
          item.value = query[key]
          vm.basic.push(item)
          vm.service = query[key]
        }
        if (key === 'method') {
          let item = {}
          item.name = 'method'
          item.value = query[key]
          vm.method.name = query[key].split('~')[0]
          method = query[key]
          vm.basic.push(item)
        }
        if (key === 'application') {
          vm.application = query[key]
        }
      })
      this.$axios.get('/test/method', {
        params: {
          application: vm.application,
          service: vm.service,
          method: method
        }
      }).then(response => {
        this.method.json = response.data.parameterTypes
      })
    },
    components: {
      JsonEditor
    }

  }
</script>

<style scoped>
</style>
