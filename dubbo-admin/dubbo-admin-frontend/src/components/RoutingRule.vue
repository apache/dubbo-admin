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
        <v-flex xs12 >
          <v-card flat>
            <v-card-text>
              <v-layout row wrap >
                <v-text-field label="Search dubbo service"
                              v-model="filter"></v-text-field>
                <v-btn @click="submit" color="primary" >Search</v-btn>
              </v-layout>

            </v-card-text>

          </v-card>
        </v-flex>
      </v-layout>
      <v-toolbar class="elevation-1" flat color="white">
        <v-toolbar-title>Search Result</v-toolbar-title>
        <v-spacer></v-spacer>
        <v-btn outline color="primary" @click.stop="openDialog" class="mb-2">CREATE</v-btn>
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
          <v-text-field
            placeholder="service:version or application, version is optional"
            required
            ref="scope"
            :rules="[() => !!scope || 'This field is required']"
            v-model="scope"
          ></v-text-field>
          <v-text-field
            placeholder="group, only effective on service"
            v-model="group"
          ></v-text-field>
          <!--<v-textarea-->
          <!--id="rule-content"-->
          <!--name="input-7-1"-->
          <!--box-->
          <!--:height="height"-->
          <!--:placeholder="placeholder"-->
          <!--&gt;</v-textarea>-->
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
  import $ from 'jquery'
  // import CodeMirror from 'codemirror'
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
      group: '',
      scope: '',
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
      submit: function () {
        console.log('submit')
      },
      openDialog: function () {
        this.dialog = true
        // $('.CodeMirror').remove()
        // this.initCodeMirror(document.getElementById('rule-content'))
        // this.refreshCodeMirror()
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
      // initCodeMirror: function (element) {
      //   return CodeMirror.fromTextArea(element, {
      //     lineNumbers: true,
      //     lineWrapping: true,
      //     mode: 'text/x-yaml'
      //   })
      // },
      // refreshCodeMirror: function () {
      //   setTimeout(function () {
      //     $('.CodeMirror').each(function (i, el) {
      //       el.CodeMirror.refresh()
      //     })
      //   }, 100)
      // }
    },

    created () {
      this.setHeight()
    }

  }
</script>
