<template>
  <div class="line-main" id="box" ref="dom"></div>
</template>

<script>
import echarts from 'echarts'
import tdTheme from './theme.json'
import { on, off } from './onoff'
echarts.registerTheme('tdTheme', tdTheme)
export default {
  props: {
    value: Array,
    title: String,
    subtext: String,
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
        const dateList = this.value.map(function (item) {
          return item[0]
        })
        const valueList = this.value.map(function (item) {
          return item[1]
        })

        const option = {
          // Make gradient line here
          visualMap: [
            {
              show: false,
              type: 'continuous',
              seriesIndex: 0,
              min: 0,
              max: 400,
            }
          ],

          title: [
            {
              left: 'center',
              text: this.title,
            }
          ],
          tooltip: {
            trigger: 'axis',
          },
          xAxis: [
            {
              data: dateList,
            }
          ],
          yAxis: [
            {
              splitLine: { show: false },
            },
          ],
          grid: [
            {
              
            },
          ],
          series: [
            {
              type: 'line',
              showSymbol: false,
              data: valueList,
            },
          ],
        }
        this.dom = echarts.init(this.$refs.dom, 'tdTheme')
        this.dom.setOption(option)
        on(window, 'resize', this.resize)
      })
    },
  },
}
</script>

<style>
.line-main {
  width: 100%;
  height: 360px;
  padding: 28px;
  background: #fff;
}
</style>