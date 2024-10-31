<template>
    <div class="layout-footer flex-all-center" v-show="isDelayFooter">
        <div class="layout-footer-warp">
            <div>Made by mayfly-go with ❤️</div>
        </div>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive } from 'vue';
import { onBeforeRouteUpdate } from 'vue-router';
export default {
    name: 'layoutFooter',
    setup() {
        const state = reactive({
            isDelayFooter: true,
        });
        // 路由改变时，等主界面动画加载完毕再显示 footer
        onBeforeRouteUpdate(() => {
            state.isDelayFooter = false;
            setTimeout(() => {
                state.isDelayFooter = true;
            }, 800);
        });
        return {
            ...toRefs(state),
        };
    },
};
</script>

<style scoped lang="scss">
.layout-footer {
    height: 30px;
    background-color: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-light);
    .layout-footer-warp {
        font-size: 14px;
        color: var(--el-text-color-secondary);
        text-decoration: none;
        letter-spacing: 0.5px;
    }
}
</style>
