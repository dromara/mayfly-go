<template>
    <!-- <el-tabs v-model="activeTabName">
        <el-tab-pane :label="$t('common.basic')" :name="basic"> </el-tab-pane>
    </el-tabs> -->
</template>
<script lang="ts" setup>
import { ref } from 'vue';

const basicTabName = 'basic';

const activeTabName = ref(basicTabName);

const form = defineModel<any>('modelValue', { required: true });
</script>
