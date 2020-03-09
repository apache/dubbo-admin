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
        <breadcrumb title="serviceRelation" :items="breads"></breadcrumb>
      </v-flex>
    </v-layout>

    <v-flex lg12>
      <v-card>
          <div id="chartContent" style="width:100%;height:500%;"/>
      </v-card>
    </v-flex>

  </v-container>

</template>
<script>
  import Breadcrumb from '@/components/public/Breadcrumb'
  export default {
    components: {
      Breadcrumb
    },
    data: () => ({
      success: null,
      breads: [
        {
          text: 'serviceMetrics',
          href: ''
        },
        {
          text: 'serviceRelation',
          href: ''
        }
      ],
      responseData: null
    }),
    methods: {
      initData: function () {
        // eslint-disable-next-line no-undef
        this.chartContent = echarts.init(document.getElementById('chartContent'))
        this.chartContent.showLoading()
        this.$axios.get('/metrics/relation')
          .then(response => {
            if (response && response.status === 200) {
              this.success = true
              this.responseData = response.data
              this.responseData.type = 'force'
              this.initChart(this.responseData)
            }
          })
          .catch(error => {
            this.success = false
            this.responseData = error.response.data
          })
      },
      initChart: function (data) {
        this.chartContent.hideLoading()

        const option = {
          legend: {
            top: 'bottom',
            data: data.categories.map(i => i.name)
          },
          series: [{
            type: 'graph',
            layout: 'force',
            animation: false,
            label: {
              normal: {
                show: true,
                position: 'right'
              }
            },
            draggable: true,
            data: data.nodes.map(function (node, idx) {
              node.id = idx
              return node
            }),
            categories: this.responseData.categories,
            force: {
              edgeLength: 100,
              repulsion: 10
            },
            edges: data.links,
            edgeSymbol: ['', 'arrow'],
            edgeSymbolSize: 7
          }]
        }
        this.chartContent.setOption(option)
      }
    },
    mounted: function () {
      this.initData()
    }

  }
</script>
