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
    <v-layout row wrap >
      <v-flex sm12>
        <h3>basic info</h3>
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
        <h3>service info</h3>
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
                <td>{{props.item.ip}}</td>
                <td>{{props.item.port}}</td>
                <td>{{props.item.timeout}}</td>
                <td>{{props.item.serial}}</td>
                <td>URL</td>
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
                <td>{{props.item.ip}}</td>
                <td>{{props.item.port}}</td>
                <td>{{props.item.appName}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
        </v-tabs>
      </v-flex>
      <v-flex sm12>
        <h3>meta data</h3>
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
  export default {
    props: {
      basic: {
        type: Array,
        default: () =>
          [
            {
              role: '应用名',
              value: 'dubbo-demo'
            },
            {
              role: '服务名',
              value: 'dubbo.com.alibaba'
            },
            {
              role: 'xxxx',
              value: 'ffffff'
            }
          ]
      },
      providerDetails: {
        type: Array,
        default: () =>
          [
            {
              ip: '192.168.0.1',
              port: '28880',
              timeout: '3000',
              serial: 'hessian'
            },
            {
              ip: '192.168.0.8',
              port: '28880',
              timeout: '3000',
              serial: 'hessian'

            }
          ]
      },
      consumerDetails: {
        type: Array,
        default: () =>
          [
            {
              ip: '192.168.1.3',
              port: '56895',
              appName: 'dubbo-demo'
            },
            {
              ip: '192.168.2.8',
              port: '35971',
              appName: 'dubbo-demo'
            }

          ]
      },

      metadata: {
        type: Array,
        default: () =>
          [
            {
              method: 'toString',
              parameter: 'java.lang.String',
              returnType: 'void'
            },
            {
              method: 'queryBatch',
              parameter: 'com.taobao.tc.domain.query.QueryBizOrderDO',
              returnType: 'com.taobao.tc.domain.result.BatchQueryBizOrderResultDO'
            },
            {
              method: 'isShowCheckcode',
              parameter: 'long',
              returnType: 'com.taobao.tc.domain.result.QueryTairResultDO'
            }

          ]
      }
    },
    data: () => ({
      metaHeaders: [
        {
          text: '方法名',
          value: 'method',
          sortable: false
        },
        {
          text: '参数列表',
          value: 'parameter',
          sortable: false
        },
        {
          text: '返回值类型',
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
            text: '端口',
            value: 'port'
          },
          {
            text: '超时时间(ms)',
            value: 'timeout'
          },
          {
            text: '序列化方式',
            value: 'serial'
          },
          {
            text: '操作',
            value: 'operate'
          }

        ],
        consumers: [
          {
            text: 'IP',
            value: 'ip'
          },
          {
            text: '端口',
            value: 'port'
          },
          {
            text: '应用名',
            value: 'appName'
          }
        ]
      }
    })
  }
</script>

