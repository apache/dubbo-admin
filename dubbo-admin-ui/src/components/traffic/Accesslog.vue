<!--
  ~ Licensed to the Apache Software Foundation (ASF) under one or more
  ~ contributor license agreements.  See the NOTICE file distributed with
  ~ this work for additional information regarding copyright ownership.
  ~ The ASF licenses this file to You under the Apache License, Version 2.0
  ~ (the "License"); you may not use this file except in compliance with
  ~ the License.  You may obtain a copy of the License at
  ~
  ~     http://www.apache.org/licenses/LICENSE-2.0
  ~
  ~ Unless required by applicable law or agreed to in writing, software
  ~ distributed under the License is distributed on an "AS IS" BASIS,
  ~ WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  ~ See the License for the specific language governing permissions and
  ~ limitations under the License.
-->
<template>
    <v-container grid-list-xl fluid>
        <v-layout row wrap>
            <v-flex lg12>
        <Breadcrumb title="trafficAccesslog" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex lg12>
          <v-card flat color="transparent">
            <v-card-text>
              <v-form>
                <v-layout row wrap>
                  <v-combobox
                    :loading="searchLoading"
                    :items="typeAhead"
                    :search-input.sync="application"
                    v-model="application"
                    flat
                    append-icon=""
                    hide-no-data
                    hint="请输入application"
                  ></v-combobox>
                  <v-combobox
                    :loading="searchLoading"
                    :items="typeAhead"
                    :search-input.sync="accesslog"
                    v-model="accesslog"
                    flat
                    append-icon=""
                    hide-no-data
                    hint="请输入accesslog"
                  ></v-combobox>
                  <v-btn @click="submit" color="primary" large>{{ $t('search') }}</v-btn>
                  <v-btn @click="submit" color="primary" large>新建</v-btn>
                </v-layout>
              </v-form>
            </v-card-text>
          </v-card>
        </v-flex>
      <v-flex xs12>
        <v-card>
          <v-toolbar flat color="transparent" class="elevation-0">
            <v-toolbar-title><span class="headline">{{$t('trafficAccesslog')}}</span></v-toolbar-title>
            <v-spacer></v-spacer>
          </v-toolbar>
          <v-card-text class="pa-0">
            <v-data-table :headers="headers" :items="tableData" hide-actions class="elevation-1">
              <template slot="items" slot-scope="props">
                <td>{{ props.item.name }}</td>
                <td>
                  <v-chip xs v-for="(type, index) in props.item.parameterTypes" :key="index" label>{{ type }}</v-chip>
                </td>
                <td>
                  <v-chip label>{{ props.item.returnType }}</v-chip>
                </td>
                <td class="text-xs-right">
                  <v-tooltip bottom>
                    <v-btn
                      fab dark small color="blue" slot="activator"
                      :href="getHref(props.item.application, props.item.service, props.item.signature)"
                    >
                      <v-icon>edit</v-icon>
                    </v-btn>
                    <span>{{$t('test')}}</span>
                  </v-tooltip>
                </td>
              </template>
            </v-data-table>
          </v-card-text>
        </v-card>
      </v-flex>
      <v-dialog v-model="dialog" width="800px" persistent >
      <v-card>
        <v-card-title class="justify-center">
          <span class="headline">{{$t('createNewRoutingRule')}}</span>
        </v-card-title>
        <v-card-text >
          <v-layout wrap>
            <v-flex xs24 sm12 md8>
              <v-text-field
                label="Service class"
                :hint="$t('dataIdClassHint')"
                v-model="service"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md2>
              <v-text-field
                label="Version"
                :hint="$t('dataIdVersionHint')"
                v-model="serviceVersion"
              ></v-text-field>
            </v-flex>
            <v-flex xs6 sm3 md2>
              <v-text-field
                label="Group"
                :hint="$t('dataIdGroupHint')"
                v-model="serviceGroup"
              ></v-text-field>
            </v-flex>
          </v-layout>
          <v-text-field
            label="Application Name"
            hint="Application name the service belongs to"
            v-model="application"
          ></v-text-field>

          <v-subheader class="pa-0 mt-3">{{$t('ruleContent')}}</v-subheader>
          <ace-editor v-model="ruleText" :readonly="readonly"></ace-editor>

        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn flat @click.native="closeDialog">{{$t('close')}}</v-btn>
          <v-btn depressed color="primary" @click.native="saveItem">{{$t('save')}}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>

      </v-layout>
    </v-container>
</template>
<script>
import Breadcrumb from '../public/Breadcrumb.vue'
export default {
  name: 'Accesslog',
  components: { Breadcrumb },
  data: () => ({
    breads: [
      {
        text: 'trafficManagement',
        href: ''
      },
      {
        text: 'trafficAccesslog',
        href: ''
      }
    ],
    typeAhead: [],
    input: null,
    searchLoading: false,
    timerID: null,
    application: '',
    accesslog: '',
    dialog: false,
    headers: [
      {
        text: '服务',
        value: 'application',
        sortable: false
      },
      {
        text: 'accesslog',
        value: 'accesslog',
        sortable: false
      }
    ],
    service: null,
    tableData: [],
    services: [],
    loading: false
  }),
  methods: {
    submit () {
      if (this.accesslog && this.application) {
        this.search()
      } else {
        this.$notify.error('service is needed')
        return false
      }
    },
    search () {
      this.$axios.post('/traffic/accesslog', {
        params: {
          application: this.application,
          accesslog: this.accesslog
        }
      }).then(response => {
        this.service = response.data
        this.service.array.forEach(element => {
          this.tableData.push(element)
        })
      }).catch(error => {
        this.showSnackbar('error', error.response.data.message)
      })
    }
  }
}

</script>
