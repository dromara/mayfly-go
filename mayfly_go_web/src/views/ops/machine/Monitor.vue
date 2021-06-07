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
                    <ChartPie v-model:value="taskData" />
                </HomeCard>
            </el-col>
            <el-col :lg="6" :md="24">
                <HomeCard desc="Mem info" title="内存">
                    <ChartPie v-model:value="memData" />
                </HomeCard>
            </el-col>
            <el-col :lg="6" :md="24">
                <HomeCard desc="Swap info" title="CPU">
                    <ChartPie v-model:value="cpuData" />
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
                <ChartContinuou :value="this.data" title="CPU" />
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
import { ref, toRefs, reactive, watch, defineComponent, onBeforeUnmount, onMounted } from 'vue';
import ActivePlate from '@/components/chart/ActivePlate.vue';
import HomeCard from '@/components/chart/Card.vue';
import ChartPie from '@/components/chart/ChartPie.vue';
import ChartLine from '@/components/chart/ChartLine.vue';
import ChartGauge from '@/components/chart/ChartGauge.vue';
import ChartBar from '@/components/chart/ChartBar.vue';
import ChartFunnel from '@/components/chart/ChartFunnel.vue';
import ChartContinuou from '@/components/chart/ChartContinuou.vue';
import BaseChart from '@/components/chart/BaseChart.vue';
import { machineApi } from './api';
export default defineComponent({
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
    props: {
        machineId: {
            type: Number,
        },
    },
    setup(props: any) {
        let timer = 0;

        const state = reactive({
            infoCardData: [
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
            ],
            taskData: [
                { value: 0, name: '运行中', color: '#3AA1FFB' },
                { value: 0, name: '睡眠中', color: '#36CBCB' },
                { value: 0, name: '结束', color: '#4ECB73' },
                { value: 0, name: '僵尸', color: '#F47F92' },
            ],

            memData: [
                { value: 0, name: '空闲', color: '#3AA1FFB' },
                { value: 0, name: '使用中', color: '#36CBCB' },
                { value: 0, name: '缓存', color: '#4ECB73' },
            ],

            swapData: [
                { value: 0, name: '空闲', color: '#3AA1FFB' },
                { value: 0, name: '使用中', color: '#36CBCB' },
            ],

            cpuData: [
                { value: 0, name: '用户空间', color: '#3AA1FFB' },
                { value: 0, name: '内核空间', color: '#36CBCB' },
                { value: 0, name: '改变优先级', color: '#4ECB73' },
                { value: 0, name: '空闲率', color: '#4ECB73' },
                { value: 0, name: '等待IO', color: '#4ECB73' },
                { value: 0, name: '硬中断', color: '#4ECB73' },
                { value: 0, name: '软中断', color: '#4ECB73' },
                { value: 0, name: '虚拟机', color: '#4ECB73' },
            ],
        });

        watch(props, (newValue, oldValue) => {
            if (newValue.machineId) {
                intervalGetTop();
            }
        });

        onMounted(() => {
            intervalGetTop();
        });

        onBeforeUnmount(() => {
            cancelInterval();
        });

        const cancelInterval = () => {
            clearInterval(timer);
            timer = 0;
        };

        const startInterval = () => {
            if (!timer) {
                timer = setInterval(getTop, 3000) as any;
            }
        };

        const intervalGetTop = () => {
            getTop();
            startInterval();
        };

        const getTop = async () => {
            const topInfo = await machineApi.top.request({ id: props.machineId });
            state.infoCardData[0].count = topInfo.totalTask;
            state.infoCardData[1].count = Math.round(topInfo.totalMem / 1024) + 'M';
            state.infoCardData[2].count = Math.round(topInfo.availMem / 1024) + 'M';
            state.infoCardData[3].count = Math.round(topInfo.freeSwap / 1024) + 'M';
            state.infoCardData[4].count = Math.round(topInfo.usedSwap / 1024) + 'M';

            state.taskData[0].value = topInfo.runningTask;
            state.taskData[1].value = topInfo.sleepingTask;
            state.taskData[2].value = topInfo.stoppedTask;
            state.taskData[3].value = topInfo.zombieTask;

            state.memData[0].value = Math.round(topInfo.freeMem / 1024);
            state.memData[1].value = Math.round(topInfo.usedMem / 1024);
            state.memData[2].value = Math.round(topInfo.cacheMem / 1024);

            state.cpuData[0].value = topInfo.cpuUs;
            state.cpuData[1].value = topInfo.cpuSy;
            state.cpuData[2].value = topInfo.cpuNi;
            state.cpuData[3].value = topInfo.cpuId;
            state.cpuData[4].value = topInfo.cpuWa;
            state.cpuData[5].value = topInfo.cpuHi;
            state.cpuData[6].value = topInfo.cpuSi;
            state.cpuData[7].value = topInfo.cpuSt;
        };
    },
});
</script>

<style lang="scss">
.count-style {
    font-size: 50px;
}
</style>