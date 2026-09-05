<template>
    <div>
        <el-form ref="loginFormRef" :model="loginForm" :rules="rules" class="login-content-form" size="large">
            <el-form-item prop="username">
                <el-input type="text" :placeholder="$t('common.username')" prefix-icon="user" v-model="loginForm.username" clearable autocomplete="off">
                </el-input>
            </el-form-item>

            <el-form-item prop="password">
                <el-input
                    type="password"
                    :placeholder="$t('common.password')"
                    prefix-icon="lock"
                    v-model="loginForm.password"
                    autocomplete="off"
                    @keyup.enter="login"
                    show-password
                >
                </el-input>
            </el-form-item>

            <el-form-item v-if="accountLoginSecurity.useCaptcha" prop="captcha">
                <el-row :gutter="15">
                    <el-col :span="16">
                        <el-input
                            type="text"
                            maxlength="6"
                            :placeholder="$t('common.captcha')"
                            prefix-icon="position"
                            v-model="loginForm.captcha"
                            clearable
                            autocomplete="off"
                            @keyup.enter="login"
                        ></el-input>
                    </el-col>
                    <el-col :span="8">
                        <div class="login-content-code">
                            <img class="login-content-code-img cursor-pointer" @click="getCaptcha" width="130px" height="40px" :src="captchaImage" />
                        </div>
                    </el-col>
                </el-row>
            </el-form-item>

            <el-form-item v-if="ldapEnabled" prop="ldapLogin">
                <el-checkbox v-model="loginForm.ldapLogin" :label="$t('login.ldapLogin')" size="small" />
            </el-form-item>

            <span v-if="showLoginFailTips" style="color: #f56c6c; font-size: 12px">
                {{
                    $t('login.loginFailTip', {
                        loginFailCount: accountLoginSecurity.loginFailCount,
                        loginFailMin: accountLoginSecurity.loginFailMin,
                    })
                }}
            </span>

            <el-form-item>
                <el-button type="primary" class="login-content-submit" round @click="login" :loading="loading.signIn">
                    <span>{{ $t('login.login') }}</span>
                </el-button>
            </el-form-item>
        </el-form>

        <el-dialog :title="$t('login.changePassword')" v-model="changePwdDialog.visible" :close-on-click-modal="false" width="350px" :destroy-on-close="true">
            <auto-form ref="changePwdFormRef" v-model="changePwdDialog.form" :items="changePwdItems" label-width="auto" />

            <template #footer>
                <el-button @click="cancelChangePwd">{{ $t('common.cancel') }}</el-button>
                <el-button @click="changePwd" type="primary" :loading="loading.changePwd">
                    {{ $t('common.confirm') }}
                </el-button>
            </template>
        </el-dialog>

        <el-dialog
            :title="$t('login.otpValidation')"
            v-model="otpDialog.visible"
            @close="loading.signIn = false"
            :close-on-click-modal="false"
            width="350px"
            :destroy-on-close="true"
        >
            <auto-form ref="otpFormRef" v-model="otpDialog.form" :items="otpItems" label-width="auto" @submit.prevent>
                <template #qr>
                    <qrcode-vue :value="otpDialog.otpUrl" :size="200" level="H" />
                </template>
                <template #code>
                    <el-input-otp
                        ref="otpCodeInputRef"
                        v-model.trim="otpDialog.form.code"
                        clearable
                        @finish="otpVerify"
                        :placeholder="$t('login.enterOtpCodeTip')"
                    ></el-input-otp>
                </template>
            </auto-form>
        </el-dialog>

        <el-dialog :title="$t('login.updateBasicInfo')" v-model="baseInfoDialog.visible" :close-on-click-modal="false" width="350px" :destroy-on-close="true">
            <auto-form ref="baseInfoFormRef" v-model="baseInfoDialog.form" :items="baseInfoItems" label-width="auto" />

            <template #footer>
                <el-button @click="updateUserInfo()" type="primary" :loading="loading.updateUserConfirm">
                    {{ $t('common.confirm') }}
                </el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { RsaEncrypt } from '@/common/crypto';
