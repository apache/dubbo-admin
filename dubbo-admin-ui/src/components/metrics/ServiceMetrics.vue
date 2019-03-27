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
  <v-container grid-list-xl fluid>
    <v-layout row wrap>
      <v-flex lg12>
        <breadcrumb title="metrics" :items="breads"></breadcrumb>
      </v-flex>
      <v-flex xs12 >
        <search id="serviceSearch" v-model="filter" :submit="submit" :label="$t('searchSingleMetrics')"></search>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <h4>Provider (Total)</h4>
            <hr style="height:3px;border:none;border-top:3px double #9D9D9D;" />
            <mini-chart
              title="qps(ms)"
              :sub-title="majorDataMap.provider.qps"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
            <mini-chart
              title="rt(ms)"
              :sub-title="majorDataMap.provider.rt"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
              class="minichart"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
            <mini-chart
              title="success rate"
              :sub-title="majorDataMap.provider.success_rate"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <h4>Consumer (Total)</h4>
            <hr style="height:3px;border:none;border-top:3px double #9D9D9D;" />
            <mini-chart
              title="qps(ms)"
              :sub-title="majorDataMap.consumer.qps"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
            <mini-chart
              title="rt(ms)"
              :sub-title="majorDataMap.consumer.rt"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
            <mini-chart
              title="success rate"
              :sub-title="majorDataMap.consumer.success_rate"
              icon="trending_up"
              :data="dataset.monthVisit"
              :chart-color="color.blue.base"
              type="line"
            ></mini-chart>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex lg4 sm6 xs12>
        <v-card>
          <v-card-text>
            <h4>Thread Pool</h4>
            <hr style="height:3px;border:none;border-top:3px double #9D9D9D;" />
              <div class="layout row ma-0 align-center justify-space-between">
                <div class="text-box">
                  <div class="subheading pb-2">active count</div>
                  <span class="grey--text">{{this.threadPoolData.active}} <v-icon small color="green">trending_down</v-icon> </span>
                </div>
                <div class="chart">
                  <v-progress-circular
                    :size="60"
                    :width="5"
                    :rotate="360"
                    :value="this.threadPoolData.activert"
                    color="success"
                  >
                    {{this.threadPoolData.activert}}
                  </v-progress-circular>
                </div>
              </div>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
            <div class="layout row ma-0 align-center justify-space-between">
              <div class="subheading pb-2">core size</div>
              <span class="grey--text">{{this.threadPoolData.core}} </span>
              <div class="subheading pb-2">max size</div>
              <span class="grey--text">{{this.threadPoolData.max}} </span>
              <div class="subheading pb-2">current size</div>
              <span class="grey--text">{{this.threadPoolData.current}} </span>
              <div style="height:60px"></div>
              <v-icon small color="green">trending_down</v-icon>
            </div>
            <hr style="height:1px;border:none;border-top:1px solid #ADADAD;" />
          </v-card-text>
        </v-card>
      </v-flex>
      <v-flex sm12>
        <h3>{{$t('methodMetrics')}}</h3>
      </v-flex>
      <v-flex lg12 >
        <v-tabs
          class="elevation-1">
          <v-tab>
            {{$t('providers')}}
          </v-tab>
          <v-tab>
            {{$t('consumers')}}
          </v-tab>
          <v-tab-item>
            <v-data-table
              id="providerList"
              class="elevation-1"
              :headers="headers"
              :items="providerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{props.item.service}}</td>
                <td>{{props.item.method}}</td>
                <td>{{props.item.qps}}</td>
                <td>{{props.item.rt}}</td>
                <td>{{props.item.successRate}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
          <v-tab-item >
            <v-data-table
              class="elevation-1"
              :headers="headers"
              :items="consumerDetails"
            >
              <template slot="items" slot-scope="props">
                <td>{{props.item.service}}</td>
                <td>{{props.item.method}}</td>
                <td>{{props.item.qps}}</td>
                <td>{{props.item.rt}}</td>
                <td>{{props.item.successRate}}</td>
              </template>
            </v-data-table>
          </v-tab-item>
        </v-tabs>
      </v-flex>
    </v-layout>
  </v-container>
</template>

<script>
  import EChart from '@/util/echart'
  import Material from 'vuetify/es5/util/colors'
  import MiniChart from '@/components/public/MiniChart'
  import Breadcrumb from '@/components/public/Breadcrumb'
  import Search from '@/components/public/Search'
  import {
    campaignData,
    locationData,
    StackData,
    SinData
  } from '@/api/chart'

  const shortMonth = [
    1, 2, 3, 4, 5, 6
  ]

  export default {

    name: 'ServiceMetrics',
    components: {
      MiniChart,
      EChart,
      Breadcrumb,
      Search
    },
    data () {
      return {
        threadPoolData: {
          "core" : 0,
          "max" : 0,
          "current" : 0,
          "active" : 0,
          "activert": 0,
        },
        echartMap:{
          "provider": [{
            "timestamp": 0,
            "qps": 0,
            "tt": 0,
            "success_rate": 0,
          }],
          "consumer": [{
            "timestamp": 0,
            "qps": 0,
            "tt": 0,
            "success_rate": 0,
          }],
        },
        majorDataMap: {
          provider: {
            qps: "0",
            rt: "0",
            success_rate: "0%"
          },
          consumer: {
            qps: "0",
            rt: "0",
            success_rate: "0%"
          },
          threadPool:{}
        },
        selectedTab: 'tab-1',
        filter: '',
        headers: [],
        providerDetails: [
          {
            service: 'a.b.c.d',
            method: 'aaaa~ICS',
            qps: '0.58',
            rt: '111',
            success_rate: '100%'
          },
          {
            service: 'a.b.c.f',
            method: 'bbbb~ICS',
            qps: '0.87',
            rt: '120',
            success_rate: '90%'
          }

        ],
        consumerDetails: [],
        option: null,
        dataset: {
          sinData: SinData,
          monthVisit: shortMonth.map(m => {
            return {
              'time': m,
              'Value': Math.floor(Math.random() * 1000) + 200,
            }
          }),
          campaign: campaignData,
          location: locationData,
          stackData: StackData
        },
        color: Material,
        breads: [
          {
            text: 'metrics',
            href: ''
          }
        ]

      }
    },
    /*
    * */
    methods: {
      submit: function () {
        this.vv = 20
        //这里变不了我就很迷了
        this.dataset.monthVisit=[{"time": 1,"Value":200}]
        this.filter = this.filter.trim()
        this.searchByIp(this.filter)
      },
      setRandomValue: function(data) {
        for(let i in data) {
          data[i]['value'] = Math.floor(Math.random() * 1000) + 200
        }
        return data
      },
      searchByIp: function (filter) {
        //TODO 到时候记得把filter塞进来
        let url = '/metrics/ipAddr/?ip' + '=127.0.0.1' + '&group=dubbo'
        this.$axios.get(url)
          .then(response => {
            if (!response.data)
              return
            this.dealNormal(response.data)
            this.dealMajor(response.data)
            this.dealThreadPoolData(response.data)
          })
      },
      dealThreadPoolData: function (data) {
        for (let index in data) {
          let metricsDTO = data[index]
          if ((metricsDTO['metric']).indexOf('threadPool') >= -1) {
            this.threadPoolData[metricsDTO['metric'].substring(metricsDTO['metric'].lastIndexOf(".")+1)] = metricsDTO['value']
          }
        }
        this.threadPoolData.activert = (100 * this.threadPoolData.active / this.threadPoolData.current).toFixed(2)

      },
      dealMajor: function (data) {
        for (let index in data) {
          let metricsDTO = data[index]
          if (metricsDTO['metricLevel'] === 'MAJOR' && (metricsDTO['metric']).indexOf('threadPool') == -1) {
            let metric = metricsDTO['metric'] + ''
            let provider = metric.split('.')[1]
            metric = metric.substring(metric.lastIndexOf('.') + 1)
            this.dealEchartData(metricsDTO, provider, metric)
            if (typeof metricsDTO.value != 'string') {
              // console.log(metricsDTO)
              metricsDTO.value = metricsDTO.value.toFixed(2)
            }
            if (this.majorDataMap[provider][metric])
              this.majorDataMap[provider][metric] = metricsDTO.value
          }
        }
         // console.log(this.majorDataMap)
        // console.log("psw", this.echartMap)
      },
      dealEchartData: function (metricsDTO, provider, metric) {
        let timestamp = metricsDTO['timestamp']
        let arr = this.echartMap[provider]
        let lastTime = arr[arr.length-1]['timestamp']
        if (timestamp > lastTime) {
          arr.push({
            'timestamp': timestamp,
            metric: metricsDTO['value']
          })
          if(arr.length > 10) {
            arr.shift()
          }
        } else {
          arr[arr.length-1][metric] = metricsDTO['value']
        }
      },
      dealNormal: function (data) {
        let serviceMethodMap = {};
        for (let index in data) {
          let metricsDTO = data[index]
          if (metricsDTO['metricLevel'] === 'NORMAL') {
            let metric = metricsDTO['metric'] + ''
            let isProvider = metric.split('.')[1]
            metric = isProvider + '.' + metric.substring(metric.lastIndexOf('.') + 1)

            let methodMap = serviceMethodMap[metricsDTO.tags.service]
            if (!methodMap) {
              methodMap = {}
              serviceMethodMap[metricsDTO.tags.service] = methodMap
            }
            let metricMap = methodMap[metricsDTO.tags.method]

            if(!metricMap) {
              metricMap = {}
              serviceMethodMap[metricsDTO.tags.service][metricsDTO.tags.method] = metricMap
            }
            metricMap[metric] = metricsDTO['value']
          }
        }
        this.providerDetails = []
        this.consumerDetails = []
        for (let service in serviceMethodMap) {
          for (let method in serviceMethodMap[service]) {
            let metricsMap = serviceMethodMap[service][method]
            this.addDataToDetails(this.providerDetails, service, method, metricsMap, "provider")
            this.addDataToDetails(this.consumerDetails, service, method, metricsMap, "consumer")
          }
        }
      },
      addDataToDetails: function (sideDetails, service, method, metricsMap, side) {
        if(metricsMap[side + '.qps'] && metricsMap[side + '.success_rate'] && metricsMap[side + '.success_bucket_count']) {
          sideDetails.push({
            service: service,
            method: method,
            qps: metricsMap[side + '.qps'].toFixed(2),
            rt: metricsMap[side + '.rt'].toFixed(2),
            successRate: metricsMap[side + '.success_rate'],
            successCount: metricsMap[side + '.success_bucket_count']
          })
        }
      },
      setHeaders: function () {
        this.headers = [
          {
            text: this.$t('service'),
            value: 'service'
          },
          {
            text: this.$t('method'),
            value: 'method'
          },
          {
            text: this.$t('qps'),
            value: 'qps'
          },
          {
            text: this.$t('rt'),
            value: 'rt'
          },
          {
            text: this.$t('successRate'),
            value: 'successRate'
          }
        ]
      }
    },
    mounted: function () {
      this.setHeaders()
      setInterval(() => {
        this.submit()
      },5000)
    }
  }
</script>

<style scoped>

</style>
