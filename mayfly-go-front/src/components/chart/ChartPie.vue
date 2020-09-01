<template>
  <div class="pie-main" id="box" ref="dom"></div>
</template>

<script>
import echarts from 'echarts'
import tdTheme from './theme.json'
import { on, off } from './onoff'
echarts.registerTheme('tdTheme', tdTheme)
export default {
  props: {
    value: Array,
    text: String,
    subtext: String,
  },
  watch: {
    value: {
      handler: function (val, oldval) {
        this.value = val
        this.initChart()
      },
      deep: true, //对象内部的属性监听，也叫深度监听
    },
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
        const legend = this.value.map((_) => _.name)
        const option = {
          title: {
            text: this.text,
            subtext: this.subtext,
            x: 'center',
          },
          position: {
            top: 40,
          },
          tooltip: {
            trigger: 'item',
            formatter: '{c} ({d}%)',
            // position: ['30%', '90%'],
            position: function (point, params, dom, rect, size) {
              console.log(size)
              const leftWidth = size.viewSize[0] / 2 - size.contentSize[0] / 2
              console.log(leftWidth)
              return { left: leftWidth, bottom: 0 }
            },
            backgroundColor: 'transparent',
            textStyle: {
              fontSize: 24,
              color: '#666',
            },
          },
          legend: {
            // orient: 'vertical',
            top: 0,
            data: legend,
            backgroundColor: 'transparent',
            icon: 'circle',
          },
          series: [
            {
              name: '访问来源',
              type: 'pie',
              radius: ['45%', '60%'],
              center: ['50%', '52%'],
              avoidLabelOverlap: false,
              label: {
                normal: {
                  show: false,
                  position: 'center',
                },
                emphasis: {
                  show: true,
                  textStyle: {
                    fontSize: '24',
                  },
                },
              },
              labelLine: {
                normal: {
                  show: false,
                },
              },
              data: this.value,
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
.pie-main {
  width: 100%;
  height: 360px;
  padding: 28px;
  background: #fff;
}
</style>