import openApi from '@/common/openApi';
import { getFileUrl } from '@/common/request';
import { Rules } from '@/common/rule';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { getAccountLoginSecurity, getLdapEnabled } from '@/common/sysconfig';
import { getRefreshToken, getToken, saveRefreshToken, saveToken, saveUser } from '@/common/utils/storage';
import { letterAvatar } from '@/common/utils/string';
import { Msg } from '@/hooks/useI18n';
import { initRouter } from '@/router/index';
import { useThemeConfig } from '@/store/themeConfig';
import { useUserInfo } from '@/store/userInfo';
import { personApi } from '@/views/personal/api';
import type { LoginResult } from '@/views/system/types';
import type { ChangePwdParam } from '@/common/openApi';
import QrcodeVue from 'qrcode.vue';
import { nextTick, onMounted, reactive, ref, toRefs, useTemplateRef } from 'vue';
import type { FormInstance } from 'element-plus';
import { useRoute, useRouter } from 'vue-router';

const rules = {
    username: [Rules.requiredInput('common.username')],
    password: [Rules.requiredInput('common.password')],
    captcha: [Rules.requiredInput('common.captcha')],
};

// 定义变量内容
const storesThemeConfig = useThemeConfig();

const route = useRoute();
const router = useRouter();
const loginFormRef = ref<FormInstance | null>(null);
const changePwdFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('changePwdFormRef');
const otpFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('otpFormRef');
const otpCodeInputRef = ref<{ focus: () => void } | null>(null);
const baseInfoFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('baseInfoFormRef');

const state = reactive({
    accountLoginSecurity: {
        useCaptcha: false,
        useOtp: false,
        loginFailCount: 5,
        loginFailMin: 10,
    },
    showLoginFailTips: false,
    captchaImage: '',
    loginForm: {
        username: '',
        password: '',
        captcha: '',
        cid: '',
        ldapLogin: false,
    },
    loginRes: {} as LoginResult,
    changePwdDialog: {
        visible: false,
        form: {
            username: '',
            oldPassword: '',
            newPassword: '',
        },
    },
    otpDialog: {
        visible: false,
        otpUrl: '',
        form: {
            code: '',
            otpToken: '',
        },
    },
    baseInfoDialog: {
        visible: false,
        form: {
            username: '',
            name: '',
        },
    },
    loading: {
        signIn: false,
        changePwd: false,
        otpConfirm: false,
        updateUserConfirm: false,
    },
    ldapEnabled: false,
});

const { accountLoginSecurity, showLoginFailTips, captchaImage, loginForm, changePwdDialog, otpDialog, baseInfoDialog, loading, ldapEnabled } = toRefs(state);

/** 修改密码弹窗表单声明（username/oldPassword 仅显示必填星号，与原行为一致） */
const changePwdItems: AutoFormItem[] = [
    { prop: 'username', label: 'common.username', required: true, disabled: true },
    { prop: 'oldPassword', label: 'login.oldPassword', required: true, type: 'password', props: { autocomplete: 'new-password' } },
    {
        prop: 'newPassword',
        label: 'login.newPassword',
        required: true,
        type: 'password',
        placeholder: 'login.passwordRuleTip',
        props: { autocomplete: 'new-password' },
        rules: [Rules.requiredInput('login.newPassword'), Rules.accountPassword],
    },
];

/** OTP 校验弹窗表单声明（二维码/OTP 输入为 custom 插槽，二维码仅 otpUrl 存在时展示） */
const otpItems: AutoFormItem[] = [
    { prop: 'qr', type: 'custom', when: () => !!otpDialog.value.otpUrl },
    { prop: 'code', label: 'OTP', required: true, type: 'custom', rules: [Rules.requiredInput('OTP')] },
];

