import Vue from 'vue'
import App from './App.vue'
import router from './router'
import store from './store'
import ElementUI from 'element-ui'
import 'element-ui/lib/theme-chalk/index.css'
import './assets/css/style.css'

// import ECharts from 'vue-echarts' // 在 webpack 环境下指向 components/ECharts.vue

// 手动引入 ECharts 各模块来减小打包体积
// import 'echarts/lib/chart/bar'
// import 'echarts/lib/component/tooltip'

Vue.config.productionTip = false
// 注册组件后即可使用
// Vue.component('v-chart', ECharts)
Vue.use(ElementUI)

// 全局error处理
Vue.config.errorHandler = function(err, vm ,info) {
  // 如果是断言错误，则进行提示即可
  if (err.name == 'AssertError') {
    ElementUI.Message.error(err.message)
  } else {
    console.error(err, info)
  }
}

new Vue({
  router,
  store,
  render: h => h(App)
}).$mount('#app')
