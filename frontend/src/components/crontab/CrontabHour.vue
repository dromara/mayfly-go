<template>
    <el-form size="small">
        <el-form-item>
            <el-radio v-model="radioValue" :label="1"> {{ $t('components.crontab.hour') }}，{{ $t('components.crontab.hourCronType1') }} </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="2">
                {{ $t('components.crontab.crontype3') }}
                <el-input-number v-model="cycle01" :min="0" :max="60" /> - <el-input-number v-model="cycle02" :min="0" :max="60" />
                {{ $t('components.crontab.hour') }}
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="3">
                {{ $t('components.crontab.crontypeFrom') }}
                <el-input-number v-model="average01" :min="0" :max="60" /> {{ $t('components.crontab.crontypeStartHour') }}，
                {{ $t('components.crontab.crontypeEvery') }} <el-input-number v-model="average02" :min="0" :max="60" />
                {{ $t('components.crontab.crontypeExecHour') }}
            </el-radio>
        </el-form-item>

        <el-form-item>
            <div class="flex items-center w-full">
                <el-radio v-model="radioValue" :label="4" class="mr-1"> {{ $t('components.crontab.appoint') }} </el-radio>
                <el-select @click="radioValue = 4" class="!w-full" clearable v-model="checkboxList" multiple>
                    <el-option v-for="item in 60" :key="item" :value="`${item - 1}`">{{ item - 1 }}</el-option>
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
    cycle01: 0,
    cycle02: 1,
    average01: 0,
    average02: 1,
    checkboxList: [] as any,
});

const { radioValue, cycle01, cycle02, average01, average02, checkboxList } = toRefs(state);

// 单选按钮值变化时
function radioChange() {
    if (state.radioValue === 1) {
        cron.value.hour = '*';
        cron.value.day = '*';
    } else {
        if (cron.value.min === '*') {
            cron.value.min = '0';
        }
        if (cron.value.second === '*') {
            cron.value.second = '0';
        }
    }
    switch (state.radioValue) {
        case 2:
            cron.value.hour = state.cycle01 + '-' + state.cycle02;
            break;
        case 3:
            cron.value.hour = state.average01 + '/' + state.average02;
            break;
        case 4:
            cron.value.hour = checkboxString.value;
            break;
    }
}
// 周期两个值变化时
function cycleChange() {
    state.cycle01 = checkNumber(state.cycle01, 0, 23);
    state.cycle02 = checkNumber(state.cycle02, 0, 23);
    if (state.radioValue == 2) {
        cron.value.hour = cycleTotal.value;
    }
}

// 平均两个值变化时
function averageChange() {
    state.average01 = checkNumber(state.average01, 0, 23);
    state.average02 = checkNumber(state.average02, 0, 23);
    if (state.radioValue == 3) {
        cron.value.hour = averageTotal.value;
    }
}

// checkbox值变化时
function checkboxChange() {
    if (state.radioValue == 4) {
        cron.value.hour = checkboxString.value;
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
    let ins = cron.value.hour;
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
