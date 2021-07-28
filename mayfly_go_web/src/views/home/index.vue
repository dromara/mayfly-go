<template>
    <div class="home-container">
        <el-row :gutter="15">
            <el-col :sm="6" class="mb15">
                <div class="home-card-item home-card-first">
                    <div class="flex-margin flex">
                        <img :src="getUserInfos.photo" />
                        <div class="home-card-first-right ml15">
                            <div class="flex-margin">
                                <div class="home-card-first-right-title">{{ `${currentTime}, ${getUserInfos.username}` }}</div>
                                <!-- <div class="home-card-first-right-msg mt5">超级管理</div> -->
                            </div>
                        </div>
                    </div>
                </div>
            </el-col>
            <el-col :sm="6" class="mb15" v-for="(v, k) in topCardItemList" :key="k">
                <div class="home-card-item home-card-item-box" :style="{ background: v.color }">
                    <div class="home-card-item-flex">
                        <div class="home-card-item-title pb3">{{ v.title }}</div>
                        <div class="home-card-item-title-num pb6" :id="`titleNum${k + 1}`"></div>
                        <div class="home-card-item-tip pb3">{{ v.tip }}</div>
                        <div class="home-card-item-tip-num" :id="`tipNum${k + 1}`"></div>
                    </div>
                    <i :class="v.icon" :style="{ color: v.iconColor }"></i>
                </div>
            </el-col>
        </el-row>
        <el-row :gutter="15">
            <el-col :xs="24" :sm="14" :md="14" :lg="16" :xl="16" class="mb15">
                <el-card shadow="hover" header="商品销售情况">
                    <div style="height: 200px" ref="homeLaboratoryRef"></div>
                </el-card>
            </el-col>
            <el-col :xs="24" :sm="10" :md="10" :lg="8" :xl="8">
                <el-card shadow="hover" header="环境监测">
                    <div class="home-monitor">
                        <div class="flex-warp">
                            <div class="flex-warp-item" v-for="(v, k) in environmentList" :key="k">
                                <div class="flex-warp-item-box">
                                    <i :class="v.icon" :style="{ color: v.iconColor }"></i>
                                    <span class="pl5">{{ v.label }}</span>
                                    <div class="mt10">{{ v.value }}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </el-card>
            </el-col>
        </el-row>
        <el-row :gutter="15">
            <el-col :xs="24" :sm="14" :md="14" :lg="16" :xl="16" class="home-warning-media">
                <el-card shadow="hover" header="布局配置" class="home-warning-card">
                    <el-table :data="tableData.data" style="width: 100%" stripe>
                        <el-table-column prop="date" label="时间"></el-table-column>
                        <el-table-column prop="name" label="实验室名称"></el-table-column>
                        <el-table-column prop="address" label="报警内容"></el-table-column>
                    </el-table>
                </el-card>
            </el-col>
            <el-col :xs="24" :sm="10" :md="10" :lg="8" :xl="8" class="home-dynamic-media">
                <el-card shadow="hover" header="动态信息">
                    <div class="home-dynamic">
                        <el-scrollbar>
                            <div class="home-dynamic-item" v-for="(v, k) in activitiesList" :key="k">
                                <div class="home-dynamic-item-left">
                                    <div class="home-dynamic-item-left-time1 mb5">{{ v.time1 }}</div>
                                    <div class="home-dynamic-item-left-time2">{{ v.time2 }}</div>
                                </div>
                                <div class="home-dynamic-item-line">
                                    <i class="iconfont icon-fangkuang"></i>
                                </div>
                                <div class="home-dynamic-item-right">
                                    <div class="home-dynamic-item-right-title mb5">
                                        <i class="el-icon-s-comment"></i>
                                        <span>{{ v.title }}</span>
                                    </div>
                                    <div class="home-dynamic-item-right-label">{{ v.label }}</div>
                                </div>
                            </div>
                        </el-scrollbar>
                    </div>
                </el-card>
            </el-col>
        </el-row>
        <el-row>
            <el-col :xs="24" :sm="24" :md="24" :lg="24" :xl="24" class="mt15">
                <el-card shadow="hover" header="履约超时预警">
                    <div style="height: 200px" ref="homeOvertimeRef"></div>
                </el-card>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, nextTick, computed, getCurrentInstance } from 'vue';
