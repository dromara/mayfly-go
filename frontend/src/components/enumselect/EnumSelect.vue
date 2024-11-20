<template>
    <el-select v-bind="$attrs" v-model="modelValue">
        <el-option v-for="item in props.enums" :key="item.value" :label="$t(item.label)" :value="item.value"> </el-option>
    </el-select>
</template>

<script lang="ts" setup>
const props = defineProps({
    enums: {
        type: Object, // 需要为EnumValue类型
        required: true,
    },
});

const modelValue: any = defineModel('modelValue');
</script>
<style scoped lang="scss"></style>
