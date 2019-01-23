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
      <v-flex xs12>
        <search v-model="filter" label="Search by service name" :submit="search"></search>
      </v-flex>
      <v-flex xs12>
        <h3>Methods</h3>
      </v-flex>
      <v-flex xs12>
        <v-data-table :headers="headers" :items="methods" hide-actions class="elevation-1">
          <template slot="items" slot-scope="props">
            <td>{{ props.item.name }}</td>
            <td><v-chip xs v-for="(type, index) in props.item.parameterTypes" :key="index" label>{{ type }}</v-chip></td>
            <td><v-chip label>{{ props.item.returnType }}</v-chip></td>
            <td class="text-xs-right">
              <v-tooltip bottom>
                <v-btn
                  fab dark small color="blue" slot="activator"
                  :href="getHref(props.item.application, props.item.service, props.item.signature)"
                >
                  <v-icon>edit</v-icon>
                </v-btn>
                <span>Try it</span>
              </v-tooltip>
            </td>
          </template>
        </v-data-table>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  import JsonEditor from '@/components/public/JsonEditor'
  import Search from '@/components/public/Search'

  export default {
    name: 'ServiceTest',
    components: {
      JsonEditor,
      Search
    },
    data () {
      return {
        filter: this.$route.query['service'] || '',
        headers: [
          {
            text: 'Method Name',
            value: 'method',
            sortable: false
          },
          {
            text: 'Parameter List',
            value: 'parameter',
            sortable: false
          },
          {
            text: 'Return Type',
            value: 'returnType',
            sortable: false
          },
          {
            text: '',
            value: 'operation',
            sortable: false
          }
        ],
        service: null,
        methods: []
      }
    },
    methods: {
      search () {
        if (!this.filter) {
          return
        }
        this.$router.replace({
          query: { service: this.filter }
        })
        this.$axios.get('/service/' + this.filter).then(response => {
          this.service = response.data
          this.methods = []
          if (this.service.metadata) {
            let methods = this.service.metadata.methods
            for (let i = 0; i < methods.length; i++) {
              let method = {}
              let sig = methods[i].name + '~'
              let parameters = methods[i].parameterTypes
              let length = parameters.length
              for (let j = 0; j < length; j++) {
                sig = sig + parameters[j]
                if (j !== length - 1) {
                  sig = sig + ';'
                }
              }
              method.signature = sig
              method.name = methods[i].name
              method.parameterTypes = methods[i].parameterTypes
              method.returnType = methods[i].returnType
              method.service = response.data.service
              method.application = response.data.application
              this.methods.push(method)
            }
          }
        }).catch(error => {
          this.showSnackbar('error', error.response.data.message)
        })
      },
      getHref (application, service, method) {
        return `/#/testMethod?application=${application}&service=${service}&method=${method}`
      }
    },
    created () {
      this.search()
    }
  }
</script>
