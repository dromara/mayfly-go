<template>
    <div class="card flex flex-col h-full p-4 layout-link-container">
        <div class="flex-1 overflow-auto layout-padding-view">
            <div class="flex flex-col items-center justify-center h-full layout-link-warp">
                <i class="relative text-8xl text-primary layout-link-icon iconfont icon-xingqiu">
                    <span
                        class="absolute top-0 left-[50px] w-4 h-24 bg-gradient-to-b from-white/5 via-white/20 to-white/5 transform -rotate-12 animate-pulse"
                    ></span>
                </i>
                <div class="mt-4 text-sm text-gray-500 opacity-70 layout-link-msg">页面 "{{ $t(state.title) }}" 已在新窗口中打开</div>
                <el-button class="mt-8 rounded-full" round size="default" @click="onGotoFullPage">
                    <i class="iconfont icon-lianjie"></i>
                    <span>立即前往体验</span>
                </el-button>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts" name="layoutLinkView">
import { reactive, watch } from 'vue';
import { useRoute } from 'vue-router';

// 定义变量内容
const route = useRoute();
const state = reactive({
    title: '',
    link: '',
});

// 立即前往
const onGotoFullPage = () => {
    window.open(state.link);
    // const { origin, pathname } = window.location;
    // if (verifyUrl(<string>state.isLink)) window.open(state.isLink);
    // else window.open(`${origin}${pathname}#${state.isLink}`);
};

// 监听路由的变化，设置内容
watch(
    () => route.path,
    () => {
        state.title = <string>route.meta.title;
        state.link = <string>route.meta.link;
    },
    {
        immediate: true,
    }
);
</script>

<style scoped lang="scss">
.layout-link-container {
    .layout-link-warp {
        margin: auto;
        .layout-link-msg {
            font-size: 12px;
            color: var(--next-bg-topBarColor);
            opacity: 0.7;
            margin-top: 15px;
        }
    }
}
</style>
