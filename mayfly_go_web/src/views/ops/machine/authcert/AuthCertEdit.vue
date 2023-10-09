<template>
    <div>
        <el-dialog :title="title" v-model="dvisible" :show-close="false" :before-close="cancel" width="500px" :destroy-on-close="true">
            <el-form ref="acForm" :rules="rules" :model="form" label-width="auto">
                <el-form-item prop="name" label="名称" required>
                    <el-input v-model="form.name"></el-input>
                </el-form-item>
                <el-form-item prop="authMethod" label="认证方式" required>
                    <el-select style="width: 100%" v-model="form.authMethod" placeholder="请选择认证方式">
                        <el-option key="1" label="密码" :value="1"> </el-option>
                        <el-option key="2" label="密钥" :value="2"> </el-option>
                    </el-select>
                </el-form-item>
                <el-form-item v-if="form.authMethod == 1" prop="password" label="密码">
                    <el-input type="password" show-password clearable v-model.trim="form.password" placeholder="请输入密码" autocomplete="new-password">
                    </el-input>
                </el-form-item>
                <el-form-item v-if="form.authMethod == 2" prop="password" label="秘钥">
                    <el-input type="textarea" :rows="5" v-model="form.password" placeholder="请将私钥文件内容拷贝至此"> </el-input>
                </el-form-item>
                <el-form-item v-if="form.authMethod == 2" prop="passphrase" label="秘钥密码">
                    <el-input type="password" v-model="form.passphrase"> </el-input>
                </el-form-item>

                <el-form-item label="备注">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancel()">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { ref, toRefs, reactive, watch } from 'vue';
import { authCertApi } from '../api';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
});

//定义事件
const emit = defineEmits(['update:visible', 'cancel', 'val-change']);

const acForm: any = ref(null);

const rules = {
    name: [
        {
            required: true,
            message: '授权凭证名称不能为空',
            trigger: ['change', 'blur'],
        },
    ],
};

const state = reactive({
    dvisible: false,
    params: [] as any,
    form: {
        id: null,
        name: '',
        authMethod: 1,
        password: '',
        passphrase: '',
        remark: '',
    },
    btnLoading: false,
});

const { dvisible, form, btnLoading } = toRefs(state);

watch(props, (newValue: any) => {
    state.dvisible = newValue.visible;
    if (newValue.data) {
        state.form = { ...newValue.data };
    } else {
        state.form = { authMethod: 1 } as any;
        state.params = [];
    }
});

const cancel = () => {
    // 更新父组件visible prop对应的值为false
    emit('update:visible', false);
    // 若父组件有取消事件，则调用
    emit('cancel');
};

const btnOk = async () => {
    acForm.value.validate(async (valid: boolean) => {
        if (valid) {
            state.btnLoading = true;
            try {
                await authCertApi.save.request(state.form);
                emit('val-change', state.form);
                cancel();
            } finally {
                state.btnLoading = false;
            }
        }
    });
};
</script>
<style lang="scss"></style>
