<template>
    <el-form ref="loginFormRef" :model="loginForm" :rules="rules" class="login-content-form">
        <el-form-item prop="username">
            <el-input type="text" placeholder="请输入用户名" prefix-icon="el-icon-user" v-model="loginForm.username" clearable autocomplete="off">
            </el-input>
        </el-form-item>
        <el-form-item prop="password">
            <el-input
                type="password"
                placeholder="请输入密码"
                prefix-icon="el-icon-lock"
                v-model="loginForm.password"
                autocomplete="off"
                show-password
            >
            </el-input>
        </el-form-item>
        <el-form-item prop="captcha">
            <el-row :gutter="15">
                <el-col :span="16">
                    <el-input
                        type="text"
                        maxlength="6"
                        placeholder="请输入验证码"
                        prefix-icon="el-icon-position"
                        v-model="loginForm.captcha"
                        clearable
                        autocomplete="off"
                        @keyup.enter="login"
                    ></el-input>
                </el-col>
                <el-col :span="8">
                    <div class="login-content-code">
                        <img
                            class="login-content-code-img"
                            @click="getCaptcha"
                            width="130px"
                            height="40px"
                            :src="captchaImage"
                            style="cursor: pointer"
                        />
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
</template>

<script lang="ts">
import { onMounted, ref, toRefs, reactive, defineComponent, computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ElMessage } from 'element-plus';
import { initAllFun, initBackEndControlRoutesFun } from '@/router/index.ts';
import { useStore } from '@/store/index.ts';
import { setSession } from '@/common/utils/storage.ts';
import { formatAxis } from '@/common/utils/formatTime.ts';
import openApi from '@/common/openApi';
import { letterAvatar } from '@/common/utils/string';
export default defineComponent({
    name: 'Account',
    setup() {
        const store = useStore();
        const route = useRoute();
        const router = useRouter();
        const loginFormRef: any = ref(null);
        const state = reactive({
            captchaImage: '',
            loginForm: {
                username: 'test',
                password: '123456',
                captcha: '',
                cid: '',
            },
            rules: {
                username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
                password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
                captcha: [{ required: true, message: '请输入验证码', trigger: 'blur' }],
            },
            loading: {
                signIn: false,
            },
        });

        onMounted(() => {
            getCaptcha();
        });

        const getCaptcha = async () => {
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
            try {
                loginRes = await openApi.login(state.loginForm);
                // // 存储 token 到浏览器缓存
                setSession('token', loginRes.token);
                setSession('menus', loginRes.menus);
            } catch (e) {
                state.loading.signIn = false;
                state.loginForm.captcha = '';
                getCaptcha();
                return;
            }
            // 用户信息模拟数据
            const userInfos = {
                username: state.loginForm.username,
                photo: letterAvatar(state.loginForm.username),
                time: new Date().getTime(),
                // // 菜单资源code数组
                // menus: loginRes.menus,
                permissions: loginRes.permissions,
            };

            // 存储用户信息到浏览器缓存
            setSession('userInfo', userInfos);
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
            setTimeout(() => {
                // 关闭 loading
                state.loading.signIn = true;
                ElMessage.success(`${currentTimeInfo}，欢迎回来！`);
            }, 300);
        };

        return {
            getCaptcha,
            currentTime,
            loginFormRef,
            login,
            ...toRefs(state),
        };
    },
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
</style>
