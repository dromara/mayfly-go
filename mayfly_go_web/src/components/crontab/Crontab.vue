<template>
    <div class="crontab">
        <el-tabs v-model="state.activeName" @tab-change="changeTab(state.activeName)" type="border-card">
            <el-tab-pane label="秒" name="second" v-if="shouldHide('second')">
                <CrontabSecond :cron="crontabValueObj" ref="secondRef" />
            </el-tab-pane>

            <el-tab-pane label="分钟" name="min" v-if="shouldHide('min')">
                <CrontabMin :cron="crontabValueObj" ref="minRef" />
            </el-tab-pane>

            <el-tab-pane label="小时" name="hour" v-if="shouldHide('hour')">
                <CrontabHour :cron="crontabValueObj" ref="hourRef" />
            </el-tab-pane>

            <el-tab-pane label="日" name="day" v-if="shouldHide('day')">
                <CrontabDay :cron="crontabValueObj" ref="dayRef" />
            </el-tab-pane>

            <el-tab-pane label="月" name="mouth" v-if="shouldHide('mouth')">
                <CrontabMouth :cron="crontabValueObj" ref="mouthRef" />
            </el-tab-pane>

            <el-tab-pane label="周" name="week" v-if="shouldHide('week')">
                <CrontabWeek :cron="crontabValueObj" ref="weekRef" />
            </el-tab-pane>

            <el-tab-pane label="年" name="year" v-if="shouldHide('year')">
                <CrontabYear :cron="crontabValueObj" ref="yearRef" />
            </el-tab-pane>
        </el-tabs>

        <div class="popup-main">
            <div class="popup-result">
                <p class="title">时间表达式</p>
                <table>
                    <thead>
                        <th v-for="item of tabTitles" width="40" :key="item">{{ item }}</th>
                        <th>crontab完整表达式</th>
                    </thead>
                    <tbody>
                        <td>
                            <span>{{ crontabValueObj.second }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.min }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.hour }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.day }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.mouth }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.week }}</span>
                        </td>
                        <td>
                            <span>{{ crontabValueObj.year }}</span>
                        </td>
                        <td>
                            <span>{{ contabValueString }}</span>
                        </td>
                    </tbody>
                </table>
            </div>
            <CrontabResult :ex="contabValueString"></CrontabResult>

            <div class="pop_btn">
                <el-button size="small" @click="hidePopup">取消</el-button>
                <el-button size="small" type="warning" @click="clearCron">重置</el-button>
                <el-button size="small" type="primary" @click="submitFill">确定</el-button>
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { computed, toRefs, onMounted, reactive, ref, nextTick, watch } from 'vue';
import CrontabSecond from './CrontabSecond.vue';
import CrontabMin from './CrontabMin.vue';
import CrontabHour from './CrontabHour.vue';
import CrontabDay from './CrontabDay.vue';
import CrontabMouth from './CrontabMouth.vue';
import CrontabWeek from './CrontabWeek.vue';
import CrontabYear from './CrontabYear.vue';
import CrontabResult from './CrontabResult.vue';

const secondRef: any = ref(null);
const minRef: any = ref(null);
const hourRef: any = ref(null);
const dayRef: any = ref(null);
const mouthRef: any = ref(null);
const weekRef: any = ref(null);
const yearRef: any = ref(null);

const props = defineProps({
    expression: {
        type: String,
        required: true,
    },
    hideComponent: {
        type: Array,
    },
});

//定义事件
const emit = defineEmits(['hide', 'fill']);

const state = reactive({
    tabTitles: ['秒', '分钟', '小时', '日', '月', '周', '年'],
    tabActive: 0,
    activeName: 'second',
    myindex: 0,
    crontabValueObj: {
        second: '*',
        min: '*',
        hour: '*',
        day: '*',
        mouth: '*',
        week: '?',
        year: '',
    },
});

const { tabTitles, crontabValueObj } = toRefs(state);

onMounted(() => {
    resolveExp();
});

watch(
    () => props.expression,
    () => {
        resolveExp();
    }
);

function shouldHide(key: string) {
    if (props.hideComponent && props.hideComponent.includes(key)) return false;
    return true;
}

function resolveExp() {
    //反解析 表达式
    if (props.expression) {
        let arr = props.expression.split(' ');
        if (arr.length >= 6) {
            //6 位以上是合法表达式
            let obj = {
                second: arr[0],
                min: arr[1],
                hour: arr[2],
                day: arr[3],
                mouth: arr[4],
                week: arr[5],
                year: arr[6] ? arr[6] : '',
            };
            state.crontabValueObj = {
                ...obj,
            };
        }
        changeTab(state.activeName);
    } else {
        //没有传入的表达式 则还原
        clearCron();
    }
}

// 改变tab
const changeTab = (name: string) => {
    nextTick(() => {
        getRefByName(name).value?.parse();
    });
};

const getRefByName = (name: string) => {
    switch (name) {
        case 'second':
            return secondRef;
        case 'min':
            return minRef;
        case 'hour':
            return hourRef;
        case 'day':
            return dayRef;
        case 'mouth':
            return mouthRef;
        case 'week':
            return weekRef;
        case 'year':
            return yearRef;
    }
};

// 隐藏弹窗
function hidePopup() {
    emit('hide');
}

// 填充表达式
const submitFill = () => {
    emit('fill', contabValueString.value);
    hidePopup();
};

const clearCron = () => {
    // 还原选择项
    state.crontabValueObj = {
        second: '*',
        min: '*',
        hour: '*',
        day: '*',
        mouth: '*',
        week: '?',
        year: '',
    };
    changeTab(state.activeName);
};

const contabValueString = computed(() => {
    let obj = state.crontabValueObj;
    let str = obj.second + ' ' + obj.min + ' ' + obj.hour + ' ' + obj.day + ' ' + obj.mouth + ' ' + obj.week + (obj.year == '' ? '' : ' ' + obj.year);
    return str;
});
</script>

<style scoped lang="scss">
.pop_btn {
    text-align: right;
}
.popup-main {
    position: relative;
    margin: 10px auto;
    background: var(--el-bg-color-overlay);
    border-radius: 5px;
    font-size: 12px;
    overflow: hidden;
}
.popup-title {
    overflow: hidden;
    line-height: 34px;
    padding-top: 6px;
    background: #f2f2f2;
}
.popup-result {
    box-sizing: border-box;
    line-height: 24px;
    margin: 15px auto;
    padding: 15px 20px 10px;
    border: 1px solid var(--el-border-color);
    position: relative;
}
.popup-result .title {
    position: absolute;
    top: -18px;
    left: 50%;
    width: 140px;
    font-size: 14px;
    margin-left: -70px;
    text-align: center;
    line-height: 30px;
    background: var(--el-bg-color-overlay);
}
.popup-result table {
    text-align: center;
    width: 100%;
    margin: 0 auto;
}
.popup-result table span {
    display: block;
    width: 100%;
    font-family: arial;
    line-height: 30px;
    height: 30px;
    white-space: nowrap;
    overflow: hidden;
    border: 1px solid var(--el-border-color);
}
.popup-result-scroll {
    font-size: 12px;
    line-height: 24px;
    height: 10em;
    overflow-y: auto;
}

.crontab {
    ::v-deep(.el-form-item) {
        margin-bottom: 10px !important;
    }
}
</style>
