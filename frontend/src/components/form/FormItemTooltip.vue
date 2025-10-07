<template>
    <el-form-item v-bind="$attrs">
        <template #label>
            <div class="flex items-center">
                {{ props.label }}

                <el-tooltip :placement="props.placement">
                    <template #content>
                        <span v-html="props.tooltip"></span>
                    </template>
                    <SvgIcon name="QuestionFilled" class="ml-1" />
                </el-tooltip>
            </div>
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
        required: true,
    },
    tooltip: {
        type: String,
        required: true,
    },
    placement: {
        type: String,
        default: 'top',
    },
});
</script>