import { useStore } from '@/store/index.ts';
import * as echarts from 'echarts';
import { CountUp } from 'countup.js';
import { formatAxis } from '@/common/utils/formatTime.ts';
import { topCardItemList, environmentList, activitiesList } from './mock.ts';
export default {
    name: 'Home',
    setup() {
        const { proxy } = getCurrentInstance() as any;
        const store = useStore();
        const state = reactive({
            topCardItemList,
            environmentList,
            activitiesList,
            tableData: {
                data: [
                    {
                        date: '2016-05-02',
                        name: '1号实验室',
                        address: '烟感2.1%OBS/M',
                    },
                    {
                        date: '2016-05-04',
                        name: '2号实验室',
                        address: '温度30℃',
                    },
                    {
                        date: '2016-05-01',
                        name: '3号实验室',
                        address: '湿度57%RH',
                    },
                ],
            },
        });
        // 当前时间提示语
        const currentTime = computed(() => {
            return formatAxis(new Date());
        });
        // 初始化数字滚动
        const initNumCountUp = () => {
            nextTick(() => {
                new CountUp('titleNum1', Math.random() * 10000).start();
                new CountUp('titleNum2', Math.random() * 10000).start();
                new CountUp('titleNum3', Math.random() * 10000).start();
                new CountUp('tipNum1', Math.random() * 1000).start();
                new CountUp('tipNum2', Math.random() * 1000).start();
                new CountUp('tipNum3', Math.random() * 1000).start();
            });
        };
        // 实验室使用情况
        const initHomeLaboratory = () => {
            const myChart = echarts.init(proxy.$refs.homeLaboratoryRef);
            const option = {
                grid: {
                    top: 50,
                    right: 20,
                    bottom: 30,
                    left: 30,
                },
                tooltip: {
                    trigger: 'axis',
                },
                legend: {
                    data: ['预购队列', '最新成交价'],
                    right: 13,
                },
                xAxis: {
                    data: ['衬衫', '羊毛衫', '雪纺衫', '裤子', '高跟鞋', '袜子'],
                },
                yAxis: [
                    {
                        type: 'value',
                        name: '价格',
                    },
                ],
                series: [
                    {
                        name: '预购队列',
                        type: 'bar',
                        data: [5, 20, 36, 10, 10, 20],
                    },
                    {
                        name: '最新成交价',
                        type: 'line',
                        data: [15, 20, 16, 20, 30, 8],
                    },
                ],
            };
            myChart.setOption(option);
            window.addEventListener('resize', () => {
                myChart.resize();
            });
        };
        // 履约超时预警
        const initHomeOvertime = () => {
            const myChart = echarts.init(proxy.$refs.homeOvertimeRef);
            const option = {
                grid: {
                    top: 50,
                    right: 20,
                    bottom: 30,
                    left: 30,
                },
                tooltip: {
                    trigger: 'axis',
                },
                legend: {
                    data: ['订单数量', '超时数量', '在线数量', '预警数量'],
                    right: 13,
                },
                xAxis: {
                    data: ['1月', '2月', '3月', '4月', '5月', '6月', '7月', '8月', '9月', '10月', '11月', '12月'],
                },
                yAxis: [
                    {
                        type: 'value',
                        name: '数量',
                    },
                ],
                series: [
                    {
                        name: '订单数量',
                        type: 'bar',
                        data: [5, 20, 36, 10, 10, 20, 11, 13, 10, 9, 17, 19],
                    },
                    {
                        name: '超时数量',
                        type: 'bar',
                        data: [15, 12, 26, 15, 11, 16, 31, 13, 5, 16, 13, 15],
                    },
                    {
                        name: '在线数量',
                        type: 'line',
                        data: [15, 20, 16, 20, 30, 8, 16, 19, 12, 18, 19, 14],
                    },
                    {
                        name: '预警数量',
                        type: 'line',
                        data: [10, 10, 13, 12, 15, 18, 19, 10, 12, 15, 11, 17],
                    },
                ],
            };
            myChart.setOption(option);
            window.addEventListener('resize', () => {
                myChart.resize();
            });
        };
        // 页面加载时
        onMounted(() => {
            initNumCountUp();
            initHomeLaboratory();
            initHomeOvertime();
        });

        // 获取用户信息 vuex
        const getUserInfos = computed(() => {
            return store.state.userInfos.userInfos;
        });
        return {
            getUserInfos,
            currentTime,
            ...toRefs(state),
        };
    },
};
</script>

