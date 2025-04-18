<template>
    <div class="layout-navbars-breadcrumb-user" :style="{ flex: layoutUserFlexNum }">
        <div class="layout-navbars-breadcrumb-user-icon">
            <el-switch
                @change="switchDark()"
                v-model="isDark"
                active-action-icon="Moon"
                inactive-action-icon="Sunny"
                style="--el-switch-off-color: #c4c9c4; --el-switch-on-color: #2c2c2c"
                class="dark-icon"
            />
        </div>
        <!-- <el-dropdown :show-timeout="70" :hide-timeout="50" trigger="click" @command="onComponentSizeChange">
            <div class="layout-navbars-breadcrumb-user-icon">
                <el-icon title="组件大小">
                    <plus />
                </el-icon>
            </div>
            <template #dropdown>
                <el-dropdown-menu>
                    <el-dropdown-item command="" :disabled="state.disabledSize === ''">默认</el-dropdown-item>
                    <el-dropdown-item command="large" :disabled="state.disabledSize === 'large'">大型</el-dropdown-item>
                    <el-dropdown-item command="small" :disabled="state.disabledSize === 'small'">小型</el-dropdown-item>
                </el-dropdown-menu>
            </template>
</el-dropdown> -->

        <el-dropdown :show-timeout="70" :hide-timeout="50" trigger="click" @command="onLanguageChange">
            <div class="layout-navbars-breadcrumb-user-icon">
                <SvgIcon :size="16" :name="EnumValue.getEnumByValue(I18nEnum, themeConfig.globalI18n)?.extra.icon" :title="$t('layout.user.langSwitch')" />
            </div>
            <template #dropdown>
                <el-dropdown-menu>
                    <el-dropdown-item v-for="item in I18nEnum" :key="item.value" :command="item.value" :disabled="themeConfig.globalI18n === item.value">
                        {{ item.label }}
                    </el-dropdown-item>
                </el-dropdown-menu>
            </template>
        </el-dropdown>

        <div class="layout-navbars-breadcrumb-user-icon" @click="onSearchClick">
            <SvgIcon name="search" :title="$t('layout.user.menuSearch')" />
        </div>
        <div class="layout-navbars-breadcrumb-user-icon" @click="onLayoutSetingClick">
            <SvgIcon name="setting" :title="$t('layout.user.layoutConf')" />
        </div>
        <div class="layout-navbars-breadcrumb-user-icon">
            <el-popover placement="bottom" trigger="click" :visible="state.isShowUserNewsPopover" :width="300" popper-class="el-popover-pupop-user-news">
                <template #reference>
                    <el-badge :is-dot="false" @click="state.isShowUserNewsPopover = !state.isShowUserNewsPopover">
                        <SvgIcon name="bell" :title="$t('layout.user.news')" />
                    </el-badge>
                </template>
                <transition name="el-zoom-in-top">
                    <UserNews v-show="state.isShowUserNewsPopover" />
                </transition>
            </el-popover>
        </div>
        <div class="layout-navbars-breadcrumb-user-icon mr-2" @click="onScreenfullClick">
            <SvgIcon v-if="!state.isScreenfull" name="full-screen" :title="$t('layout.user.fullScreenOff')" />
            <SvgIcon v-else name="crop" />
        </div>
        <el-dropdown trigger="click" :show-timeout="70" :hide-timeout="50" @command="onHandleCommandClick">
            <span class="layout-navbars-breadcrumb-user-link cursor-pointer">
                <img :src="userInfo.photo" class="layout-navbars-breadcrumb-user-link-photo mr-1" />
                {{ userInfo.name || userInfo.username }}
                <i class="el-icon-arrow-down el-icon--right"></i>
            </span>
            <template #dropdown>
                <el-dropdown-menu>
                    <el-dropdown-item command="/home">{{ $t('layout.user.index') }}</el-dropdown-item>
                    <el-dropdown-item command="/personal">{{ $t('layout.user.personalCenter') }}</el-dropdown-item>
                    <el-dropdown-item divided command="logOut">{{ $t('layout.user.logout') }}</el-dropdown-item>
                </el-dropdown-menu>
            </template>
        </el-dropdown>
        <SearchMenu ref="searchRef" />
    </div>
</template>

<script setup lang="ts" name="layoutBreadcrumbUser">
import { ref, computed, reactive, onMounted, watch } from 'vue';
import { useRouter } from 'vue-router';
import { ElMessageBox, ElMessage } from 'element-plus';
import screenfull from 'screenfull';
import { resetRoute } from '@/router/index';
import { storeToRefs } from 'pinia';
import { useUserInfo } from '@/store/userInfo';
import { useThemeConfig } from '@/store/themeConfig';
import { clearSession } from '@/common/utils/storage';
import UserNews from '@/layout/navBars/breadcrumb/userNews.vue';
import SearchMenu from '@/layout/navBars/breadcrumb/search.vue';
import mittBus from '@/common/utils/mitt';
import openApi from '@/common/openApi';
import { getThemeConfig } from '@/common/utils/storage';
import { useDark, usePreferredDark } from '@vueuse/core';
import { useI18n } from 'vue-i18n';
import { I18nEnum } from '@/common/commonEnum';
import EnumValue from '@/common/Enum';

