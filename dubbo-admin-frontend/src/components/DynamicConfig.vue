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
            :items="loadBalances"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-left">{{ props.item.method }}</td>
              <td class="justify-center px-0">
                <v-tooltip bottom v-for="op in operations" :key="op.id">
                  <v-icon small class="mr-2" slot="activator" @click="itemOperation(op.icon(props.item), props.item)">
                    {{op.icon(props.item)}}
                  </v-icon>
                  <span>{{op.tooltip(props.item)}}</span>
                </v-tooltip>
              </td>
            </template>
          </v-data-table>
        </v-card-text>
      </v-card>
    </v-flex>

    <v-dialog   v-model="dialog" width="800px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">Create New LoadBalance Rule</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            label="Service Unique ID"
            hint="A service ID is service name"
            required
            v-model="service"
          ></v-text-field>
          <v-subheader class="pa-0 mt-3">RULE CONTENT</v-subheader>
          <codemirror ref="myCm"
                      v-model="ruleText"
                      :options="cmOption">
          </codemirror>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="blue darken-1" flat @click.native="closeDialog">Close</v-btn>
          <v-btn color="blue darken-1" flat @click.native="saveItem">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="warn" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{this.warnTitle}}</v-card-title>
        <v-card-text >{{this.warnText}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="green darken-1" flat @click.native="closeWarn">CANCLE</v-btn>
          <v-btn color="green darken-1" flat @click.native="deleteItem(warnStatus)">CONFIRM</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>

</template>
<script>
  import { codemirror } from 'vue-codemirror'
  import 'codemirror/lib/codemirror.css'
  import 'codemirror/theme/paraiso-light.css'
  import 'codemirror/mode/yaml/yaml.js'
  import 'codemirror/addon/display/autorefresh.js'
  import 'codemirror/addon/display/placeholder'
  import yaml from 'js-yaml'
  import {AXIOS} from './http-common'
  export default {
    components: {
      codemirror
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      ruleKeys: ['method', 'strategy'],
      pattern: 'Service',
      filter: '',
      dialog: false,
      warn: false,
      application: '',
      service: '',
      warnTitle: '',
      warnText: '',
      warnStatus: {},
      height: 0,
      operations: [
        {id: 0,
          icon: function (item) {
            return 'visibility'
          },
          tooltip: function (item) {
            return 'View'
          }},
        {id: 1,
          icon: function (item) {
            return 'edit'
          },
          tooltip: function (item) {
            return 'Edit'
          }},
        {id: 3,
          icon: function (item) {
            return 'delete'
          },
          tooltip: function (item) {
            return 'Delete'
          }}
      ],
      loadBalances: [
      ],
      template:
        'methodName: sayHello  # 0 for all methods\n' +
        'strategy:  # leastactive, random, roundrobin',
      ruleText: '',
      cmOption: {
        theme: 'paraiso-light',
        autoRefresh: true,
        readOnly: false,
        mode: 'text/x-yaml',
        line: true
      },
      headers: [
        {
          text: 'Service Name',
          value: 'service',
          align: 'left'
        },
        {
          text: 'Method',
          value: 'method',
          align: 'left'

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
        this.search(this.filter, true)
      },
      search: function (filter, rewrite) {
        let params = {}
        params.serviceName = filter
        AXIOS.post('/balancing/search', params)
          .then(response => {
            this.loadBalances = response.data
            if (rewrite) {
              this.$router.push({path: 'loadbalance', query: {serviceName: filter}})
            }
          })
      },
      handleRule: function (route) {
        let result = {}
        for (let property in route) {
          if (this.ruleKeys.includes(property)) {
            result[property] = route[property]
          }
        }
        return yaml.safeDump(result)
      },
      closeDialog: function () {
        this.ruleText = this.template
        this.service = ''
        this.dialog = false
        this.cmOption.readOnly = false
      },
      openDialog: function () {
        this.dialog = true
      },
      openWarn: function (title, text) {
        this.warnTitle = title
        this.warnText = text
        this.warn = true
      },
      closeWarn: function () {
        this.warnTitle = ''
        this.warnText = ''
        this.warn = false
      },
      saveItem: function () {
        let text = encodeURIComponent(this.ruleText)  // contains illegal url character, need encode
        let rule = {}
        rule.serviceName = this.service
        rule.rule = text
        AXIOS.post('/balancing/create', rule)
          .then(response => {
            if (response.data) {
              this.search(this.service, true)
            }
          })
      },
      itemOperation: function (icon, item) {
        switch (icon) {
          case 'visibility':
            AXIOS.get('/balancing/detail?id=' + item.id)
              .then(response => {
                let balancing = response.data
                let result = this.handleRule(balancing)
                this.service = balancing.service
                this.ruleText = result
                this.cmOption.readOnly = true
                this.dialog = true
              })
            break
          case 'edit':
            let edit = {}
            edit.id = item.id
            AXIOS.post('/balancing/edit', edit)
              .then(response => {
                let loadbalance = response.data
                let result = this.handleRule(loadbalance)
                this.service = loadbalance.service
                this.ruleText = result
                this.cmOption.readOnly = false
                this.dialog = true
              })
            break
          case 'delete':
            this.openWarn(' Are you sure to Delete Routing Rule', 'serviceName: ' + item.service)
            this.warnStatus.operation = 'delete'
            this.warnStatus.id = item.id
        }
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      },
      deleteItem: function (warnStatus) {
        if (warnStatus.operation === 'delete') {
          let id = {}
          id.id = warnStatus.id
          AXIOS.post('/balancing/delete', id)
            .then(response => {
              this.warn = false
              this.search(this.filter, false)
            })
        } else if (warnStatus.operation === 'block') {
          let status = {}
          status.enabled = warnStatus.enabled
          status.id = warnStatus.id
          AXIOS.post('/routes/changeStatus', status)
            .then(response => {
              this.warn = false
              this.search(this.filter, false)
            })
        }
      }
    },
    computed: {
      codemirror () {
        return this.$refs.myCm.codemirror
      }
    },
    created () {
      this.setHeight()
    },
    mounted: function () {
      this.ruleText = this.template
      let query = this.$route.query
      let service = ''
      Object.keys(query).forEach(function (key) {
        if (key === 'serviceName') {
          service = query[key]
        }
      })
      if (service !== '') {
        this.filter = service
        this.search(service, false)
      }
    }

  }
</script>
