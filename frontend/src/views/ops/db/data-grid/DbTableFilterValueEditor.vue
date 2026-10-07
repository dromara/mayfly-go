<template>
    <!-- 时间列：EP date-picker 的 type 枚举不含 time（传入会静默回退日面板），须用 time-picker -->
    <el-time-picker
        v-if="pickerType === 'time'"
        v-model="model"
        value-format="HH:mm:ss"
        size="small"
        style="--el-date-editor-width: 100%"
        :teleported="false"
        :placeholder="placeholder"
    />
    <el-date-picker
        v-else-if="pickerType"
        v-model="model"
        :type="pickerType"
        :value-format="DATE_VALUE_FORMAT[pickerType]"
        size="small"
        style="--el-date-editor-width: 100%"
        :teleported="false"
        :placeholder="placeholder"
    />
    <el-input v-else v-model="model" size="small" :placeholder="placeholder" @keyup.enter="emit('enter')" />
</template>

<script lang="ts" setup>
import { computed } from 'vue';

import type { DbDialect } from '@/views/ops/db/dialect';
import type { TableColumnDef } from '@/views/ops/db/types';
import { datePickerType, DATE_VALUE_FORMAT, type TableFilterCondition } from './filterModel';

/**
 * 条件值编辑器（类型感知）：按列类型切换 time-picker / date-picker / 文本输入，
 * 直接读写传入条件对象上的值字段（endpoint 指定单值或区间端点），并内聚
 * EP 宽度覆盖（--el-date-editor-width）与下拉不 teleport 两个细节，新增类型只需扩展 filterModel。
 */
const props = defineProps<{
    cond: TableFilterCondition;
    column?: TableColumnDef;
    dialect: DbDialect;
    /** 区间（BETWEEN）端点字段，缺省为单值字段 value */
    endpoint?: 'value' | 'value2';
    placeholder?: string;
}>();

const emit = defineEmits<{
    enter: [];
}>();

const pickerType = computed(() => datePickerType(props.column, props.dialect));

// 经 computed 转写条件对象属性：传入的是构建器草稿（本地状态副本），草稿编辑模式下原地双向生效
const cond = computed(() => props.cond);

const model = computed({
    get: () => cond.value[props.endpoint ?? 'value'],
    set: (v: string | null) => {
        cond.value[props.endpoint ?? 'value'] = v ?? '';
    },
});
</script>