/** 基础信息完善弹窗表单声明 */
const baseInfoItems: AutoFormItem[] = [
    { prop: 'username', label: 'common.username', required: true, rules: [Rules.requiredInput('common.username'), Rules.accountUsername] },
    { prop: 'name', label: 'login.name', required: true, rules: [Rules.requiredInput('common.name')] },
];

onMounted(async () => {
    nextTick(async () => {
        const res = await getAccountLoginSecurity();
        if (res) {
            state.accountLoginSecurity = res;
        }
        getCaptcha();

        const ldap = await getLdapEnabled();
        state.ldapEnabled = ldap;
        state.loginForm.ldapLogin = ldap;
    });
    // 移除公钥, 方便后续重新获取
    sessionStorage.removeItem('RsaPublicKey');
});

const getCaptcha = async () => {
    if (!state.accountLoginSecurity.useCaptcha) {
        return;
    }
    let res = await openApi.captcha();
    state.captchaImage = res.captchaImage;
    state.loginForm.cid = res.captchaId;
};

// 校验登录表单并登录
const login = () => {
    loginFormRef.value?.validate((valid: boolean) => {
        if (valid) {
            onSignIn();
        }
    });
};

const otpVerify = async () => {
    try {
        await otpFormRef.value?.validate();
    } catch (e: unknown) {
        return false;
    }

    try {
        state.loading.otpConfirm = true;
        const res = await openApi.otpVerify(state.otpDialog.form);
        await signInSuccess(res.token, res.refresh_token);
        state.otpDialog.visible = false;
    } finally {
        state.loading.otpConfirm = false;
    }
};

// 登录
const onSignIn = async () => {
    state.loading.signIn = true;
    let loginRes;
    const originPwd = state.loginForm.password;
    try {
        const loginReq = { ...state.loginForm };
        loginReq.password = await RsaEncrypt(originPwd);
        if (state.loginForm.ldapLogin) {
            loginRes = await openApi.ldapLogin(loginReq);
        } else {
            loginRes = await openApi.login(loginReq);
        }
    } catch (e: unknown) {
        state.loading.signIn = false;
        state.loginForm.captcha = '';
        // 密码强度不足
        if ((e as Record<string, unknown>)?.code == 401) {
            state.changePwdDialog.form.username = state.loginForm.username;
            state.changePwdDialog.form.oldPassword = originPwd;
            state.changePwdDialog.form.newPassword = '';
            state.changePwdDialog.visible = true;
        } else {
            getCaptcha();
            state.showLoginFailTips = true;
        }
        return;
    }
    state.showLoginFailTips = false;
    loginResDeal(loginRes);
};

const updateUserInfo = async () => {
    try {
        await baseInfoFormRef.value?.validate();
    } catch (e: unknown) {
        return false;
    }

    try {
        state.loading.updateUserConfirm = true;
        const form = state.baseInfoDialog.form;
        await personApi.updateAccount.request(state.baseInfoDialog.form);
        state.baseInfoDialog.visible = false;
        useUserInfo().userInfo.username = form.username;
        useUserInfo().userInfo.name = form.name;
        await toIndex();
    } finally {
        state.loading.updateUserConfirm = false;
    }
};

const loginResDeal = async (loginRes: LoginResult) => {
    state.loginRes = loginRes;
    // 用户信息
    const userInfos = {
        name: loginRes.name,
        username: loginRes.username,
        time: new Date().getTime(),
        lastLoginTime: loginRes.lastLoginTime,
        lastLoginIp: loginRes.lastLoginIp,
        photo: '',
    };

    const avatarFileKey = `avatar_${loginRes.username}`;
    const avatarFileDetail = await openApi.getFileDetail([avatarFileKey]);
    // 说明存在头像文件
    if (avatarFileDetail.length > 0) {
        userInfos.photo = getFileUrl(avatarFileKey);
    } else {
        userInfos.photo = letterAvatar(loginRes.username);
    }

    // 存储用户信息到浏览器缓存
    saveUser(userInfos);
    // 1、请注意执行顺序(存储用户信息到vuex)
    useUserInfo().setUserInfo(userInfos);

    const token = loginRes.token;
    // 如果不需要otp校验，则该token即为accessToken，否则为otp校验token
    if (loginRes.otp == -1) {
        signInSuccess(token, loginRes.refresh_token);
        return;
    }

    state.otpDialog.form.otpToken = token;
    state.otpDialog.otpUrl = loginRes.otpUrl || '';
    state.otpDialog.visible = true;
    setTimeout(() => {
        otpCodeInputRef.value?.focus();
    }, 400);
};

