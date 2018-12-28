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
        <v-data-table
          :items="basic"
          class="elevation-1"
          hide-actions
          hide-headers >
          <template slot="items" slot-scope="props">
            <td><h4>{{props.item.name}}</h4></td>
            <td>{{props.item.value}}</td>
          </template>
        </v-data-table>
      </v-flex>
      <v-flex lg12>
        <v-card>
          <v-toolbar flat color="transparent" class="elevation-0">
            <v-toolbar-title><h3>Please fill in parameters</h3></v-toolbar-title>
          </v-toolbar>
          <v-layout row wrap>
            <v-flex lg10>
              <json-editor id="test"/>
            </v-flex>
            <v-toolbar flat color="transparent" class="elevation-0">
              <v-toolbar-title><h3>Test Result</h3></v-toolbar-title>
            </v-toolbar>
            <v-flex lg10>
              <json-editor id="result" v-if="method.showResult"/>
            </v-flex>
          </v-layout>
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
        method: {
          name: null,
          types: [],
          json: [],
          showResult: false
        }
      }
    },

    mounted: function () {
      let query = this.$route.query
      let vm = this
      Object.keys(query).forEach(function (key) {
        if (key === 'service') {
          let item = {}
          item.name = 'service'
          item.value = query[key]
          vm.basic.push(item)
        }
        if (key === 'method') {
          let item = {}
          item.name = 'method'
          item.value = query[key]
          vm.method.name = query[key].split('~')[0]
          let sig = query[key].split('~')[1]
          vm.types = sig.split(';')
          vm.types.forEach(function (item) {
            vm.json.push(item)
          })

          vm.basic.push(item)
        }
      })
    },
    components: {
      JsonEditor
    }

  }
</script>

<style scoped>

</style>
