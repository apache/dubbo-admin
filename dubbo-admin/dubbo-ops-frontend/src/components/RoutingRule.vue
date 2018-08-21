<!--
  - Licensed to the Apache Software Foundation (ASF) under one or more
  - contributor license agreements.  See the NOTICE file distributed with
  - this work for additional information regarding copyright ownership.
  - The ASF licenses this file to You under the Apache License, Version 2.0
  - (the "License"); you may not use this file except in compliance with
  -  he License.  You may obtain a copy of the License at
  -
  -      http://www.apache.org/licenses/LICENSE-2.0
  -
  -  Unless required by applicable law or agreed to in writing, software
  -  distributed under the License is distributed on an "AS IS" BASIS,
  -  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  -  See the License for the specific language governing permissions and
  -  limitations under the License.
  -->

<template>
  <v-container grid-list-xl fluid >
    <!--<v-layout row wrap>-->
    <v-layout row justify-center>
      <v-flex lg10>
        <v-text-field
          flat
          solo-inverted
          append-icon="search"
          @click:append="click"
        />
      </v-flex>
      <v-flex xs1
              class="pl-0 ml-0"
      >
        <v-select
          :items="dropdown_font"
          label="Select"
          v-model="service"
          solo-inverted
          single-line
        ></v-select>
      </v-flex>
    </v-layout>
    <v-layout justify-space-between row>
      <v-flex lg10>
        <h3>Search Result</h3>
      </v-flex>
      <v-flex xs2>
        <v-btn @click.stop="dialog = true">create Rule</v-btn>
      </v-flex>
    </v-layout>
    <v-flex lg12>
      <v-data-table
        class="elevation-1"
        :headers="headers"
        :items="result"
      >
        <template slot="items" slot-scope="props">
          <td>{{props.item.service}}</td>
          <td>{{props.item.group}}</td>
          <td>{{props.item.application}}</td>
          <td>Details</td>
        </template>
      </v-data-table>
    </v-flex>
    <v-dialog   v-model="dialog" width="450px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">Creat new Routing rule</span>
        </v-card-title>
        <v-card-text >
          <v-textarea
            name="input-7-1"
            box
            :height="height"
            label="Label"
            :placeholder="placeholder"
          ></v-textarea>
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" flat @click.native="dialog = false">Close</v-btn>
          <v-btn color="blue darken-1" flat @click.native="dialog = false">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </v-container>

</template>
<script>
  export default {
    props: {
      result: {
        type: Array,
        default: () => [
          {
            service: 'com.alibaba.dubbo.com',
            group: 'dubbo',
            application: 'demo-provider'
          },
          {
            service: 'com.alibaba.sample',
            group: 'dubbo',
            application: 'demo-provider'
          },
          {
            service: 'com.taobao.core.engine',
            group: 'dubbo',
            application: 'demo-provider'
          }

        ]
      }
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      service: 'Service',
      dialog: false,
      placeholder: 'dataId: serviceKey + CONFIGURATORS\n' +
      '\n' +
      '%yaml 1.2\n' +
      '---\n' +
      'scope: service/application\n' +
      'key: serviceKey/appName\n' +
      'configs:\n' +
      ' - addresses:[ip1, ip2]\n' +
      '   apps: [app1, app2]\n' +
      '   services: [s1, s2]\n' +
      '   side: provider\n' +
      '   rules:\n' +
      '    threadpool:\n' +
      '     size:\n' +
      '     core:\n' +
      '     queue:\n' +
      '    cluster:\n' +
      '     loadbalance:\n' +
      '     cluster:\n' +
      '    config:\n' +
      '     timeout:\n' +
      '     weight:\n' +
      '     mock: return null\n' +
      ' - addresses: [ip1, ip2]\n' +
      '   rules:\n' +
      '    threadpool:\n' +
      '     size:\n' +
      '     core:\n' +
      '     queue:\n' +
      '    cluster:\n' +
      '     loadbalance:\n' +
      '     cluster:\n' +
      '    config:\n' +
      '     timeout:\n' +
      '     weight:\n' +
      '   apps: [app1, app2]\n' +
      '   services: [s1, s2]\n' +
      '   side: provider\n' +
      '...\n',
      headers: [
        {
          text: 'Service',
          value: 'service',
          class: 'font-weight-black'
        },
        {
          text: 'Group',
          value: 'group',
          class: 'font-weight-black'
        },
        {
          text: 'Application',
          value: 'application',
          class: 'font-weight-black'
        },
        {
          text: 'Operation',
          value: 'operation',
          class: 'font-weight-black'
        }
      ]
    }),
    methods: {
      click: function () {
        console.log('aaa')
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.65
        console.log(this.height)
      }
    },
    created () {
      this.setHeight()
    }

  }
</script>