// 登录成功后的跳转
const signInSuccess = async (accessToken: string = '', refreshToken = '') => {
    if (!accessToken) {
        accessToken = getToken();
    }
    if (!refreshToken) {
        refreshToken = getRefreshToken();
    }
    // 存储 token 到浏览器缓存
    saveToken(accessToken);
    saveRefreshToken(refreshToken);

    // 初始化路由
    await initRouter();

    // 判断是否为第一次oauth2登录，是的话需要用户填写姓名和用户名
    if (state.loginRes.isFirstOauth2Login) {
        state.baseInfoDialog.form.username = state.loginRes.username;
        state.baseInfoDialog.visible = true;
    } else {
        await toIndex();
    }
};

const toIndex = async () => {
    // 登录成功，跳到转首页
    // 添加完动态路由，再进行 router 跳转，否则可能报错 No match found for location with path "/"
    // 如果是复制粘贴的路径，非首页/登录页，那么登录成功后重定向到对应的路径中
    route.query?.redirect ? router.push(route.query.redirect as string) : router.push('/');
    // 登录成功提示
    setTimeout(async () => {
        // 关闭 loading
        state.loading.signIn = true;
        Msg.success('login.loginSuccessTip');
        // 水印设置用户信息
        storesThemeConfig.setWatermarkUser();
    }, 300);
};

const changePwd = async () => {
    try {
        await changePwdFormRef.value?.validate();
    } catch (e: unknown) {
        return false;
    }

    try {
        state.loading.changePwd = true;
        const form = state.changePwdDialog.form;
        const changePwdReq: ChangePwdParam = { ...form };
        changePwdReq.oldPassword = await RsaEncrypt(form.oldPassword);
        changePwdReq.newPassword = await RsaEncrypt(form.newPassword);
        await openApi.changePwd(changePwdReq);
        Msg.success('login.passwordChangeSuccessTip');
        state.loginForm.password = state.changePwdDialog.form.newPassword;
        state.changePwdDialog.visible = false;
        getCaptcha();
    } finally {
        state.loading.changePwd = false;
    }
};

const cancelChangePwd = () => {
    state.changePwdDialog.visible = false;
    state.changePwdDialog.form.newPassword = '';
    state.changePwdDialog.form.oldPassword = '';
    state.changePwdDialog.form.username = '';
    getCaptcha();
};

defineExpose({
    loginResDeal,
});
</script>

<style scoped lang="scss">
.login-content-form {
    margin-top: 20px;

    .login-content-code {
        display: flex;
        align-items: center;
        justify-content: space-around;

        .login-content-code-img {
            width: 100%;
            height: 40px;
            line-height: 40px;
            background-color: #ffffff;
            border: 1px solid rgb(220, 223, 230);
            color: #333;
            font-size: 16px;
            font-weight: 700;
            letter-spacing: 5px;
            text-indent: 5px;
            text-align: center;
            cursor: pointer;
            transition: all ease 0.2s;
            border-radius: 4px;
            user-select: none;

            &:hover {
                border-color: #c0c4cc;
                transition: all ease 0.2s;
            }
        }
    }

    .login-content-submit {
        width: 100%;
        letter-spacing: 2px;
        font-weight: 300;
        margin-top: 15px;
    }
}

// 修复表单label与输入框不对齐的问题
:deep(.el-form-item .el-form-item__label) {
    align-items: center;
    display: flex;
    height: 100%;
}
</style>
