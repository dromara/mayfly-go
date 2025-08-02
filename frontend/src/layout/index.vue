<template>
    <Defaults v-if="themeConfig.layout === 'defaults'" />
    <Classic v-else-if="themeConfig.layout === 'classic'" />
    <Transverse v-else-if="themeConfig.layout === 'transverse'" />
    <Columns v-else-if="themeConfig.layout === 'columns'" />
</template>

<script setup lang="ts" name="layout">
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import Defaults from '@/layout/main/defaults.vue';
import Classic from '@/layout/main/classic.vue';
import Transverse from '@/layout/main/transverse.vue';
import Columns from '@/layout/main/columns.vue';

const { themeConfig } = storeToRefs(useThemeConfig());
</script>
