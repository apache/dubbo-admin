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
      <v-layout row wrap>
        <v-flex xs12 >
          <v-card flat color="transparent">
            <v-card-text>
              <v-layout row wrap >
                <v-text-field label="Search dubbo service"
                              v-model="filter" clearable></v-text-field>
                <v-btn @click="submit" color="primary" large>Search</v-btn>
              </v-layout>
            </v-card-text>
          </v-card>
        </v-flex>
      </v-layout>

    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">Search Result</span></v-toolbar-title>
          <v-divider
            class="mx-2"
            inset
            vertical
          ></v-divider>
          <v-spacer></v-spacer>
          <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">CREATE</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0">
          <v-data-table
            :headers="headers"
            :items="routingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td>{{ props.item.rule }}</td>
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-left">{{ props.item.priority }}</td>
              <td class="text-xs-left">{{ props.item.status }}</td>
              <td class="justify-center px-0">
                <v-icon
                  small
                  class="mr-2"
                  @click="deleteItem(props.item)"
                >
                  visibility
                </v-icon>
                <v-icon
                  small
                  class="mr-2"
                  @click="editItem(props.item)"
                >
                  edit
                </v-icon>
                <v-icon
                  small
                  class="mr-2"
                  @click="editItem(props.item)"
                >
                  block
                </v-icon>
                <v-icon
                  small
                  @click="deleteItem(props.item)"
                >
                  delete
                </v-icon>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog   v-model="dialog" width="450px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">Create new Routing rule</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            placeholder="service:version, version is optional"
            required
            v-model="service"
          ></v-text-field>
          <v-text-field
            placeholder="application name"
            required
            v-model="application"
          ></v-text-field>
          <codemirror :placeholder='placeholder' :options="cmOption"></codemirror>
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
  import { codemirror } from 'vue-codemirror'
  import 'codemirror/lib/codemirror.css'
  import 'codemirror/mode/yaml/yaml.js'
  import 'codemirror/addon/display/placeholder'
  export default {
    components: {
      codemirror
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      pattern: 'Service',
      filter: '',
      dialog: false,
      application: '',
      service: '',
      height: 0,
      routingRules: [
        {
          id: 0,
          rule: 'test',
          service: 'com.alibaba.dubbo.com',
          priority: 0,
          status: 'enabled'
        }
      ],
      placeholder: '%yaml 1.2\n' +
        '---\n' +
        'enable: true/false\n' +
        'priority:\n' +
        'runtime: false/true\n' +
        'category: routers\n' +
        'force: true/false\n' +
        'dynamic: true/false\n' +
        'conditions:\n' +
        '  - => host != 172.22.3.91\n' +
        '  - host != 10.20.153.10,10.20.153.11 =>\n' +
        '  - host = 10.20.153.10,10.20.153.11 =>\n' +
        '  - application != kylin => host != 172.22.3.95,172.22.3.96\n' +
        '  - method = find*,list*,get*,is* => host = 172.22.3.94,172.22.3.95,172.22.3.96\n' +
        '...\n',
      cmOption: {
        lineNumbers: true,
        mode: 'text/x-yaml'
      },
      headers: [
        {
          text: 'Rule Name',
          value: 'rule'
        },
        {
          text: 'Service Name',
          value: 'service'
        },
        {
          text: 'Priority',
          value: 'priority',
          sortable: false
        },
        {
          text: 'Status',
          value: 'status',
          sortable: false
        },
        {
          text: 'Operation',
          value: 'operation',
          sortable: false
        }
      ]
    }),
    methods: {
      submit: function () {
        console.log('submit')
      },
      openDialog: function () {
        this.dialog = true
      },
      enable: function (status) {
        if (status === 'enabled') {
          return 'disable'
        }
        return 'enable'
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      }
    },

    created () {
      this.setHeight()
    }

  }
</script>
