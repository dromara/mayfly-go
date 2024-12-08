<template>
    <div class="layout-logo" v-if="setShowLogo" @click="onThemeConfigChange">
        <img :src="themeConfig.logoIcon" class="layout-logo-medium-img" />
        <span>
            {{ `${themeConfig.globalTitle}` }}
            <sub
                ><span style="font-size: 10px; color: goldenrod">{{ ` ${config.version}` }}</span></sub
            >
        </span>
    </div>
    <div class="layout-logo-size" v-else @click="onThemeConfigChange">
        <img :src="themeConfig.logoIcon" class="layout-logo-size-img" />
    </div>
</template>

<script setup lang="ts" name="layoutLogo">
import { computed } from 'vue';
import { storeToRefs } from 'pinia';
import { useThemeConfig } from '@/store/themeConfig';
import config from '@/common/config';
import mittBus from '@/common/utils/mitt';

const { themeConfig } = storeToRefs(useThemeConfig());

// 设置 logo 的显示。classic 经典布局默认显示 logo
const setShowLogo = computed(() => {
    let { isCollapse, layout } = themeConfig.value;
    return !isCollapse || layout === 'classic' || document.body.clientWidth < 1000;
});
// logo 点击实现菜单展开/收起
const onThemeConfigChange = () => {
    if (themeConfig.value.layout === 'transverse') return false;
    mittBus.emit('onMenuClick');
    themeConfig.value.isCollapse = !themeConfig.value.isCollapse;
};
</script>

<style scoped lang="scss">
.layout-logo {
    width: 220px;
    height: 50px;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: rgb(0 21 41 / 2%) 0px 1px 4px;
    color: var(--el-color-primary);
    font-size: 16px;
    cursor: pointer;
    animation: logoAnimation 0.3s ease-in-out;

    &:hover {
        span {
            color: var(--el-color-primary-light-2);
        }
    }

    &-medium-img {
        width: 20px;
        margin-right: 5px;
    }
}

.layout-logo-size {
    width: 100%;
    height: 50px;
    display: flex;
    cursor: pointer;
    animation: logoAnimation 0.3s ease-in-out;

    &-img {
        width: 20px;
        margin: auto;
    }

    &:hover {
        img {
            animation: logoAnimation 0.3s ease-in-out;
        }
    }
}
</style>
