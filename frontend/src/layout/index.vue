<template>
    <component :is="layouts[themeConfig.layout]" />
</template>

<script setup lang="ts" name="layout">
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { defineAsyncComponent, type Component } from 'vue';

const layouts: Record<string, Component> = {
    defaults: defineAsyncComponent(() => import('@/layout/main/defaults.vue')),
    classic: defineAsyncComponent(() => import('@/layout/main/classic.vue')),
    transverse: defineAsyncComponent(() => import('@/layout/main/transverse.vue')),
    columns: defineAsyncComponent(() => import('@/layout/main/columns.vue')),
};

const { themeConfig } = storeToRefs(useThemeConfig());
</script>
