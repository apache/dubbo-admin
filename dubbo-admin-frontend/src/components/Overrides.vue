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
            :items="configs"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="justify-center px-0">
                <v-tooltip bottom v-for="op in operations" :key="op.id">
                  <v-icon small class="mr-2" slot="activator" @click="itemOperation(op.icon, props.item)">
                    {{op.icon}}
                  </v-icon>
                  <span>{{op.tooltip}}</span>
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
          <span class="headline">Create New Dynamic Config Rule</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            label="Service Unique ID"
            hint="A service ID is service name"
            required
            v-model="service"
          ></v-text-field>
          <v-subheader class="pa-0 mt-3">RULE CONTENT</v-subheader>
          <ace-editor v-model="ruleText" :readonly="readonly"/>

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
  import AceEditor from '@/components/AceEditor'
  import yaml from 'js-yaml'
  import {AXIOS} from './http-common'
  export default {
    components: {
      AceEditor
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
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
        {id: 0, icon: 'visibility', tooltip: 'View'},
        {id: 1, icon: 'edit', tooltip: 'Edit'},
        {id: 3, icon: 'delete', tooltip: 'Delete'}
      ],
      configs: [
      ],
      template:
        'application:  # consumer\'s application name, empty for all \n' +
        'address: 192.168.0.1 # consumer\'s ip address, empty for all consumers\n' +
        'dynamic: false\n' +
        'enabled: false # enable this rule\n' +
        'parameters:\n' +
        '  - timeout: 100\n' +
        '\n' +
        'mock: \n' +
        '  - 0: \'force: return null\'\n' +
        '  - sayHello: \'force: return null\'\n' +
        '  - test: \'fail: return empty\'',
      ruleText: '',
      readonly: false,
      headers: [
        {
          text: 'Service Name',
          value: 'service',
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
        AXIOS.get('/override/search?serviceName=' + filter)
          .then(response => {
            this.configs = response.data
            if (rewrite) {
              this.$router.push({path: 'config', query: {service: filter}})
            }
          })
      },
      closeDialog: function () {
        this.ruleText = this.template
        this.service = ''
        this.dialog = false
        this.readonly = false
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
        let override = yaml.safeLoad(this.ruleText)  // contains illegal url character, need encode
        override.service = this.service
        AXIOS.post('/override/create', override)
          .then(response => {
            if (response.data) {
              this.search(this.service, true)
              this.filter = this.service
            }
            this.closeDialog()
          })
      },
      itemOperation: function (icon, item) {
        switch (icon) {
          case 'visibility':
            AXIOS.get('/override/detail?id=' + item.id)
              .then(response => {
                let config = response.data
                this.service = config.service
                delete config.service
                this.ruleText = yaml.safeDump(config)
                this.readonly = true
                this.dialog = true
              })
            break
          case 'edit':
            AXIOS.get('/override/detail?id=' + item.id)
              .then(response => {
                let config = response.data
                this.service = config.service
                delete config.service
                this.ruleText = yaml.safeDump(config)
                this.readonly = false
                this.dialog = true
              })
            break
          case 'delete':
            this.openWarn(' Are you sure to Delete Dynamic Config', 'service: ' + item.service)
            this.warnStatus.operation = 'delete'
            this.warnStatus.id = item.id
        }
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      },
      deleteItem: function (warnStatus) {
        let id = {}
        id.id = warnStatus.id
        AXIOS.post('/override/delete', id)
          .then(response => {
            this.warn = false
            this.search(this.filter, false)
          })
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
        if (key === 'service') {
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
