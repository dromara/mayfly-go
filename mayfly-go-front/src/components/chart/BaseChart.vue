<template>
  <div class="base-chart" id="box" ref="dom"></div>
</template>

<script>
import echarts from 'echarts'
import tdTheme from './theme.json'
import { on, off } from './onoff'
echarts.registerTheme('tdTheme', tdTheme)
export default {
  props: {
    option: Object,
  },
  mounted() {
    this.initChart()
  },
  methods: {
    resize() {
      this.dom.resize()
    },
    initChart() {
      this.$nextTick(() => {
        this.dom = echarts.init(this.$refs.dom, 'tdTheme')
        this.dom.setOption(this.option)
        on(window, 'resize', this.resize)
      })
    },
  },
}
</script>

<style>
.base-chart {
  width: 100%;
  height: 360px;
  padding: 28px;
  background: #fff;
}
</style>