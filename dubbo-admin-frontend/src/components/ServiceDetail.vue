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
  <v-container grid-list-xl fluid >
    <v-layout row wrap >
      <v-flex sm12>
        <h3>Basic Info</h3>
      </v-flex>
      <v-flex lg12>
        <v-data-table
          :items="basic"
          class="elevation-1"
          hide-actions
          hide-headers >
          <template slot="items" slot-scope="props">
            <td>{{props.item.role}} </td>
            <td>{{props.item.value}}</td>
          </template>
        </v-data-table>
      </v-flex>
      <v-flex sm12>
        <h3>Service Info</h3>
      </v-flex>
      <v-flex lg12 >
        <v-tabs
          class="elevation-1">
          <v-tab>
            providers
          </v-tab>
          <v-tab>
            consumers
          </v-tab>
          <v-tab-item>
            <v-data-table
              class="elevation-1"
              :headers="detailHeaders.providers"
              :items="providerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{getIp(props.item.address)}}</td>
                <td>{{getPort(props.item.address)}}</td>
                <td></td>
                <td></td>
                <td><v-tooltip top>
                  <v-btn
                          small
                          slot="activator"
                          color="primary"
                  >
                    URL
                  </v-btn>
                  <span>{{props.item.url}}</span>
                </v-tooltip></td>
              </template>
            </v-data-table>
          </v-tab-item>
          <v-tab-item >
            <v-data-table
              class="elevation-1"
              :headers="detailHeaders.consumers"
              :items="consumerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{getIp(props.item.address)}}</td>
                <td>{{getPort(props.item.address)}}</td>
                <td>{{props.item.application}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
        </v-tabs>
      </v-flex>
      <v-flex sm12>
        <h3>Metadata</h3>
      </v-flex>
      <v-flex lg12>
        <v-data-table
          class="elevation-1"
          :headers="metaHeaders"
          :items="metadata">
          <template slot="items" slot-scope="props">
            <td>{{props.item.method}}</td>
            <td>{{props.item.parameter}}</td>
            <td>{{props.item.returnType}}</td>
          </template>
        </v-data-table>
      </v-flex>
    </v-layout>
  </v-container>
</template>
<script>
  import {AXIOS} from './http-common'

  export default {
    data: () => ({
      metaHeaders: [
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
        }
      ],
      detailHeaders: {
        providers: [
          {
            text: 'IP',
            value: 'ip'
          },
          {
            text: 'Port',
            value: 'port'
          },
          {
            text: 'Timeout(ms)',
            value: 'timeout'
          },
          {
            text: 'Serialization',
            value: 'serial'
          },
          {
            text: 'Operation',
            value: 'operate'
          }

        ],
        consumers: [
          {
            text: 'IP',
            value: 'ip'
          },
          {
            text: 'Port',
            value: 'port'
          },
          {
            text: 'Application Name',
            value: 'appName'
          }
        ]
      },
      providerDetails: [],
      consumerDetails: [],
      metadata: [],
      basic: []
    }),
    methods: {
      detail: function (app, service) {
        AXIOS.get('/service/detail?' + 'app=' + app + '&service=' + service)
            .then(response => {
              this.providerDetails = response.data.providers
              this.consumerDetails = response.data.consumers
            })
      },
      getIp: function (address) {
        return address.split(':')[0]
      },
      getPort: function (address) {
        return address.split(':')[1]
      }
    },
    mounted: function () {
      let query = this.$route.query
      let app = ''
      let service = ''
      Object.keys(query).forEach(function (key) {
        if (key === 'app') {
          app = query[key]
        }
        if (key === 'service') {
          service = query[key]
        }
      })
      if (app !== '' && service !== '') {
        this.detail(app, service)
        let serviceItem = {}
        serviceItem.role = 'Service Name'
        serviceItem.value = service
        this.basic.push(serviceItem)
        let appItem = {}
        appItem.role = 'Application Name'
        appItem.value = app
        this.basic.push(appItem)
      }
    }
  }
</script>

