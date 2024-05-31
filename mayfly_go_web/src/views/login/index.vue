<template>
    <div class="login-container flex">
        <div class="login-left">
            <div class="login-left-logo">
                <img :src="themeConfig.logoIcon" />
                <div class="login-left-logo-text">
                    <span>{{ themeConfig.globalViceTitle }}</span>
                    <!-- <span class="login-left-logo-text-msg">mayfly-go</span> -->
                </div>
            </div>
            <div class="login-left-img">
                <img :src="loginBgImg" />
            </div>
            <img :src="loginBgSplitImg" class="login-left-waves" />
        </div>
        <div class="login-right flex">
            <div class="login-right-warp flex-margin">
                <span class="login-right-warp-one"></span>
                <span class="login-right-warp-two"></span>
                <div class="login-right-warp-mian">
                    <div class="login-right-warp-main-title">{{ themeConfig.globalViceTitle }}</div>
                    <div class="login-right-warp-main-form">
                        <div v-if="!state.isScan">
                            <el-tabs v-model="state.tabsActiveName">
                                <el-tab-pane label="账号密码登录" name="account">
                                    <Account ref="loginForm" />
                                </el-tab-pane>
                            </el-tabs>
                        </div>
                        <div class="mt20" v-show="state.oauth2LoginConfig.enable">
                            <el-button link size="small">第三方登录: </el-button>
                            <el-tooltip :content="state.oauth2LoginConfig.name" placement="top-start">
                                <el-button link size="small" type="primary" @click="oauth2Login">
                                    <el-icon :size="18">
                                        <Link />
                                    </el-icon>
                                </el-button>
                            </el-tooltip>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts" name="loginIndex">
import { ref, defineAsyncComponent, onMounted, reactive } from 'vue';
import { useThemeConfig } from '@/store/themeConfig';
import loginBgImg from '@/assets/image/login-bg-main.svg';
import loginBgSplitImg from '@/assets/image/login-bg-split.svg';
import openApi from '@/common/openApi';
import config from '@/common/config';
import { storeToRefs } from 'pinia';

// 引入组件
const Account = defineAsyncComponent(() => import('./component/AccountLogin.vue'));

const loginForm = ref<{ loginResDeal: (data: any) => void } | null>(null);

// 定义变量内容
const storesThemeConfig = useThemeConfig();
const { themeConfig } = storeToRefs(storesThemeConfig);

const state = reactive({
    tabsActiveName: 'account',
    isScan: false,
    oauth2LoginConfig: {
        name: 'OAuth2登录',
        enable: false,
    },
});

onMounted(async () => {
    storesThemeConfig.setWatermarkUser(true);
    state.oauth2LoginConfig = await openApi.oauth2LoginConfig();
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
            if (e.data.action === 'oauthLogin') {
                window.removeEventListener('message', handler);
                loginForm.value!.loginResDeal(e.data);
            }
        };
        window.addEventListener('message', handler);
        setInterval(() => {
            if (oauthWindow!.closed) {
                window.removeEventListener('message', handler);
            }
        }, 1000);
    }
};
</script>

<style scoped lang="scss">
.login-container {
    height: 100%;
    background: var(--bg-main-color);
    .login-left {
        flex: 1;
        position: relative;
        background-color: rgba(211, 239, 255, 1);
        margin-right: 100px;
        .login-left-logo {
            display: flex;
            align-items: center;
            position: absolute;
            top: 50px;
            left: 80px;
            z-index: 1;
            animation: logoAnimation 0.3s ease;
            img {
                width: 52px;
                height: 52px;
            }
            .login-left-logo-text {
                display: flex;
                flex-direction: column;
                span {
                    margin-left: 10px;
                    font-size: 28px;
                    color: #26a59a;
                }
                .login-left-logo-text-msg {
                    font-size: 12px;
                    color: #32a99e;
                }
            }
        }
        .login-left-img {
            position: absolute;
            top: 50%;
            left: 50%;
            transform: translate(-50%, -50%);
            width: 100%;
            height: 52%;
            img {
                width: 100%;
                height: 100%;
                animation: error-num 0.6s ease;
            }
        }
        .login-left-waves {
            position: absolute;
            top: 0;
            right: -100px;
        }
    }
    .login-right {
        width: 700px;
        .login-right-warp {
            border: 1px solid var(--el-color-primary-light-3);
            border-radius: 3px;
            width: 500px;
            height: 500px;
            position: relative;
            overflow: hidden;
            background-color: var(--bg-main-color);
            .login-right-warp-one,
            .login-right-warp-two {
                position: absolute;
                display: block;
                width: inherit;
                height: inherit;
                &::before,
                &::after {
                    content: '';
                    position: absolute;
                    z-index: 1;
                }
            }
            .login-right-warp-one {
                &::before {
                    filter: hue-rotate(0deg);
                    top: 0px;
                    left: 0;
                    width: 100%;
                    height: 3px;
                    background: linear-gradient(90deg, transparent, var(--el-color-primary));
                    animation: loginLeft 3s linear infinite;
                }
                &::after {
                    filter: hue-rotate(60deg);
                    top: -100%;
                    right: 2px;
                    width: 3px;
                    height: 100%;
                    background: linear-gradient(180deg, transparent, var(--el-color-primary));
                    animation: loginTop 3s linear infinite;
                    animation-delay: 0.7s;
                }
            }
            .login-right-warp-two {
                &::before {
                    filter: hue-rotate(120deg);
                    bottom: 2px;
                    right: -100%;
                    width: 100%;
                    height: 3px;
                    background: linear-gradient(270deg, transparent, var(--el-color-primary));
                    animation: loginRight 3s linear infinite;
                    animation-delay: 1.4s;
                }
                &::after {
                    filter: hue-rotate(300deg);
                    bottom: -100%;
                    left: 0px;
                    width: 3px;
                    height: 100%;
                    background: linear-gradient(360deg, transparent, var(--el-color-primary));
                    animation: loginBottom 3s linear infinite;
                    animation-delay: 2.1s;
                }
            }
            .login-right-warp-mian {
                display: flex;
                flex-direction: column;
                height: 100%;
                .login-right-warp-main-title {
                    height: 110px;
                    line-height: 110px;
                    font-size: 27px;
                    text-align: center;
                    letter-spacing: 3px;
                    animation: logoAnimation 0.3s ease;
                    animation-delay: 0.3s;
                    color: var(--el-text-color-primary);
                }
                .login-right-warp-main-form {
                    flex: 1;
                    padding: 0 50px 50px;
                    .login-content-main-sacn {
                        position: absolute;
                        top: 0;
                        right: 0;
                        width: 50px;
                        height: 50px;
                        overflow: hidden;
                        cursor: pointer;
                        transition: all ease 0.3s;
                        color: var(--el-color-primary);
                        &-delta {
                            position: absolute;
                            width: 35px;
                            height: 70px;
                            z-index: 2;
                            top: 2px;
                            right: 21px;
                            background: var(--el-color-white);
                            transform: rotate(-45deg);
                        }
                        &:hover {
                            opacity: 1;
                            transition: all ease 0.3s;
                            color: var(--el-color-primary) !important;
                        }
                        i {
                            width: 47px;
                            height: 50px;
                            display: inline-block;
                            font-size: 48px;
                            position: absolute;
                            right: 1px;
                            top: 0px;
                        }
                    }
                }
            }
        }
    }
}
</style>
