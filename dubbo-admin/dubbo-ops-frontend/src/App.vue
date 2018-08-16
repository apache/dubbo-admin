<template>
  <v-app>
    <drawer></drawer>
    <toolbar></toolbar>
    <v-content>
      <router-view/>
    </v-content>
    <footers></footers>
  </v-app>
</template>
<script>
import Drawer from '@/components/Drawer'
import Toolbar from '@/components/Toolbar'
import Footers from '@/components/Footers'
export default {
  components: {
    Drawer,
    Toolbar,
    Footers
  },
  data () {
    return {
      clipped: false,
      drawer: true,
      fixed: false,
      items: [{
        icon: 'bubble_chart',
        title: 'Inspire'
      }],
      miniVariant: false,
      right: true,
      rightDrawer: false,
      title: 'Vuetify.js'
    }
  },
  created () {
    window.getApp = this
  },
  computed: {
    appTitle () {
      return this.$store.state.appTitle
    }
  },
  name: 'App'
}
</script>
