<template>
    <el-select v-bind="$attrs" v-model="modelValue">
        <el-option v-for="item in enumValues" :key="item.value" :label="$t(item.label)" :value="item.value"> </el-option>
    </el-select>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import type { EnumValue } from '@/common/Enum';

const props = defineProps<{
    enums: Record<string, EnumValue>;
}>();

const modelValue = defineModel<string | number | null>();

const enumValues = computed(() => Object.values(props.enums));
</script>
