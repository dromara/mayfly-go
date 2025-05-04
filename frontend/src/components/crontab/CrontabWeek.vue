<template>
    <el-form size="small">
        <el-form-item>
            <el-radio v-model="radioValue" :label="1"> {{ $t('components.crontab.weekCronType1') }} </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="2"> {{ $t('components.crontab.crontype2') }} </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="3">
                {{ $t('components.crontab.crontype3') }}
                <el-input-number v-model="cycle01" :min="1" :max="7" /> -
                <el-input-number v-model="cycle02" :min="1" :max="7" />
            </el-radio>
        </el-form-item>

        <!-- <el-form-item>
            <el-radio v-model="radioValue" :label="4">
                第
                <el-input-number v-model="average01" :min="1" :max="4" /> 周的星期
                <el-input-number v-model="average02" :min="1" :max="7" />
            </el-radio>
        </el-form-item>

        <el-form-item>
            <el-radio v-model="radioValue" :label="5">
                本月最后一个星期
                <el-input-number v-model="weekday" :min="1" :max="7" />
            </el-radio>
        </el-form-item> -->

        <el-form-item>
            <div class="flex items-center w-full">
                <el-radio v-model="radioValue" :label="6" class="mr-1"> {{ $t('components.crontab.appoint') }} </el-radio>
                <el-select @click="radioValue = 6" class="!w-full" clearable v-model="checkboxList" multiple>
                    <el-option v-for="(item, index) of weekList" :label="item" :key="index" :value="`${index + 1}`">{{ $t(item) }}</el-option>
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
    radioValue: 2,
    weekday: 1,
    cycle01: 1,
    cycle02: 2,
    average01: 1,
    average02: 1,
    checkboxList: [] as any,
    weekList: [
        'components.crontab.monday',
        'components.crontab.tuesday',
        'components.crontab.wednesday',
        'components.crontab.thursday',
        'components.crontab.friday',
        'components.crontab.saturday',
        'components.crontab.sunday',
    ],
});

const { radioValue, cycle01, cycle02, average01, average02, checkboxList, weekday, weekList } = toRefs(state);

// 单选按钮值变化时
function radioChange() {
    if (state.radioValue === 1) {
        cron.value.week = '*';
        cron.value.year = '*';
    } else {
        if (cron.value.mouth === '*') {
            cron.value.mouth = '0';
        }
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
            cron.value.week = '?';
            break;
        case 3:
            cron.value.week = state.cycle01 + '-' + state.cycle02;
            break;
        case 4:
            cron.value.week = state.average01 + '#' + state.average02;
            break;
        case 5:
            cron.value.week = state.weekday + 'L';
            break;
        case 6:
            cron.value.week = checkboxString.value;
            break;
    }
}
// 周期两个值变化时
function cycleChange() {
    state.cycle01 = checkNumber(state.cycle01, 1, 7);
    state.cycle02 = checkNumber(state.cycle02, 1, 7);
    if (state.radioValue == 3) {
        cron.value.week = cycleTotal.value;
    }
}

// 平均两个值变化时
function averageChange() {
    state.average01 = checkNumber(state.average01, 1, 4);
    state.average02 = checkNumber(state.average02, 1, 7);
    if (state.radioValue == 4) {
        cron.value.week = averageTotal.value;
    }
}

// checkbox值变化时
function checkboxChange() {
    if (state.radioValue == 6) {
        cron.value.week = checkboxString.value;
    }
}

// 计算两个周期值
const cycleTotal = computed(() => {
    return state.cycle01 + '-' + state.cycle02;
});

// 计算平均用到的值
const averageTotal = computed(() => {
    return state.average01 + '#' + state.average02;
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

watch(
    () => state.weekday,
    () => {
        state.weekday = checkNumber(state.weekday, 1, 7);
        if (state.radioValue == 5) {
            cron.value.week = state.weekday + 'L';
        }
    }
);

const parse = () => {
    //反解析
    let value = cron.value.week;
    if (!value) {
        return;
    }
    if (value === '*') {
        state.radioValue = 1;
    } else if (value == '?') {
        state.radioValue = 2;
    } else if (value.indexOf('-') > -1) {
        let indexArr = value.split('-') as any;
        isNaN(indexArr[0]) ? (state.cycle01 = 0) : (state.cycle01 = indexArr[0]);
        state.cycle02 = indexArr[1];
        state.radioValue = 3;
    } else if (value.indexOf('#') > -1) {
        let indexArr = value.split('#') as any;
        isNaN(indexArr[0]) ? (state.average01 = 1) : (state.average01 = indexArr[0]);
        state.average02 = indexArr[1];
        state.radioValue = 4;
    } else if (value.indexOf('L') > -1) {
        let indexArr = value.split('L') as any;
        isNaN(indexArr[0]) ? (state.weekday = 1) : (state.weekday = indexArr[0]);
        state.radioValue = 5;
    } else {
        state.checkboxList = value.split(',');
        state.radioValue = 6;
    }
};

defineExpose({ parse });
</script>
