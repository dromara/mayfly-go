<template>
    <div class="flex min-h-screen bg-gradient-to-br from-blue-50 to-cyan-100 dark:from-gray-900 dark:to-gray-950">
        <div class="w-full flex items-center justify-center p-4">
            <div
                class="bg-white/90 backdrop-blur-lg border border-white rounded-3xl shadow-2xl w-full max-w-md overflow-hidden dark:bg-gray-800/90 dark:border-gray-700/50 transition-all duration-300 hover:shadow-2xl flex flex-col my-8"
            >
                <div class="bg-gradient-to-br from-cyan-500/5 to-blue-600/5 dark:from-cyan-400/5 dark:to-blue-500/5 flex-grow"></div>

                <!-- Logo and Title Section -->
                <div class="text-center pt-10 pb-6 px-4">
                    <div class="flex flex-col items-center justify-center">
                        <div class="flex items-center justify-center mb-4 transform transition-transform duration-300 hover:scale-105">
                            <img :src="themeConfig.logoIcon" class="w-16 h-16 drop-shadow-lg mr-3" />
                            <div>
                                <h1
                                    class="text-3xl font-bold bg-clip-text text-transparent bg-gradient-to-r from-cyan-600 to-blue-600 dark:from-cyan-400 dark:to-blue-400"
                                >
                                    {{ themeConfig.globalViceTitle }}
                                </h1>
                                <p v-if="themeConfig.appSlogan" class="text-sm text-gray-500 dark:text-gray-400 mt-1">{{ $t(themeConfig.appSlogan) }}</p>
                            </div>
                        </div>
                    </div>
                </div>

                <!-- Language Switch -->
                <div class="absolute top-4 right-4 z-20">
                    <el-dropdown
                        :show-timeout="70"
                        :hide-timeout="50"
                        trigger="click"
                        @command="
                            (lang: string) => {
                                themeConfig.globalI18n = lang;
                            }
                        "
                    >
                        <div class="cursor-pointer p-2 rounded-full hover:bg-white/30 dark:hover:bg-gray-700/50 transition-colors">
                            <SvgIcon
                                :size="18"
                                :name="EnumValue.getEnumByValue(I18nEnum, themeConfig.globalI18n)?.extra.icon"
                                :title="$t('layout.user.langSwitch')"
                                class="text-gray-500 hover:text-cyan-600 transition-colors dark:text-gray-400 dark:hover:text-cyan-400"
                            />
                        </div>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item
                                    v-for="item in I18nEnum"
                                    :key="item.value"
                                    :command="item.value"
                                    :disabled="themeConfig.globalI18n === item.value"
                                    class="flex items-center"
                                >
                                    <span class="mr-2">{{ item.extra.flag }}</span>
                                    {{ item.label }}
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </div>

                <!-- Login Form Section -->
                <div class="px-8 pb-8 flex-grow">
                    <div v-if="!state.isScan">
                        <el-tabs v-model="state.tabsActiveName" class="custom-tabs">
                            <el-tab-pane :label="$t('login.accountPasswordLogin')" name="account">
                                <Account ref="loginForm" />
                            </el-tab-pane>
                        </el-tabs>
                    </div>

                    <!-- Third Party Login Divider -->
                    <div class="mt-8 flex items-center" v-if="state.oauth2LoginConfig.enable">
                        <div class="flex-1 border-t border-gray-200 dark:border-gray-600"></div>
                        <span class="px-4 text-sm text-gray-500 bg-white dark:bg-gray-800 dark:text-gray-400">{{ $t('login.thirdPartyLogin') }}</span>
                        <div class="flex-1 border-t border-gray-200 dark:border-gray-600"></div>
                    </div>

                    <!-- OAuth2 Login Button -->
                    <div class="mt-6 text-center" v-if="state.oauth2LoginConfig.enable">
                        <el-tooltip :content="state.oauth2LoginConfig.name" placement="bottom">
                            <el-button
                                circle
                                type="primary"
                                @click="oauth2Login"
                                class="shadow-lg hover:shadow-xl transition-all duration-300 transform hover:scale-105 border-0"
                                size="large"
                            >
                                <SvgIcon name="link" :size="20" />
                            </el-button>
                        </el-tooltip>
                    </div>
                </div>

                <!-- Footer -->
                <div class="text-center pb-6 text-xs text-gray-500 dark:text-gray-400">
                    © {{ new Date().getFullYear() }} {{ themeConfig.globalViceTitle }}. All rights reserved.
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts" name="loginIndex">
import { ref, defineAsyncComponent, onMounted, reactive } from 'vue';
import { useThemeConfig } from '@/store/themeConfig';
import openApi from '@/common/openApi';
import config from '@/common/config';
import { storeToRefs } from 'pinia';
import { useI18n } from 'vue-i18n';
import EnumValue from '@/common/Enum';
import { I18nEnum } from '@/common/commonEnum';
import { NextLoading } from '@/common/utils/loading';