<style scoped lang="scss">
.home-container {
    overflow-x: hidden;
    .home-card-item {
        width: 100%;
        height: 103px;
        background: gray;
        border-radius: 4px;
        transition: all ease 0.3s;
        &:hover {
            box-shadow: 0 2px 12px 0 rgb(0 0 0 / 10%);
            transition: all ease 0.3s;
        }
    }
    .home-card-item-box {
        display: flex;
        align-items: center;
        position: relative;
        overflow: hidden;
        &:hover {
            i {
                right: 0px !important;
                bottom: 0px !important;
                transition: all ease 0.3s;
            }
        }
        i {
            position: absolute;
            right: -10px;
            bottom: -10px;
            font-size: 70px;
            transform: rotate(-30deg);
            transition: all ease 0.3s;
        }
        .home-card-item-flex {
            padding: 0 20px;
            color: white;
            .home-card-item-title,
            .home-card-item-tip {
                font-size: 13px;
            }
            .home-card-item-title-num {
                font-size: 18px;
            }
            .home-card-item-tip-num {
                font-size: 13px;
            }
        }
    }
    .home-card-first {
        background: white;
        border: 1px solid #ebeef5;
        display: flex;
        align-items: center;
        img {
            width: 60px;
            height: 60px;
            border-radius: 100%;
            border: 2px solid var(--color-primary-light-5);
        }
        .home-card-first-right {
            flex: 1;
            display: flex;
            flex-direction: column;
            .home-card-first-right-msg {
                font-size: 13px;
                color: gray;
            }
        }
    }
    .home-monitor {
        height: 200px;
        .flex-warp-item {
            width: 50%;
            height: 100px;
            display: flex;
            .flex-warp-item-box {
                margin: auto;
                height: auto;
                text-align: center;
            }
        }
    }
    .home-warning-card {
        height: 292px;
        ::v-deep(.el-card) {
            height: 100%;
        }
    }
    .home-dynamic {
        height: 200px;
        .home-dynamic-item {
            display: flex;
            width: 100%;
            height: 60px;
            overflow: hidden;
            &:first-of-type {
                .home-dynamic-item-line {
                    i {
                        color: orange !important;
                    }
                }
            }
            .home-dynamic-item-left {
                text-align: right;
                .home-dynamic-item-left-time1 {
                }
                .home-dynamic-item-left-time2 {
                    font-size: 13px;
                    color: gray;
                }
            }
            .home-dynamic-item-line {
                height: 60px;
                border-right: 2px dashed #dfdfdf;
                margin: 0 20px;
                position: relative;
                i {
                    color: var(--color-primary);
                    font-size: 12px;
                    position: absolute;
                    top: 1px;
                    left: -6px;
                    transform: rotate(46deg);
                    background: white;
                }
            }
            .home-dynamic-item-right {
                flex: 1;
                .home-dynamic-item-right-title {
                    i {
                        margin-right: 5px;
                        border: 1px solid #dfdfdf;
                        width: 20px;
                        height: 20px;
                        border-radius: 100%;
                        padding: 3px 2px 2px;
                        text-align: center;
                        color: var(--color-primary);
                    }
                }
                .home-dynamic-item-right-label {
                    font-size: 13px;
                    color: gray;
                }
            }
        }
    }
}
</style>
