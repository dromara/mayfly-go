<template>
  <div>
    <el-row>
      <el-col>
        <HomeCard desc="Base info" title="基础信息">
          <ActivePlate :infoList="infoCardData" />
        </HomeCard>
      </el-col>
    </el-row>
    <el-row :gutter="20">
      <el-col :lg="6" :md="24">
        <HomeCard desc="Task info" title="任务">
          <ChartPie :value.sync="taskData" />
        </HomeCard>
      </el-col>
      <el-col :lg="6" :md="24">
        <HomeCard desc="Mem info" title="内存">
          <ChartPie :value.sync="memData" />
        </HomeCard>
      </el-col>
      <el-col :lg="6" :md="24">
        <HomeCard desc="Swap info" title="CPU">
          <ChartPie :value.sync="cpuData" />
        </HomeCard>
      </el-col>
    </el-row>

    <!-- <el-row :gutter="20">
      <el-col :lg="18" :md="24">
        <HomeCard desc="User active" title="每周用户活跃量">
          <ChartLine :value="lineData" />
        </HomeCard>
      </el-col>
    </el-row>-->

    <el-row :gutter="20">
      <el-col :lg="12" :md="24">
        <ChartContinuou :value="this.data" title="内存" />
      </el-col>
      <el-col :lg="12" :md="24">
        <ChartContinuou  :value="this.data" title="CPU" />
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :lg="12" :md="24">
        <HomeCard desc="load info" title="负载情况">
          <BaseChart :option="this.loadChartOption" />
        </HomeCard>
      </el-col>
      <el-col :lg="12" :md="24">
        <ChartContinuou :value="this.data" title="磁盘IO" />
      </el-col>
    </el-row>
  </div>
</template>

<script lang="ts">
import { Component, Vue, Prop, Watch } from 'vue-property-decorator'
import ActivePlate from '@/components/chart/ActivePlate.vue'
import HomeCard from '@/components/chart/Card.vue'
import ChartPie from '@/components/chart/ChartPie.vue'
import ChartLine from '@/components/chart/ChartLine.vue'
import ChartGauge from '@/components/chart/ChartGauge.vue'
import ChartBar from '@/components/chart/ChartBar.vue'
import ChartFunnel from '@/components/chart/ChartFunnel.vue'
import ChartContinuou from '@/components/chart/ChartContinuou.vue'
import BaseChart from '@/components/chart/BaseChart.vue'
import { machineApi } from './api'

@Component({
  name: 'Monitor',
  components: {
    HomeCard,
    ActivePlate,
    ChartPie,
    ChartFunnel,
    ChartLine,
    ChartGauge,
    ChartBar,
    ChartContinuou,
    BaseChart,
  },
})
export default class Monitor extends Vue {
  @Prop()
  machineId: number

  timer: number

  infoCardData = [
    {
      title: 'total task',
      icon: 'md-person-add',
      count: 0,
      color: '#11A0F8',
    },
    { title: '总内存', icon: 'md-locate', count: '', color: '#FFBB44 ' },
    {
      title: '可用内存',
      icon: 'md-help-circle',
      count: '',
      color: '#7ACE4C',
    },
    { title: '空闲交换空间', icon: 'md-share', count: 657, color: '#11A0F8' },
    {
      title: '使用中交换空间',
      icon: 'md-chatbubbles',
      count: 12,
      color: '#91AFC8',
    },
    { title: '新增页面', icon: 'md-map', count: 14, color: '#91AFC8' },
  ]
  taskData = [
    { value: 0, name: '运行中', color: '#3AA1FFB' },
    { value: 0, name: '睡眠中', color: '#36CBCB' },
    { value: 0, name: '结束', color: '#4ECB73' },
    { value: 0, name: '僵尸', color: '#F47F92' },
  ]

  memData = [
    { value: 0, name: '空闲', color: '#3AA1FFB' },
    { value: 0, name: '使用中', color: '#36CBCB' },
    { value: 0, name: '缓存', color: '#4ECB73' },
  ]

  swapData = [
    { value: 0, name: '空闲', color: '#3AA1FFB' },
    { value: 0, name: '使用中', color: '#36CBCB' },
  ]

