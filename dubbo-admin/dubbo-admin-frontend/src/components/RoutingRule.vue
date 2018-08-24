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
    <div>
      <v-layout row wrap>
        <v-flex xs12 class="justify-space-between">
          <v-form>
            <v-layout row wrap>
              <v-flex xs11>
                <v-text-field label="Search dubbo service"
                              v-model="filter"></v-text-field>
              </v-flex>

              <v-flex xs1>
                <v-btn @click="submit" color="primary" >Search</v-btn>
              </v-flex>
            </v-layout>
          </v-form>
        </v-flex>
      </v-layout>
      <v-toolbar flat color="white">
        <v-toolbar-title>Search Result</v-toolbar-title>
        <v-spacer></v-spacer>
        <v-btn outline color="primary" @click.stop="dialog = true" class="mb-2">CREATE</v-btn>
      </v-toolbar>
      <v-data-table
        :headers="headers"
        :items="routingRules"
        hide-actions
        class="elevation-1"
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
    </div>
    <v-dialog   v-model="dialog" width="450px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">Create new Routing rule</span>
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
    data: () => ({
      dropdown_font: [ 'Service', 'App', 'IP' ],
      pattern: 'Service',
      filter: '',
      dialog: false,
      selected: [],
      routingRules: [
        {
          id: 0,
          rule: 'test',
          service: 'com.alibaba.dubbo.com',
          priority: 0,
          status: 'enabled'
        }
      ],
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
          text: 'Rule Name',
          value: 'rule',
          class: 'font-weight-black'
        },
        {
          text: 'Service Name',
          value: 'service',
          class: 'font-weight-black'
        },
        {
          text: 'Priority',
          value: 'priority',
          class: 'font-weight-black'
        },
        {
          text: 'Status',
          value: 'status',
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
      submit () {
        console.log('submit')
      },
      toggleAll () {
        if (this.selected.length) this.selected = []
        else this.selected = this.routingRules.slice()
      },
      enable: function (status) {
        if (status === 'enabled') {
          return 'disable'
        }
        return 'enable'
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

<style scoped>
  div.btn__content {
    padding: 0;
  }
</style>
