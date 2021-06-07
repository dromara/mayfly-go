<template>
    <el-form class="login-content-form">
        <el-form-item>
            <el-input type="text" placeholder="请输入手机号" prefix-icon="el-icon-user" v-model="ruleForm.userName" clearable autocomplete="off">
            </el-input>
        </el-form-item>
        <el-form-item>
            <el-row :gutter="15">
                <el-col :span="16">
                    <el-input
                        type="text"
                        maxlength="4"
                        placeholder="请输入验证码"
                        prefix-icon="el-icon-position"
                        v-model="ruleForm.code"
                        clearable
                        autocomplete="off"
                    ></el-input>
                </el-col>
                <el-col :span="8">
                    <el-button>获取验证码</el-button>
                </el-col>
            </el-row>
        </el-form-item>
        <el-form-item>
            <el-button type="primary" class="login-content-submit" round>
                <span>登 录</span>
            </el-button>
        </el-form-item>
    </el-form>
</template>

<script lang="ts">
import { toRefs, reactive, defineComponent } from 'vue';
export default defineComponent({
    name: 'login',
    setup() {
        const state = reactive({
            ruleForm: {
                userName: '',
                code: '',
            },
        });
        return {
            ...toRefs(state),
        };
    },
});
</script>

<style scoped lang="scss">
.login-content-form {
    margin-top: 20px;
    .login-content-submit {
        width: 100%;
        letter-spacing: 2px;
        font-weight: 300;
        margin-top: 15px;
    }
}
</style>
