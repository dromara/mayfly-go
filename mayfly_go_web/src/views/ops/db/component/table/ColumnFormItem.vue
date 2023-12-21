<template>
    <el-input
        v-if="dataType == DataType.String"
        :ref="(el: any) => focus && el?.focus()"
        @blur="emit('blur')"
        class="w100 mb4"
        input-style="text-align: center; height: 26px;"
        size="small"
        v-model="itemValue"
        :placeholder="placeholder"
    />

    <el-input
        v-else-if="dataType == DataType.Number"
        :ref="(el: any) => focus && el?.focus()"
        @blur="emit('blur')"
        class="w100 mb4"
        input-style="text-align: center; height: 26px;"
        size="small"
        v-model.number="itemValue"
        :placeholder="placeholder"
        type="number"
    />

    <el-date-picker
        v-else-if="dataType == DataType.Date"
        :ref="(el: any) => focus && el?.focus()"
        @change="emit('blur')"
        @blur="emit('blur')"
        class="edit-time-picker mb4"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        type="Date"
        value-format="YYYY-MM-DD"
        placeholder="选择日期"
    />

    <el-date-picker
        v-else-if="dataType == DataType.DateTime"
        :ref="(el: any) => focus && el?.focus()"
        @change="emit('blur')"
        @blur="emit('blur')"
        class="edit-time-picker mb4"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        type="datetime"
        value-format="YYYY-MM-DD HH:mm:ss"
        placeholder="选择日期时间"
    />

    <el-time-picker
        v-else-if="dataType == DataType.Time"
        :ref="(el: any) => focus && el?.focus()"
        @change="emit('blur')"
        @blur="emit('blur')"
        class="edit-time-picker mb4"
        popper-class="edit-time-picker-popper"
        size="small"
        v-model="itemValue"
        :clearable="false"
        value-format="HH:mm:ss"
        placeholder="选择时间"
    />
</template>

<script lang="ts" setup>
import { Ref } from 'vue';
import { ElInput } from 'element-plus';
import { DataType } from '../../dialect/index';
import { useVModel } from '@vueuse/core';

export interface ColumnFormItemProps {
    modelValue: string | number; // 绑定的值
    dataType: DataType; // 数据类型
    focus?: boolean; // 是否获取焦点
    placeholder?: string;
}

const props = withDefaults(defineProps<ColumnFormItemProps>(), {
    focus: false,
    dataType: DataType.String,
});

const emit = defineEmits(['update:modelValue', 'blur']);

const itemValue: Ref<any> = useVModel(props, 'modelValue', emit);
</script>

<style lang="scss">
.edit-time-picker {
    height: 26px;
    width: 100% !important;
    .el-input__prefix {
        display: none;
    }
    .el-input__inner {
        text-align: center;
    }
}

.edit-time-picker-popper {
    .el-date-picker {
        width: 250px !important;
        .el-date-picker__header {
            margin: 0 5px;
        }
        .el-picker-panel__content {
            width: unset;
            margin: 0 5px;
        }
        .el-date-picker__header-label {
            font-size: 13px;
        }
        .el-picker-panel__footer {
            padding: 0 5px;
            button {
                font-size: 11px;
                padding: 5px 6px;
                height: 20px;
            }
        }
    }
    .el-date-table {
        th {
            font-size: 10px;
            font-weight: 600;
            padding: 0;
        }
        td {
            padding: 0;
        }
    }
    .el-time-panel {
        width: 100px;

        .el-time-spinner__list {
            &::after,
            &::before {
                height: 10px;
            }
            .el-time-spinner__item {
                height: 20px;
                line-height: 20px;
            }
        }
    }
}
</style>
