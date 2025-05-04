<template>
    <el-form-item v-bind="$attrs">
        <template #label>
            {{ props.label }}

            <el-tooltip :placement="props.placement">
                <template #content>
                    <span v-html="props.tooltip"></span>
                </template>
                <SvgIcon name="QuestionFilled" />
            </el-tooltip>
        </template>

        <!-- 遍历父组件传入的 solts 透传给子组件 -->
        <template v-for="(_, key) in useSlots()" v-slot:[key]>
            <slot :name="key"></slot>
        </template>
    </el-form-item>
</template>

<script setup lang="ts">
import { useSlots } from 'vue';

const props = defineProps({
    label: {
        type: String,
        require: true,
    },
    tooltip: {
        type: String,
        require: true,
    },
    placement: {
        type: String,
        default: 'top',
    },
});
</script>
