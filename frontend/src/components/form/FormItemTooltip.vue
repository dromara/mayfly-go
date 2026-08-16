<template>
    <el-form-item v-bind="$attrs">
        <template #label>
            <div class="flex items-center">
                {{ props.label }}

                <el-tooltip :placement="props.placement">
                    <template #content>
                        <span>{{ props.tooltip }}</span>
                    </template>
                    <SvgIcon name="QuestionFilled" class="ml-1" />
                </el-tooltip>
            </div>
        </template>

        <!-- 遍历父组件传入的 slots 透传给子组件 -->
        <template v-for="(_, key) in useSlots()" v-slot:[key]>
            <slot :name="key"></slot>
        </template>
    </el-form-item>
</template>

<script setup lang="ts">
import { useSlots } from 'vue';

const props = withDefaults(
    defineProps<{
        label: string;
        tooltip: string;
        placement?: string;
    }>(),
    { placement: 'top' }
);
</script>