const router = useRouter();
const searchRef = ref();
const state = reactive({
    isScreenfull: false,
    isShowUserNewsPopover: false,
    disabledSize: '',
});
const { userInfo } = storeToRefs(useUserInfo());
const themeConfigStore = useThemeConfig();
const { themeConfig } = storeToRefs(themeConfigStore);
const { t } = useI18n();

// 设置分割样式
const layoutUserFlexNum = computed(() => {
    let { layout, isClassicSplitMenu } = themeConfig.value;
    let num = '';
    if (layout === 'defaults' || (layout === 'classic' && !isClassicSplitMenu) || layout === 'columns') num = '1';
    else num = '';
    return num;
});

// 页面加载时
onMounted(() => {
    const themeConfig = getThemeConfig();
    if (themeConfig) {
        initComponentSize();
        isDark.value = themeConfig.isDark;
    }
});

// 全屏点击时
const onScreenfullClick = () => {
    if (!screenfull.isEnabled) {
        ElMessage.warning('暂不不支持全屏');
        return false;
    }
    screenfull.toggle();
    state.isScreenfull = !state.isScreenfull;
};
// 布局配置 icon 点击时
const onLayoutSetingClick = () => {
    mittBus.emit('openSetingsDrawer');
};
// 下拉菜单点击时
const onHandleCommandClick = (path: string) => {
    if (path === 'logOut') {
        ElMessageBox({
            closeOnClickModal: false,
            closeOnPressEscape: false,
            title: t('layout.user.logOutTitle'),
            message: t('layout.user.logOutMessage'),
            showCancelButton: true,
            confirmButtonText: t('common.confirm'),
            cancelButtonText: t('common.cancel'),
            beforeClose: async (action, instance, done) => {
                if (action === 'confirm') {
                    await openApi.logout();
                    instance.confirmButtonLoading = true;
                    instance.confirmButtonText = t('layout.user.logOutExit');
                    setTimeout(() => {
                        done();
                        setTimeout(() => {
                            instance.confirmButtonLoading = false;
                        }, 300);
                    }, 700);
                } else {
                    done();
                }
            },
        })
            .then(() => {
                clearSession(); // 清除缓存/token等
                resetRoute(); // 删除/重置路由
                router.push('/login');
                setTimeout(() => {
                    ElMessage.success(t('layout.user.logoutSuccess'));
                }, 300);
            })
            .catch(() => {});
    } else {
        router.push(path);
    }
};

const isDark = useDark();
const preDark = usePreferredDark();

watch(preDark, (newValue) => {
    isDark.value = newValue;
    switchDark();
});

const switchDark = () => {
    themeConfigStore.switchDark(isDark.value);
};

// // 菜单搜索点击
const onSearchClick = () => {
    searchRef.value.openSearch();
};

// 组件大小改变
// const onComponentSizeChange = (size: string) => {
//     removeLocal('themeConfig');
//     themeConfig.value.globalComponentSize = size;
//     saveThemeConfig(themeConfig.value);
//     // proxy.$ELEMENT.size = size;
//     initComponentSize();
//     window.location.reload();
// };

// 初始化全局组件大小
const initComponentSize = () => {
    switch (getThemeConfig().globalComponentSize) {
        case '':
            state.disabledSize = '';
            break;
        case 'default':
            state.disabledSize = 'default';
            break;
        case 'small':
            state.disabledSize = 'small';
            break;
        case 'large':
            state.disabledSize = 'large';
            break;
    }
};

// 语言切换
const onLanguageChange = (lang: string) => {
    themeConfig.value.globalI18n = lang;
};
</script>

<style scoped lang="scss">
.layout-navbars-breadcrumb-user {
    display: flex;
    align-items: center;
    justify-content: flex-end;

    &-link {
        height: 100%;
        display: flex;
        align-items: center;
        white-space: nowrap;

        &-photo {
            width: 25px;
            height: 25px;
            border-radius: 100%;
        }
    }

    &-icon {
        padding: 0 10px;
        cursor: pointer;
        color: var(--bg-topBarColor);
        height: 50px;
        line-height: 50px;
        display: flex;
        align-items: center;

        &:hover {
            background: rgba(0, 0, 0, 0.04);

            i {
                display: inline-block;
                animation: logoAnimation 0.3s ease-in-out;
            }
        }
    }

    ::v-deep(.el-dropdown) {
        color: var(--bg-topBarColor);
    }

    ::v-deep(.el-badge) {
        height: 40px;
        line-height: 40px;
        display: flex;
        align-items: center;
    }

    ::v-deep(.el-badge__content.is-fixed) {
        top: 12px;
    }
}
</style>