  cpuData = [
    { value: 0, name: '用户空间', color: '#3AA1FFB' },
    { value: 0, name: '内核空间', color: '#36CBCB' },
    { value: 0, name: '改变优先级', color: '#4ECB73' },
    { value: 0, name: '空闲率', color: '#4ECB73' },
    { value: 0, name: '等待IO', color: '#4ECB73' },
    { value: 0, name: '硬中断', color: '#4ECB73' },
    { value: 0, name: '软中断', color: '#4ECB73' },
    { value: 0, name: '虚拟机', color: '#4ECB73' },
  ]
  data = [
    ['06/05 15:01', 116.12],
    ['06/05 15:06', 129.21],
    ['06/05 15:11', 135.43],
    ['2000-06-08', 86.33],
    ['2000-06-09', 73.98],
    ['2000-06-10', 85],
    ['2000-06-11', 73],
    ['2000-06-12', 68],
    ['2000-06-13', 92],
    ['2000-06-14', 130],
    ['2000-06-15', 245],
    ['2000-06-16', 139],
    ['2000-06-17', 115],
    ['2000-06-18', 111],
    ['2000-06-19', 309],
    ['2000-06-20', 206],
    ['2000-06-21', 137],
    ['2000-06-22', 128],
    ['2000-06-23', 85],
    ['2000-06-24', 94],
    ['2000-06-25', 71],
    ['2000-06-26', 106],
    ['2000-06-27', 84],
    ['2000-06-28', 93],
    ['2000-06-29', 85],
    ['2000-06-30', 73],
    ['2000-07-01', 83],
    ['2000-07-02', 125],
    ['2000-07-03', 107],
    ['2000-07-04', 82],
    ['2000-07-05', 44],
    ['2000-07-06', 72],
    ['2000-07-07', 106],
    ['2000-07-08', 107],
    ['2000-07-09', 66],
    ['2000-07-10', 91],
    ['2000-07-11', 92],
    ['2000-07-12', 113],
    ['2000-07-13', 107],
    ['2000-07-14', 131],
    ['2000-07-15', 111],
    ['2000-07-16', 64],
    ['2000-07-17', 69],
    ['2000-07-18', 88],
    ['2000-07-19', 77],
    ['2000-07-20', 83],
    ['2000-07-21', 111],
    ['2000-07-22', 57],
    ['2000-07-23', 55],
    ['2000-07-24', 60],
  ]

  dateList = this.data.map(function (item) {
    return item[0]
  })
  valueList = this.data.map(function (item) {
    return item[1]
  })
  loadChartOption = {
    // Make gradient line here
    visualMap: [
      {
        show: false,
        type: 'continuous',
        seriesIndex: 0,
        min: 0,
        max: 400,
      },
    ],
    legend: {
      data: ['1分钟', '5分钟', '15分钟'],
    },
    tooltip: {
      trigger: 'axis',
    },
    xAxis: [
      {
        data: this.dateList,
      },
    ],
    yAxis: [
      {
        splitLine: { show: false },
      },
    ],
    grid: [{}],
    series: [
      {
        name: '1分钟',
        type: 'line',
        showSymbol: false,
        data: this.valueList,
      },
      {
        name: '5分钟',
        type: 'line',
        showSymbol: false,
        data: [100, 22, 33, 121, 32, 332, 322, 222, 232],
      },
      {
        name: '15分钟',
        type: 'line',
        showSymbol: true,
        data: [130, 222, 373, 135, 456, 332, 333, 343, 342],
      },
    ],
  }

  lineData = {
    Mon: 13253,
    Tue: 34235,
    Wed: 26321,
    Thu: 12340,
    Fri: 24643,
    Sat: 1322,
    Sun: 1324,
  }

  @Watch('machineId', { deep: true })
  onDataChange() {
    if (this.machineId) {
      this.intervalGetTop()
    }
  }

  mounted() {
    this.intervalGetTop()
  }

  beforeDestroy() {
    this.cancelInterval()
  }

  cancelInterval() {
    clearInterval(this.timer)
    this.timer = 0
  }

  startInterval() {
    if (!this.timer) {
      this.timer = setInterval(this.getTop, 3000)
    }
  }

  intervalGetTop() {
    this.getTop()
    this.startInterval()
  }

  async getTop() {
    const topInfo = await machineApi.top.request({ id: this.machineId })
    this.infoCardData[0].count = topInfo.totalTask
    this.infoCardData[1].count = Math.round(topInfo.totalMem / 1024) + 'M'
    this.infoCardData[2].count = Math.round(topInfo.availMem / 1024) + 'M'
    this.infoCardData[3].count = Math.round(topInfo.freeSwap / 1024) + 'M'
    this.infoCardData[4].count = Math.round(topInfo.usedSwap / 1024) + 'M'

    this.taskData[0].value = topInfo.runningTask
    this.taskData[1].value = topInfo.sleepingTask
    this.taskData[2].value = topInfo.stoppedTask
    this.taskData[3].value = topInfo.zombieTask

    this.memData[0].value = Math.round(topInfo.freeMem / 1024)
    this.memData[1].value = Math.round(topInfo.usedMem / 1024)
    this.memData[2].value = Math.round(topInfo.cacheMem / 1024)

    this.cpuData[0].value = topInfo.cpuUs
    this.cpuData[1].value = topInfo.cpuSy
    this.cpuData[2].value = topInfo.cpuNi
    this.cpuData[3].value = topInfo.cpuId
    this.cpuData[4].value = topInfo.cpuWa
    this.cpuData[5].value = topInfo.cpuHi
    this.cpuData[6].value = topInfo.cpuSi
    this.cpuData[7].value = topInfo.cpuSt
  }
}
</script>

<style lang="less">
.count-style {
  font-size: 50px;
}
</style>