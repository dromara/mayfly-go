<template>
    <div class="layout-navbars-container">
        <BreadcrumbIndex />
        <TagsView v-if="setShowTagsView" />
    </div>
</template>

<script setup lang="ts" name="layoutNavBars">
import { computed } from 'vue';
import { useThemeConfig } from '@/store/themeConfig';
import BreadcrumbIndex from '@/layout/navBars/breadcrumb/index.vue';
import TagsView from '@/layout/navBars/tagsView/tagsView.vue';

// 是否显示 tagsView
const setShowTagsView = computed(() => {
    let { layout, isTagsview } = useThemeConfig().themeConfig;
    return layout !== 'classic' && isTagsview;
});
</script>

<style scoped lang="scss">
.layout-navbars-container {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
}
</style>
