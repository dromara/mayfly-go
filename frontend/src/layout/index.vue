<template>
    <!-- 全局背景图层 -->
    <div class="app-backdrop">
        <div class="app-backdrop__image"></div>
        <div class="app-backdrop__overlay"></div>
        <!-- 全局磨砂纱层(设置-磨砂程度滑杆驱动): 位于壁纸之上、所有面板之下,
             整屏统一糊壁纸且不透明白纱, 面板/文字保持锐利; 0 档时完全无效果 -->
        <div class="app-backdrop__frost"></div>
    </div>
    <component :is="layouts[themeConfig.layout]" />
</template>

<script setup lang="ts" name="layout">
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import { defineAsyncComponent, onMounted, type Component } from 'vue';

const layouts: Record<string, Component> = {
    defaults: defineAsyncComponent(() => import('@/layout/main/defaults.vue')),
    classic: defineAsyncComponent(() => import('@/layout/main/classic.vue')),
    transverse: defineAsyncComponent(() => import('@/layout/main/transverse.vue')),
    columns: defineAsyncComponent(() => import('@/layout/main/columns.vue')),
};

const { themeConfig } = storeToRefs(useThemeConfig());

/**
 * 进入控制台后预取 monaco 编辑器主体（生产约 967KB gzip / 3.8MB 源码），使点开编辑器时秒出。
 *
 * 放在 layout 而非 main.ts：登录页用不到编辑器，不该为它下载这份内容；layout 是登录后所有页面的共同外壳，
 * 挂载时机即「已进入控制台」，且此时首屏请求早已发出。
 * 为什么不直接静态 import：那会让每个页面的首次渲染都排队等这份下载和解析（由 monacoBoundary 守卫锁住）。
 * 为什么不立即 import：会与页面自身首屏数据请求抢带宽，故等浏览器空闲（最长 3s）再取。
 * 各使用方动态引入的都是同一个 MonacoEditor.vue，命中同一份 chunk，不会重复下载；
 * 预取失败（弱网等）不提示，真正挂载编辑器时异步组件会重新拉取。
 */
onMounted(() => {
    const prefetchEditor = () => import('@/components/monaco/MonacoEditor.vue').catch(() => undefined);
    if (window.requestIdleCallback) {
        window.requestIdleCallback(prefetchEditor, { timeout: 3000 });
    } else {
        // Safari 旧版本无 requestIdleCallback，退回到一个宏任务
        window.setTimeout(prefetchEditor, 1);
    }
});
</script>
