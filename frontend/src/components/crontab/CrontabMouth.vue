<template>
    <el-form size="small">
        <el-form-item>
            <el-radio v-model="radioValue" :label="1"> 月，允许的通配符[, - * /] </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="2">
                周期从
                <el-input-number v-model="cycle01" :min="1" :max="12" /> - <el-input-number v-model="cycle02" :min="1" :max="12" /> 月
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="3">
                从
                <el-input-number v-model="average01" :min="1" :max="12" /> 月开始，每 <el-input-number v-model="average02" :min="1" :max="12" /> 月月执行一次
            </el-radio>
        </el-form-item>

        <el-form-item>
            <div class="flex-align-center w100">
                <el-radio v-model="radioValue" :label="4" class="mr5"> 指定 </el-radio>
                <el-select @click="radioValue = 4" class="w100" clearable v-model="checkboxList" placeholder="可多选" multiple>
                    <el-option v-for="item in 12" :key="item" :value="`${item}`">{{ item }}</el-option>
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
    cycle01: 1,
    cycle02: 2,
    average01: 1,
    average02: 1,
    checkboxList: [] as any,
});

const { radioValue, cycle01, cycle02, average01, average02, checkboxList } = toRefs(state);

// 单选按钮值变化时
function radioChange() {
    if (state.radioValue === 1) {
        cron.value.mouth = '*';
        cron.value.year = '*';
    } else {
        if (cron.value.day === '*') {
            cron.value.day = '0';
        }
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
            cron.value.mouth = state.cycle01 + '-' + state.cycle02;
            break;
        case 3:
            cron.value.mouth = state.average01 + '/' + state.average02;
            break;
        case 4:
            cron.value.mouth = checkboxString.value;
            break;
    }
}

// 周期两个值变化时
function cycleChange() {
    state.cycle01 = checkNumber(state.cycle01, 1, 12);
    state.cycle02 = checkNumber(state.cycle02, 1, 12);
    if (state.radioValue == 2) {
        cron.value.mouth = cycleTotal.value;
    }
}

// 平均两个值变化时
function averageChange() {
    state.average01 = checkNumber(state.average01, 1, 12);
    state.average02 = checkNumber(state.average02, 1, 12);
    if (state.radioValue == 3) {
        cron.value.mouth = averageTotal.value;
    }
}

// checkbox值变化时
function checkboxChange() {
    if (state.radioValue == 4) {
        cron.value.mouth = checkboxString.value;
    }
}

// 计算两个周期值
const cycleTotal = computed(() => {
    return state.cycle01 + '-' + state.cycle02;
});

// 计算平均用到的值
const averageTotal = computed(() => {
    return state.average01 + '/' + state.average02;
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

watch(checkboxString, () => {
    checkboxChange();
});

const parse = () => {
    //反解析
    let ins = cron.value.mouth;
    if (ins === '*') {
        state.radioValue = 1;
    } else if (ins.indexOf('-') > -1) {
        state.radioValue = 2;
        let indexArr = ins.split('-') as any;
        isNaN(indexArr[0]) ? (state.cycle01 = 0) : (state.cycle01 = indexArr[0]);
        state.cycle02 = indexArr[1];
    } else if (ins.indexOf('/') > -1) {
        state.radioValue = 3;
        let indexArr = ins.split('/') as any;
        isNaN(indexArr[0]) ? (state.average01 = 0) : (state.average01 = indexArr[0]);
        state.average02 = indexArr[1];
    } else {
        state.radioValue = 4;
        state.checkboxList = ins.split(',');
    }
};

defineExpose({ parse });
</script>