// 引入组件
const Account = defineAsyncComponent(() => import('./component/AccountLogin.vue'));

const loginForm = ref<{ loginResDeal: (data: any) => void } | null>(null);

// 定义变量内容
const storesThemeConfig = useThemeConfig();
const { themeConfig } = storeToRefs(storesThemeConfig);
const { locale } = useI18n();

const state = reactive({
    tabsActiveName: 'account',
    isScan: false,
    oauth2LoginConfig: {
        name: 'OAuth2 Login',
        enable: false,
    },
});

onMounted(async () => {
    try {
        if (themeConfig.value.isDark) {
            document.documentElement.classList.add('dark');
        } else {
            document.documentElement.classList.remove('dark');
        }
        NextLoading.start();
        storesThemeConfig.setWatermarkUser(true);
        locale.value = themeConfig.value.globalI18n;
        state.oauth2LoginConfig = await openApi.oauth2LoginConfig();
    } finally {
        NextLoading.done();
    }
});

const oauth2Login = () => {
    const width = 700;
    const height = 500;
    var iTop = (window.screen.height - 30 - height) / 2; //获得窗口的垂直位置;
    var iLeft = (window.screen.width - 10 - width) / 2; //获得窗口的水平位置;
    // 小窗口打开oauth2鉴权
    let oauthWindow = window.open(config.baseApiUrl + '/auth/oauth2/login', 'oauth2', `height=${height},width=${width},top=${iTop},left=${iLeft},location=no`);
    if (oauthWindow) {
        const handler = (e: any) => {
            window.removeEventListener('message', handler);
            if (e.data.action === 'oauthLogin') {
                loginForm.value!.loginResDeal(e.data);
            }
        };
        window.addEventListener('message', handler);
        setTimeout(() => {
            if (oauthWindow!.closed) {
                window.removeEventListener('message', handler);
            }
        }, 10000);
    }
};
</script>

<style scoped>
.custom-tabs :deep(.el-tabs__nav-wrap)::after {
    display: none;
}

.custom-tabs :deep(.el-tabs__header) {
    margin-bottom: 20px;
}

.custom-tabs :deep(.el-tabs__item) {
    font-size: 16px;
    font-weight: 500;
    color: #666;
    padding: 0 20px;
    transition: all 0.3s ease;
}

.custom-tabs :deep(.el-tabs__item.is-active) {
    color: #0ea5e9;
}

.custom-tabs :deep(.el-tabs__active-bar) {
    background-color: #0ea5e9;
}

.dark .custom-tabs :deep(.el-tabs__item) {
    color: #999;
}

.dark .custom-tabs :deep(.el-tabs__item.is-active) {
    color: #0ea5e9;
}

:deep(.el-form-item) {
    margin-bottom: 20px;
}

:deep(.el-input__wrapper) {
    border-radius: 12px;
    box-shadow: 0 0 0 1px #e5e7eb inset !important;
    transition: all 0.3s ease;
    height: 42px;
}

:deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 1px #0ea5e9 inset !important;
}

.dark :deep(.el-input__wrapper) {
    box-shadow: 0 0 0 1px #374151 inset !important;
}

.dark :deep(.el-input__wrapper.is-focus) {
    box-shadow: 0 0 0 1px #0ea5e9 inset !important;
}

/* 默认蓝色渐变按钮 */
:deep(.el-button--primary) {
    border-radius: 12px;
    height: 42px;
    font-weight: 500;
    letter-spacing: 1px;
    background: linear-gradient(135deg, #0ea5e9, #0284c7);
    border: none;
    transition: all 0.3s ease;
    font-size: 15px;
}

:deep(.el-button--primary:hover) {
    background: linear-gradient(135deg, #0284c7, #0ea5e9);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(2, 132, 199, 0.3);
}

/* 高级阴影效果 */
.shadow-2xl {
    box-shadow:
        0 4px 20px rgba(0, 0, 0, 0.05),
        0 8px 32px rgba(0, 0, 0, 0.08),
        inset 0 1px 0 rgba(255, 255, 255, 0.2);
}

.dark .shadow-2xl {
    box-shadow:
        0 4px 20px rgba(0, 0, 0, 0.2),
        0 8px 32px rgba(0, 0, 0, 0.25),
        inset 0 1px 0 rgba(255, 255, 255, 0.05);
}
</style>
