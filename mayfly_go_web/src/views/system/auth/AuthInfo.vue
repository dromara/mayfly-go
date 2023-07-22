<template>
    <el-card>
        <template #header>
            <el-space>
                <span>登录认证</span>
                <el-text type="info">管理三方登录认证平台</el-text>
            </el-space>
        </template>
        <el-card>
            <template #header>
                <el-space>
                    <span>OAuth2.0</span>
                    <el-text type="info">自定义oauth2.0 server登录</el-text>
                </el-space>
            </template>
            <el-form ref="oauth2Form" :model="oauth2" label-width="160px" status-icon>
                <el-form-item prop="clientID" label="Client ID" required>
                    <el-input v-model="oauth2.clientID" placeholder="客户端id"></el-input>
                </el-form-item>
                <el-form-item prop="clientSecret" label="Client secret" required>
                    <el-input v-model="oauth2.clientSecret" placeholder="客户端密钥"></el-input>
                </el-form-item>
                <el-form-item prop="authorizationURL" label="Authorization URL" required>
                    <el-input v-model="oauth2.authorizationURL"
                        placeholder="授权码获取地址 例如: https://example.com/oauth/authorize"></el-input>
                </el-form-item>
                <el-form-item prop="accessTokenURL" label="Access token URL" required>
                    <el-input v-model="oauth2.accessTokenURL"
                        placeholder="访问token获取地址 例如: https://example.com/oauth/token"></el-input>
                </el-form-item>
                <el-form-item prop="resourceURL" label="Resource URL" required>
                    <el-input v-model="oauth2.resourceURL"
                        placeholder="获取用户信息地址 例如: https://example.com/api/v4/user"></el-input>
                </el-form-item>
                <el-form-item prop="redirectURL" label="Redirect URL" required>
                    <el-input v-model="oauth2.redirectURL" placeholder="mayfly地址 例如: http://localhost:8889/"></el-input>
                </el-form-item>
                <el-form-item prop="userIdentifier" label="User identifier" required>
                    <el-input v-model="oauth2.userIdentifier"
                        placeholder="用户唯一标识key 例如:username,如果有多层可以写为: data.username"></el-input>
                </el-form-item>
                <el-form-item prop="scopes" label="Scopes" required>
                    <el-input v-model="oauth2.scopes" placeholder="read_user,read_api 多个使用,分割"></el-input>
                </el-form-item>
                <el-form-item prop="autoRegister" label="自动注册">
                    <el-checkbox v-model="oauth2.autoRegister" label="开启自动注册将会自动注册账号, 否则需要手动创建账号后再进行绑定" />
                </el-form-item>
                <el-form-item>
                    <el-button type="primary" @click="onSubmit" :loading="btnLoading">保存</el-button>
                </el-form-item>
            </el-form>
        </el-card>
    </el-card>
</template>

<script lang="ts" setup>
import { onMounted, reactive, ref, toRefs } from 'vue';
import { authApi } from '../api';
import { ElMessage, FormInstance } from 'element-plus';


const oauth2Form = ref<FormInstance>();

const state = reactive({
    oauth2: {
        clientID: '',
        clientSecret: '',
        authorizationURL: '',
        accessTokenURL: '',
        resourceURL: '',
        redirectURL: '',
        userIdentifier: '',
        scopes: '',
        autoRegister: false,
    },
    btnLoading: false,
});


const { oauth2, btnLoading } = toRefs(state);

onMounted(async () => {
    const resp = await authApi.info.request();
    console.log(resp);
    if (resp.oauth2) {
        state.oauth2 = resp.oauth2;
    }
});

const onSubmit = () => {
    oauth2Form.value?.validate(async (valid) => {
        if (valid) {
            state.btnLoading = true;
            try {
                await authApi.saveOAuth2.request(oauth2.value);
            } catch (e) {
            }
            state.btnLoading = false;
            ElMessage.success('保存成功');
        }
    })
}

</script>
<style lang="scss"></style>
