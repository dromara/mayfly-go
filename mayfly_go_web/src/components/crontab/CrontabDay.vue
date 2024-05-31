<template>
    <el-form size="small">
        <el-form-item>
            <el-radio v-model="radioValue" :label="1"> 日，允许的通配符[, - * / L M] </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="2"> 不指定 </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="3">
                周期从
                <el-input-number v-model="cycle01" :min="0" :max="31" /> - <el-input-number v-model="cycle02" :min="0" :max="31" /> 日
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="4">
                从
                <el-input-number v-model="average01" :min="0" :max="31" /> 号开始，每 <el-input-number v-model="average02" :min="0" :max="31" /> 日执行一次
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="5">
                每月
                <el-input-number v-model="workday" :min="0" :max="31" /> 号最近的那个工作日
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="6"> 本月最后一天 </el-radio>
        </el-form-item>

        <el-form-item>
            <div class="flex-align-center w100">
                <el-radio v-model="radioValue" :label="7" class="mr5"> 指定 </el-radio>
                <el-select @click="radioValue = 7" class="w100" clearable v-model="checkboxList" placeholder="可多选" multiple>
                    <el-option v-for="item in 31" :key="item" :value="`${item}`">{{ item }}</el-option>
                </el-select>
            </div>
        </el-form-item>
    </el-form>
</template>

<script lang="ts" setup>
import { computed, toRefs, watch, reactive } from 'vue';
import { checkNumber, CrontabValueObj } from './index';

const cron = defineModel<CrontabValueObj>('cron', { required: true });

const state = reactive({
    radioValue: 1,
    workday: 1,
    cycle01: 1,
    cycle02: 2,
    average01: 1,
    average02: 1,
    checkboxList: [] as any,
});

const { radioValue, workday, cycle01, cycle02, average01, average02, checkboxList } = toRefs(state);

// 单选按钮值变化时
function radioChange() {
    if (state.radioValue === 1) {
        cron.value.day = '*';
        cron.value.week = '?';
        cron.value.mouth = '*';
    } else {
        if (cron.value.hour === '*') {
            cron.value.hour = '0';
        }
        if (cron.value.min === '*') {
            cron.value.min = '0';
        }
        if (cron.value.second === '*') {
            cron.value.second = '0';
        }
    }

    switch (state.radioValue) {
        case 2:
            cron.value.day = '?';
            break;
        case 3:
            cron.value.day = state.cycle01 + '-' + state.cycle02;
            break;
        case 4:
            cron.value.day = state.average01 + '/' + state.average02;
            break;
        case 5:
            cron.value.day = state.workday + 'W';
            break;
        case 6:
            cron.value.day = 'L';
            break;
        case 7:
            cron.value.day = checkboxString.value;
            break;
    }
}
// 周期两个值变化时
function cycleChange() {
    if (state.radioValue == 3) {
        cron.value.day = cycleTotal.value;
    }
}
// 平均两个值变化时
function averageChange() {
    state.average01 = checkNumber(state.average01, 1, 31);
    state.average02 = checkNumber(state.average02, 1, 31);
    if (state.radioValue == 4) {
        cron.value.day = averageTotal.value;
    }
}
// 最近工作日值变化时
function workdayChange() {
    state.workday = checkNumber(state.workday, 1, 31);
    if (state.radioValue == 5) {
        cron.value.day = state.workday + 'W';
    }
}
// checkbox值变化时
function checkboxChange() {
    if (state.radioValue == 7) {
        cron.value.day = checkboxString.value;
    }
}

// 父组件传递的week发生变化触发
// function weekChange() {
//     //判断week值与day不能同时为“?”
//     if (cron.value.week == '?' && state.radioValue == 2) {
//         state.radioValue = 1;
//     } else if (cron.value.week !== '?' && state.radioValue != 2) {
//         state.radioValue = 2;
//     }
// }

// 计算两个周期值
const cycleTotal = computed(() => {
    return state.cycle01 + '-' + state.cycle02;
});

// 计算平均用到的值
const averageTotal = computed(() => {
    return state.average01 + '/' + state.average02;
});

// 计算工作日格式
const workdayCheck = computed(() => {
    return state.workday;
});

// 计算勾选的checkbox值合集
const checkboxString = computed(() => {
    let str = state.checkboxList.join();
    return str == '' ? '*' : str;
});

watch(
    () => state.radioValue,
    () => {
        radioChange();
    }
);

watch(cycleTotal, () => {
    cycleChange();
});

watch(averageTotal, () => {
    averageChange();
});

watch(workdayCheck, () => {
    workdayChange();
});

watch(checkboxString, () => {
    checkboxChange();
});

const parse = () => {
    //反解析
    let value = cron.value.day;
    if (value === '*') {
        state.radioValue = 1;
    } else if (value == '?') {
        state.radioValue = 2;
    } else if (value.indexOf('-') > -1) {
        state.radioValue = 3;
        let indexArr = value.split('-') as any;
        isNaN(indexArr[0]) ? (state.cycle01 = 0) : (state.cycle01 = indexArr[0]);
        state.cycle02 = indexArr[1];
    } else if (value.indexOf('/') > -1) {
        state.radioValue = 4;
        let indexArr = value.split('/') as any;
        isNaN(indexArr[0]) ? (state.average01 = 0) : (state.average01 = indexArr[0]);
        state.average02 = indexArr[1];
    } else if (value.indexOf('W') > -1) {
        state.radioValue = 5;
        let indexArr = value.split('W') as any;
        isNaN(indexArr[0]) ? (state.workday = 0) : (state.workday = indexArr[0]);
    } else if (value === 'L') {
        state.radioValue = 6;
    } else {
        state.checkboxList = value.split(',');
        state.radioValue = 7;
    }
};

defineExpose({ parse });
</script>
