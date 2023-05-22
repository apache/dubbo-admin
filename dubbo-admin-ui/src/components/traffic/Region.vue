<template>
    <v-container grid-list-xl fluid>
        <v-layout row wrap>
            <v-flex lg12>
        <Breadcrumb title="trafficRegion" :items="breads"></breadcrumb>
      </v-flex>
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
        text: 'trafficRegion',
        href: ''
      }
    ]
  })
}
</script>
