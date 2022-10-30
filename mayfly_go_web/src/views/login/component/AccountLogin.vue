<template>
    <div>
        <el-form ref="loginFormRef" :model="loginForm" :rules="rules" class="login-content-form" size="large">
            <el-form-item prop="username">
                <el-input type="text" placeholder="请输入用户名" prefix-icon="user" v-model="loginForm.username" clearable
                    autocomplete="off">
                </el-input>
            </el-form-item>
            <el-form-item prop="password">
                <el-input type="password" placeholder="请输入密码" prefix-icon="lock" v-model="loginForm.password"
                    autocomplete="off" show-password>
                </el-input>
            </el-form-item>
            <el-form-item v-if="isUseLoginCaptcha" prop="captcha">
                <el-row :gutter="15">
                    <el-col :span="16">
                        <el-input type="text" maxlength="6" placeholder="请输入验证码" prefix-icon="position"
                            v-model="loginForm.captcha" clearable autocomplete="off" @keyup.enter="login"></el-input>
                    </el-col>
                    <el-col :span="8">
                        <div class="login-content-code">
                            <img class="login-content-code-img" @click="getCaptcha" width="130px" height="40px"
                                :src="captchaImage" style="cursor: pointer" />
                        </div>
                    </el-col>
                </el-row>
            </el-form-item>
            <el-form-item>
                <el-button type="primary" class="login-content-submit" round @click="login" :loading="loading.signIn">
                    <span>登 录</span>
                </el-button>
            </el-form-item>
        </el-form>

        <el-dialog title="修改密码" v-model="changePwdDialog.visible" :close-on-click-modal="false" width="450px"
            :destroy-on-close="true">
            <el-form :model="changePwdDialog.form" :rules="changePwdDialog.rules" ref="changePwdFormRef"
                label-width="65px">
                <el-form-item prop="username" label="用户名" required>
                    <el-input v-model.trim="changePwdDialog.form.username" disabled></el-input>
                </el-form-item>
                <el-form-item prop="oldPassword" label="旧密码" required>
                    <el-input v-model.trim="changePwdDialog.form.oldPassword" autocomplete="new-password"
                        type="password"></el-input>
                </el-form-item>
                <el-form-item prop="newPassword" label="新密码" required>
                    <el-input v-model.trim="changePwdDialog.form.newPassword" placeholder="须为8位以上且包含字⺟⼤⼩写+数字+特殊符号"
                        type="password" autocomplete="new-password"></el-input>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelChangePwd">取 消</el-button>
                    <el-button @click="changePwd" type="primary" :loading="loading.changePwd">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { nextTick, onMounted, ref, toRefs, reactive, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { initBackEndControlRoutesFun } from '@/router/index.ts';
import { useStore } from '@/store/index.ts';
import { setSession, setUserInfo2Session, setUseWatermark2Session } from '@/common/utils/storage.ts';
import { formatAxis } from '@/common/utils/formatTime.ts';
import openApi from '@/common/openApi';
import { RsaEncrypt } from '@/common/rsa';
import { useLoginCaptcha, useWartermark } from '@/common/sysconfig';
import { letterAvatar } from '@/common/utils/string';

const rules = {
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
    captcha: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
}

const store = useStore();
const route = useRoute();
const router = useRouter();
const loginFormRef: any = ref(null);
const changePwdFormRef: any = ref(null);

const state = reactive({
    isUseLoginCaptcha: false,
    captchaImage: '',
    loginForm: {
        username: '',
        password: '',
        captcha: '',
        cid: '',
    },
    changePwdDialog: {
        visible: false,
        form: {
            username: '',
            oldPassword: '',
            newPassword: '',
        },
        rules: {
            newPassword: [
                { required: true, message: '请输入新密码', trigger: 'blur' },
                {
                    pattern: /^(?=.*[A-Za-z])(?=.*\d)(?=.*[`~!@#$%^&*()_+<>?:"{},.\/\\;'[\]])[A-Za-z\d`~!@#$%^&*()_+<>?:"{},.\/\\;'[\]]{8,}$/,
                    message: '须为8位以上且包含字⺟⼤⼩写+数字+特殊符号',
                    trigger: 'blur',
                },
            ],
        },
    },
    loading: {
        signIn: false,
        changePwd: false,
    },
});

const {
    isUseLoginCaptcha,
    captchaImage,
    loginForm,
    changePwdDialog,
    loading,
} = toRefs(state)

onMounted(async () => {
    nextTick(async () => {
        state.isUseLoginCaptcha = await useLoginCaptcha();
        getCaptcha();
    });
    // 移除公钥, 方便后续重新获取
    sessionStorage.removeItem('RsaPublicKey');
});

const getCaptcha = async () => {
    if (!state.isUseLoginCaptcha) {
        return;
    }
    let res: any = await openApi.captcha();
    state.captchaImage = res.base64Captcha;
    state.loginForm.cid = res.cid;
};

// 时间获取
const currentTime = computed(() => {
    return formatAxis(new Date());
});

// 校验登录表单并登录
const login = () => {
    loginFormRef.value.validate((valid: boolean) => {
        if (valid) {
            onSignIn();
        } else {
            return false;
        }
    });
};

// 登录
const onSignIn = async () => {
    state.loading.signIn = true;
    let loginRes;
    const originPwd = state.loginForm.password;
    try {
        const loginReq = { ...state.loginForm };
        loginReq.password = await RsaEncrypt(originPwd);
        loginRes = await openApi.login(loginReq);
        // 存储 token 到浏览器缓存
        setSession('token', loginRes.token);
        setSession('menus', loginRes.menus);
    } catch (e: any) {
        state.loading.signIn = false;
        state.loginForm.captcha = '';
        // 密码强度不足
        if (e.code && e.code == 401) {
            state.changePwdDialog.form.username = state.loginForm.username;
            state.changePwdDialog.form.oldPassword = originPwd;
            state.changePwdDialog.form.newPassword = '';
            state.changePwdDialog.visible = true;
        } else {
            getCaptcha();
        }
        return;
    }
    // 用户信息
    const userInfos = {
        name: loginRes.name,
        username: state.loginForm.username,
        // 头像
        photo: letterAvatar(state.loginForm.username),
        time: new Date().getTime(),
        // // 菜单资源code数组
        // menus: loginRes.menus,
        permissions: loginRes.permissions,
        lastLoginTime: loginRes.lastLoginTime,
        lastLoginIp: loginRes.lastLoginIp,
    };

    // 存储用户信息到浏览器缓存
    setUserInfo2Session(userInfos);
    // 1、请注意执行顺序(存储用户信息到vuex)
    store.dispatch('userInfos/setUserInfos', userInfos);
    if (!store.state.themeConfig.themeConfig.isRequestRoutes) {
        // 前端控制路由，2、请注意执行顺序
        // await initAllFun();
        await initBackEndControlRoutesFun();
        signInSuccess();
    } else {
        // 模拟后端控制路由，isRequestRoutes 为 true，则开启后端控制路由
        // 添加完动态路由，再进行 router 跳转，否则可能报错 No match found for location with path "/"
        await initBackEndControlRoutesFun();
        // 执行完 initBackEndControlRoutesFun，再执行 signInSuccess
        signInSuccess();
    }
};

// 登录成功后的跳转
const signInSuccess = () => {
    // 初始化登录成功时间问候语
    let currentTimeInfo = currentTime.value;
    // 登录成功，跳到转首页
    // 添加完动态路由，再进行 router 跳转，否则可能报错 No match found for location with path "/"
    // 如果是复制粘贴的路径，非首页/登录页，那么登录成功后重定向到对应的路径中
    route.query?.redirect ? router.push(route.query.redirect as string) : router.push('/');
    // 登录成功提示
    setTimeout(async () => {
        // 关闭 loading
        state.loading.signIn = true;
        ElMessage.success(`${currentTimeInfo}，欢迎回来！`);
        if (await useWartermark()) {
            setUseWatermark2Session(true);
        }
    }, 300);
};

const changePwd = () => {
    changePwdFormRef.value.validate(async (valid: boolean) => {
        if (!valid) {
            return false;
        }
        try {
            state.loading.changePwd = true;
            const form = state.changePwdDialog.form;
            const changePwdReq: any = { ...form };
            changePwdReq.oldPassword = await RsaEncrypt(form.oldPassword);
            changePwdReq.newPassword = await RsaEncrypt(form.newPassword);
            await openApi.changePwd(changePwdReq);
            ElMessage.success('密码修改成功, 新密码已填充至登录密码框');
            state.loginForm.password = state.changePwdDialog.form.newPassword;
            state.changePwdDialog.visible = false;
            getCaptcha();
        } finally {
            state.loading.changePwd = false;
        }
    });
};

const cancelChangePwd = () => {
    state.changePwdDialog.visible = false;
    state.changePwdDialog.form.newPassword = '';
    state.changePwdDialog.form.oldPassword = '';
    state.changePwdDialog.form.username = '';
    getCaptcha();
};
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
</style>
