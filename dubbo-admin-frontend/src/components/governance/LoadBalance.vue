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
        <search v-model="filter" :submit="submit" label="Search Load Balance by service Name"></search>

      </v-flex>
    </v-layout>

    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">Search Result</span></v-toolbar-title>
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
              <td class="text-xs-left">{{ props.item.methodName }}</td>
              <td class="text-xs-center px-0">
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
          <span class="headline">Create New LoadBalance Rule</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            label="Service Unique ID"
            hint="A service ID in form of group/service:version, group and version are optional"
            :rules="[required]"
            v-model="service"
          ></v-text-field>
          <v-text-field
            label="Application Name"
            hint="Application name the service belongs to"
            v-model="application"
          ></v-text-field>
          <v-subheader class="pa-0 mt-3">RULE CONTENT</v-subheader>
          <ace-editor v-model="ruleText" :readonly="readonly"/>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1" flat @click.native="closeDialog">Close</v-btn>
          <v-btn color="primary darken-1" depressed @click.native="saveItem">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="warn" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{this.warnTitle}}</v-card-title>
        <v-card-text >{{this.warnText}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn color="darken-1" flat @click.native="closeWarn">CANCLE</v-btn>
          <v-btn color="primary darken-1" depressed @click.native="deleteItem(warnStatus.id)">CONFIRM</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>

</template>
<script>
  import yaml from 'js-yaml'
  import AceEditor from '@/components/public/AceEditor'
  import Search from '@/components/public/Search'
  export default {
    components: {
      AceEditor,
      Search
    },
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      ruleKeys: ['method', 'strategy'],
      pattern: 'Service',
      filter: '',
      updateId: '',
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
      loadBalances: [
      ],
      required: value => !!value || 'Service ID is required, in form of group/service:version, group and version are optional',
      template:
        'methodName: sayHello  # 0 for all methods\n' +
        'strategy:  # leastactive, random, roundrobin',
      ruleText: '',
      readonly: false,
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
          sortable: false,
          width: '115px'
        }
      ]
    }),
    methods: {
      submit: function () {
        this.search(this.filter, true)
      },
      search: function (filter, rewrite) {
        this.$axios.get('/rules/balancing', {
          params: {
            service: filter
          }
        }).then(response => {
          this.loadBalances = response.data
          if (rewrite) {
            this.$router.push({path: 'loadbalance', query: {service: filter}})
          }
        })
      },
      closeDialog: function () {
        this.ruleText = this.template
        this.service = ''
        this.dialog = false
        this.updateId = ''
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
        let balancing = yaml.safeLoad(this.ruleText)
        if (this.service === '') {
          return
        }
        balancing.service = this.service
        if (this.updateId !== '') {
          if (this.updateId === 'close') {
            this.closeDialog()
          } else {
            balancing.id = this.updateId
            this.$axios.put('/rules/balancing/' + balancing.id, balancing)
              .then(response => {
                if (response.status === 200) {
                  this.search(this.service, true)
                  this.filter = this.service
                  this.closeDialog()
                  this.$notify.success('Update success')
                }
              })
          }
        } else {
          this.$axios.post('/rules/balancing', balancing)
            .then(response => {
              if (response.status === 201) {
                this.search(this.service, true)
                this.filter = this.service
                this.closeDialog()
                this.$notify.success('Create success')
              }
            })
        }
      },
      itemOperation: function (icon, item) {
        switch (icon) {
          case 'visibility':
            this.$axios.get('/rules/balancing/' + item.id)
              .then(response => {
                let balancing = response.data
                this.handleBalance(balancing, true)
                this.updateId = 'close'
              })
            break
          case 'edit':
            this.$axios.get('/rules/balancing/' + item.id)
              .then(response => {
                let balancing = response.data
                this.handleBalance(balancing, false)
                this.updateId = item.id
              })
            break
          case 'delete':
            this.openWarn(' Are you sure to Delete Routing Rule', 'service: ' + item.service)
            this.warnStatus.operation = 'delete'
            this.warnStatus.id = item.id
        }
      },
      handleBalance: function (balancing, readonly) {
        this.service = balancing.service
        delete balancing.service
        delete balancing.id
        this.ruleText = yaml.safeDump(balancing)
        this.readonly = readonly
        this.dialog = true
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      },
      deleteItem: function (id) {
        this.$axios.delete('/rules/balancing/' + id)
          .then(response => {
            if (response.status === 200) {
              this.warn = false
              this.search(this.filter, false)
              this.$notify.success('Delete success')
            }
          })
      }
    },
    created () {
      this.setHeight()
    },
    mounted: function () {
      this.ruleText = this.template
      let query = this.$conditionRoute.query
      let service = null
      Object.keys(query).forEach(function (key) {
        if (key === 'service') {
          service = query[key]
        }
      })
      if (service !== null) {
        this.filter = service
        this.search(service, false)
      }
    }

  }
</script>
