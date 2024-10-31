<template>
    <el-form size="small">
        <el-form-item>
            <el-radio :label="1" v-model="radioValue"> 不填，允许的通配符[, - * /] </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio :label="2" v-model="radioValue"> 每年 </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio :label="3" v-model="radioValue">
                周期从
                <el-input-number v-model="cycle01" :min="fullYear" /> -
                <el-input-number v-model="cycle02" :min="fullYear" />
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio :label="4" v-model="radioValue">
                从
                <el-input-number v-model="average01" :min="fullYear" /> 年开始，每 <el-input-number v-model="average02" :min="fullYear" /> 年执行一次
            </el-radio>
        </el-form-item>

        <el-form-item>
            <div class="flex-align-center w100">
                <el-radio v-model="radioValue" :label="5" class="mr5"> 指定 </el-radio>
                <el-select @click="radioValue = 5" class="w100" clearable v-model="checkboxList" placeholder="可多选" multiple>
                    <el-option v-for="item in 9" :key="item" :value="`${item - 1 + fullYear}`" :label="item - 1 + fullYear" />
                </el-select>
            </div>
        </el-form-item>
    </el-form>
</template>

<script lang="ts" setup>
import { computed, toRefs, watch, onMounted, reactive } from 'vue';
import { checkNumber, CrontabValueObj } from './index';

const cron = defineModel<CrontabValueObj>('cron', { required: true });

const state = reactive({
    fullYear: 0,
    radioValue: 1,
    cycle01: 0,
    cycle02: 0,
    average01: 0,
    average02: 1,
    checkboxList: [] as any,
});

const { radioValue, cycle01, cycle02, average01, average02, checkboxList, fullYear } = toRefs(state);

onMounted(() => {
    // 仅获取当前年份
    state.fullYear = Number(new Date().getFullYear());
});

// 单选按钮值变化时
function radioChange() {
    switch (state.radioValue) {
        case 1:
            cron.value.year = '';
            break;
        case 2:
            cron.value.year = '*';
            break;
        case 3:
            cron.value.year = state.cycle01 + '-' + state.cycle02;
            break;
        case 4:
            cron.value.year = state.average01 + '/' + state.average02;
            break;
        case 5:
            cron.value.year = checkboxString.value;
            break;
    }
}

// 周期两个值变化时
function cycleChange() {
    state.cycle01 = checkNumber(state.cycle01, state.fullYear, state.fullYear + 100);
    state.cycle02 = checkNumber(state.cycle02, state.fullYear + 1, state.fullYear + 101);
    if (state.radioValue == 3) {
        cron.value.year = cycleTotal.value;
    }
}

// 平均两个值变化时
function averageChange() {
    state.average01 = checkNumber(state.average01, state.fullYear, state.fullYear + 100);
    state.average02 = checkNumber(state.average02, 1, 10);
    if (state.radioValue == 4) {
        cron.value.year = averageTotal.value;
    }
}

// checkbox值变化时
function checkboxChange() {
    if (state.radioValue == 5) {
        cron.value.year = checkboxString.value;
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
    let value = cron.value.year;
    if (value == '') {
        state.radioValue = 1;
    } else if (value == '*') {
        state.radioValue = 2;
    } else if (value.indexOf('-') > -1) {
        state.radioValue = 3;
    } else if (value.indexOf('/') > -1) {
        state.radioValue = 4;
    } else {
        state.checkboxList = value.split(',');
        state.radioValue = 5;
    }
};

defineExpose({ parse });
</script>
