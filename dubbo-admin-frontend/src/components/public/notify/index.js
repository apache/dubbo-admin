import Snackbar from './Snackbar.vue'

const Notify = {}

Notify.install = function (Vue) {
  const SnackbarConstructor = Vue.extend(Snackbar)
  const instance = new SnackbarConstructor()
  let vm = instance.$mount()
  document.querySelector('body').appendChild(vm.$el)

  Vue.prototype.$notify = (text, color) => {
    instance.text = text
    instance.color = color
    instance.show = true
  }
  Vue.prototype.$notify.error = text => {
    instance.text = text
    instance.color = 'error'
    instance.show = true
  }
  Vue.prototype.$notify.success = text => {
    instance.text = text
    instance.color = 'success'
    instance.show = true
  }
  Vue.prototype.$notify.info = text => {
    instance.text = text
    instance.color = 'info'
    instance.show = true
  }
}

export default Notify
