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
        <v-flex lg12>
          <v-card flat color="transparent">
            <v-card-text>
              <v-form>
                <v-layout row wrap>
                  <v-combobox
                    id="serviceSearch"
                    v-model="filter"
                    flat
                    append-icon=""
                    hide-no-data
                    :suffix="queryBy"
                    label="Search Routing Rule"
                  ></v-combobox>
                  <v-menu class="hidden-xs-only">
                    <v-btn slot="activator" large icon>
                      <v-icon>unfold_more</v-icon>
                    </v-btn>

                    <v-list>
                      <v-list-tile
                        v-for="(item, i) in items"
                        :key="i"
                        @click="selected = i">
                        <v-list-tile-title>{{ item.title }}</v-list-tile-title>
                      </v-list-tile>
                    </v-list>
                  </v-menu>
                  <v-btn @click="submit" color="primary" large>Search</v-btn>

                </v-layout>
              </v-form>
            </v-card-text>
          </v-card>
        </v-flex>
      </v-layout>
    <v-flex lg12>
      <v-card>
        <v-toolbar flat color="transparent" class="elevation-0">
          <v-toolbar-title><span class="headline">Search Result</span></v-toolbar-title>
          <v-spacer></v-spacer>
          <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">CREATE</v-btn>
        </v-toolbar>

        <v-card-text class="pa-0" v-if="selected == 0">
          <v-data-table
            :headers="serviceHeaders"
            :items="serviceRoutingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.service }}</td>
              <td class="text-xs-left">{{ props.item.group }}</td>
              <td class="text-xs-left">{{ props.item.priority }}</td>
              <td class="text-xs-left">{{ props.item.enabled }}</td>
              <td class="text-xs-center px-0">
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
        <v-card-text class="pa-0" v-if="selected == 1">
          <v-data-table
            :headers="appHeaders"
            :items="appRoutingRules"
            hide-actions
            class="elevation-0"
          >
            <template slot="items" slot-scope="props">
              <td class="text-xs-left">{{ props.item.application }}</td>
              <td class="text-xs-left">{{ props.item.priority }}</td>
              <td class="text-xs-left">{{ props.item.enabled }}</td>
              <td class="text-xs-center px-0">
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
          <span class="headline">Create New Routing Rule</span>
        </v-card-title>
        <v-card-text >
          <v-text-field
            label="Service Unique ID"
            hint="A service ID in form of group/service:version, group and version are optional"
            v-model="service"
          ></v-text-field>
          <v-text-field
            label="Application Name"
            hint="Application name the service belongs to"
            v-model="application"
          ></v-text-field>

          <v-subheader class="pa-0 mt-3">RULE CONTENT</v-subheader>
          <ace-editor v-model="ruleText" :readonly="readonly"></ace-editor>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeDialog">Close</v-btn>
          <v-btn depressed color="primary" @click.native="saveItem">Save</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

    <v-dialog v-model="warn" persistent max-width="500px">
      <v-card>
        <v-card-title class="headline">{{this.warnTitle}}</v-card-title>
        <v-card-text >{{this.warnText}}</v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeWarn">CANCLE</v-btn>
          <v-btn depressed color="primary" @click.native="deleteItem(warnStatus)">CONFIRM</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

  </v-container>

