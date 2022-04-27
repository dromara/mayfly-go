<template>
    <div class="home-container">
        <el-row :gutter="15">
            <el-col :sm="6" class="mb15">
                <div @click="toPage({ id: 'personal' })" class="home-card-item home-card-first">
                    <div class="flex-margin flex">
                        <img :src="getUserInfos.photo" />
                        <div class="home-card-first-right ml15">
                            <div class="flex-margin">
                                <div class="home-card-first-right-title">{{ `${currentTime}, ${getUserInfos.username}` }}</div>
                            </div>
                        </div>
                    </div>
                </div>
            </el-col>
            <el-col :sm="3" class="mb15" v-for="(v, k) in topCardItemList" :key="k">
                <div @click="toPage(v)" class="home-card-item home-card-item-box" :style="{ background: v.color }">
                    <div class="home-card-item-flex">
                        <div class="home-card-item-title pb3">{{ v.title }}</div>
                        <div class="home-card-item-title-num pb6" :id="v.id"></div>
                    </div>
                    <i :class="v.icon" :style="{ color: v.iconColor }"></i>
                </div>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, onMounted, nextTick, computed } from 'vue';
import { useStore } from '@/store/index.ts';
// import * as echarts from 'echarts';
import { CountUp } from 'countup.js';
import { formatAxis } from '@/common/utils/formatTime.ts';
import { indexApi } from './api';
import { useRouter } from 'vue-router';
export default {
    name: 'HomePage',
    setup() {
        // const { proxy } = getCurrentInstance() as any;
        const router = useRouter();
        const store = useStore();
        const state = reactive({
            topCardItemList: [
                {
                    title: '项目数',
                    id: 'projectNum',
                    color: '#FEBB50',
                },
                {
                    title: 'Linux机器数',
                    id: 'machineNum',
                    color: '#F95959',
                },
                {
                    title: '数据库总数',
                    id: 'dbNum',
                    color: '#8595F4',
                },
                {
                    title: 'redis总数',
                    id: 'redisNum',
                    color: '#1abc9c',
                },
            ],
        });

        // 当前时间提示语
        const currentTime = computed(() => {
            return formatAxis(new Date());
        });

        // 初始化数字滚动
        const initNumCountUp = async () => {
            const res: any = await indexApi.getIndexCount.request();
            nextTick(() => {
                new CountUp('projectNum', res.projectNum).start();
                new CountUp('machineNum', res.machineNum).start();
                new CountUp('dbNum', res.dbNum).start();
                new CountUp('redisNum', res.redisNum).start();
            });
        };

        const toPage = (item: any) => {
            switch (item.id) {
                case 'personal': {
                    router.push('/personal');
                    break;
                }
                case 'projectNum': {
                    router.push('/ops/projects');
                    break;
                }
                case 'machineNum': {
                    router.push('/ops/machines');
                    break;
                }
                case 'dbNum': {
                    router.push('/ops/dbms/dbs');
                    break;
                }
                case 'redisNum': {
                    router.push('/ops/redis/manage');
                    break;
                }
            }
        };

        // 页面加载时
        onMounted(() => {
            initNumCountUp();
            // initHomeLaboratory();
            // initHomeOvertime();
        });

        // 获取用户信息 vuex
        const getUserInfos = computed(() => {
            return store.state.userInfos.userInfos;
        });

        return {
            getUserInfos,
            currentTime,
            toPage,
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
        cursor: pointer;
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