</template>
<script>
  import yaml from 'js-yaml'
  import AceEditor from '@/components/public/AceEditor'
  import operations from '@/api/operation'
  export default {
    components: {
      AceEditor
    },
    data: () => ({
      items: [
        {id: 0, title: 'service name', value: 'serviceName'},
        {id: 1, title: 'application', value: 'application'}
      ],
      selected: 0,
      dropdown_font: [ 'Service', 'App', 'IP' ],
      ruleKeys: ['enabled', 'force', 'dynamic', 'runtime', 'group', 'version', 'rule', 'priority'],
      pattern: 'Service',
      filter: '',
      dialog: false,
      warn: false,
      updateId: '',
      application: '',
      service: '',
      warnTitle: '',
      warnText: '',
      warnStatus: {},
      height: 0,
      operations: operations,
      serviceRoutingRules: [
      ],
      appRoutingRules: [
      ],
      required: value => !!value || 'Service ID is required, in form of group/service:version, group and version are optional',
      template:
        'enabled: true\n' +
        'priority: 100\n' +
        'runtime: false\n' +
        'force: true\n' +
        'conditions:\n' +
        ' - \'=> host != 172.22.3.91\'\n',
      ruleText: '',
      readonly: false,
      appHeaders: [
        {
          text: 'Application Name',
          value: 'application',
          align: 'left'
        },
        {
          text: 'Priority',
          value: 'priority',
          sortable: false
        },
        {
          text: 'Enabled',
          value: 'enabled',
          sortable: false
        },
        {
          text: 'Operation',
          value: 'operation',
          sortable: false,
          width: '115px'
        }
      ],
      serviceHeaders: [
        {
          text: 'Service Name',
          value: 'service',
          align: 'left'
        },
        {
          text: 'Group',
          value: 'group',
          align: 'left'

        },
        {
          text: 'Priority',
          value: 'priority',
          sortable: false
        },
        {
          text: 'Enabled',
          value: 'enabled',
          sortable: false
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
        let type = this.items[this.selected].value
        let url = '/rules/route/condition/?' + type + '=' + filter
        this.$axios.get(url)
          .then(response => {
            if (this.selected === 0) {
              this.serviceRoutingRules = response.data
            } else {
              this.appRoutingRules = response.data
            }
            if (rewrite) {
              if (this.selected === 0) {
                this.$router.push({path: 'routingRule', query: {serviceName: filter}})
              } else if (this.selected === 1) {
                this.$router.push({path: 'routingRule', query: {application: filter}})
              }
            }
          })
      },
      closeDialog: function () {
        this.ruleText = this.template
        this.updateId = ''
        this.service = ''
        this.application = ''
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
        let rule = yaml.safeLoad(this.ruleText)
        if (this.service === '' && this.application === '') {
          return
        }
        rule.service = this.service
        rule.application = this.application
        if (this.updateId !== '') {
          if (this.updateId === 'close') {
            this.closeDialog()
          } else {
            rule.id = this.updateId
            this.$axios.put('/rules/route/condition/' + rule.id, rule)
              .then(response => {
                if (response.status === 200) {
                  if (this.service !== '') {
                    this.selected = 0
                    this.search(this.service, true)
                    this.filter = this.service
                  } else {
                    this.selected = 1
                    this.search(this.application, true)
                    this.filter = this.application
                  }
                  this.closeDialog()
                  this.$notify.success('Update success')
                }
              })
          }
        } else {
          this.$axios.post('/rules/route/condition/', rule)
            .then(response => {
              if (response.status === 201) {
                if (this.service !== '') {
                  this.selected = 0
                  this.search(this.service, true)
                  this.filter = this.service
                } else {
                  this.selected = 1
                  this.search(this.application, true)
                  this.filter = this.application
                }
                this.closeDialog()
                this.$notify.success('Create success')
              }
            })
            .catch(error => {
              console.log(error)
            })
        }
      },
      itemOperation: function (icon, item) {
        let itemId = ''
        if (this.selected === 0) {
          itemId = item.service
        } else {
          itemId = item.application
        }
        switch (icon) {
          case 'visibility':
            this.$axios.get('/rules/route/condition/' + itemId)
              .then(response => {
                let conditionRoute = response.data
                this.handleBalance(conditionRoute, true)
                this.updateId = 'close'
              })
            break
          case 'edit':
            let id = {}
            id.id = itemId
            this.$axios.get('/rules/route/condition/' + itemId)
              .then(response => {
                let conditionRoute = response.data
                this.handleBalance(conditionRoute, false)
                this.updateId = itemId
              })
            break
          case 'block':
            this.openWarn(' Are you sure to block Routing Rule', 'service: ' + item.service)
            this.warnStatus.operation = 'disable'
            this.warnStatus.id = itemId
            break
          case 'check_circle_outline':
            this.openWarn(' Are you sure to enable Routing Rule', 'service: ' + item.service)
            this.warnStatus.operation = 'enable'
            this.warnStatus.id = itemId
            break
          case 'delete':
            this.openWarn(' Are you sure to Delete Routing Rule', 'service: ' + item.service)
            this.warnStatus.operation = 'delete'
            this.warnStatus.id = itemId
        }
      },
      handleBalance: function (conditionRoute, readonly) {
        this.service = conditionRoute.service
        this.application = conditionRoute.application
        delete conditionRoute.service
        delete conditionRoute.id
        delete conditionRoute.app
        delete conditionRoute.group
        delete conditionRoute.application
        this.ruleText = yaml.safeDump(conditionRoute)
        this.readonly = readonly
        this.dialog = true
      },
      setHeight: function () {
        this.height = window.innerHeight * 0.5
      },
      deleteItem: function (warnStatus) {
        let id = warnStatus.id
        let operation = warnStatus.operation
        if (operation === 'delete') {
          this.$axios.delete('/rules/route/condition/' + id)
            .then(response => {
              if (response.status === 200) {
                this.warn = false
                this.search(this.filter, false)
                this.$notify.success('Delete success')
              }
            })
        } else if (operation === 'disable') {
          this.$axios.put('/rules/route/condition/disable/' + id)
            .then(response => {
              if (response.status === 200) {
                this.warn = false
                this.search(this.filter, false)
                this.$notify.success('Disable success')
              }
            })
        } else if (operation === 'enable') {
          this.$axios.put('/rules/route/condition/enable/' + id)
            .then(response => {
              if (response.status === 200) {
                this.warn = false
                this.search(this.filter, false)
                this.$notify.success('Enable success')
              }
            })
        }
      }
    },
    created () {
      this.setHeight()
    },
    computed: {
      queryBy () {
        return 'by ' + this.items[this.selected].title
      }
    },
    mounted: function () {
      this.ruleText = this.template
      let query = this.$route.query
      let filter = null
      let vm = this
      Object.keys(query).forEach(function (key) {
        if (key === 'service') {
          filter = query[key]
          vm.selected = 0
        }
        if (key === 'application') {
          filter = query[key]
          vm.selected = 1
        }
      })
      if (filter !== null) {
        this.filter = filter
        this.search(filter, false)
      }
    }

  }
</script>
