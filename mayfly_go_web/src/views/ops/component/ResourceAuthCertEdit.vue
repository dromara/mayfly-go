<template>
    <div class="auth-cert-edit">
        <el-dialog title="凭证保存" v-model="dialogVisible" :show-close="false" width="500px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form ref="acForm" :model="state.form" label-width="auto" :rules="rules">
                <el-form-item prop="ciphertextType" label="密文类型" required>
                    <el-select
                        :disabled="form.id && props.resourceEdit"
                        v-model="form.ciphertextType"
                        placeholder="请选择密文类型"
                        @change="changeCiphertextType"
                    >
                        <el-option
                            v-for="item in AuthCertCiphertextTypeEnum"
                            :key="item.value"
                            :label="item.label"
                            :value="item.value"
                            v-show="!props.disableCiphertextType?.includes(item.value)"
                        >
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="type" label="凭证类型" required>
                    <el-select style="width: 100%" v-model="form.type" placeholder="请选择凭证类型">
                        <el-option
                            v-for="item in AuthCertTypeEnum"
                            :key="item.value"
                            :label="item.label"
                            :value="item.value"
                            v-show="!props.disableType?.includes(item.value)"
                        >
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="name" label="名称" required>
                    <el-input :disabled="form.id" v-model="form.name"></el-input>
                </el-form-item>

                <template v-if="form.ciphertextType != AuthCertCiphertextTypeEnum.Public.value">
                    <el-form-item prop="username" label="用户名">
                        <el-input :disabled="form.id && props.resourceEdit" v-model="form.username"></el-input>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.Password.value" prop="ciphertext" label="密码">
                        <el-input type="password" show-password clearable v-model.trim="form.ciphertext" placeholder="请输入密码" autocomplete="new-password">
                            <template #suffix>
                                <SvgIcon v-if="form.id" v-auth="'authcert:showciphertext'" @click="getCiphertext" name="search" />
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="ciphertext" label="秘钥">
                        <div class="w100" style="position: relative">
                            <SvgIcon
                                v-if="form.id"
                                v-auth="'authcert:showciphertext'"
                                @click="getCiphertext"
                                name="search"
                                style="position: absolute; top: 5px; right: 5px; cursor: pointer; z-index: 1"
                            />
                            <el-input type="textarea" :rows="5" v-model="form.ciphertext" placeholder="请将私钥文件内容拷贝至此"> </el-input>
                        </div>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="passphrase" label="秘钥密码">
                        <el-input type="password" show-password v-model="form.extra.passphrase"> </el-input>
                    </el-form-item>
                </template>

                <template v-else>
                    <el-form-item label="公共凭证">
                        <el-select default-first-option filterable v-model="form.ciphertext" @change="changePublicAuthCert">
                            <el-option v-for="item in state.publicAuthCerts" :key="item.name" :label="item.name" :value="item.name">
                                {{ item.name }}
                                <el-divider direction="vertical" border-style="dashed" />
                                {{ item.username }}
                                <el-divider direction="vertical" border-style="dashed" />
                                <EnumTag :value="item.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />
                            </el-option>
                        </el-select>
                    </el-form-item>
                </template>

                <el-form-item label="备注">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelEdit">取 消</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, onMounted, watch } from 'vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { resourceAuthCertApi } from '../tag/api';
import { ResourceCodePattern } from '@/common/pattern';

const props = defineProps({
    authCert: {
        type: Object,
    },
    disableCiphertextType: {
        type: Array,
    },
    disableType: {
        type: Array,
    },
    // 是否为资源编辑该授权凭证
    resourceEdit: {
        type: Boolean,
        default: true,
    },
});

const DefaultForm = {
    id: null,
    name: '',
    username: '',
    ciphertextType: AuthCertCiphertextTypeEnum.Password.value,
    type: AuthCertTypeEnum.Private.value,
    ciphertext: '',
    extra: {} as any,
    remark: '',
};

const rules = {
    name: [
        {
            required: true,
            message: '请输入凭证名',
            trigger: ['change', 'blur'],
        },
        {
            pattern: ResourceCodePattern.pattern,
            message: ResourceCodePattern.message,
            trigger: ['blur'],
        },
    ],
};

const emit = defineEmits(['confirm']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const acForm: any = ref(null);

const state = reactive({
    form: { ...DefaultForm },
    btnLoading: false,
    publicAuthCerts: [] as any,
});

onMounted(() => {
    setForm(props.authCert);
});

watch(
    () => props.authCert,
    (val: any) => {
        setForm(val);
    }
);

const setForm = (val: any) => {
    if (!val.extra) {
        val.extra = {};
    }
    state.form = val;
    if (state.form.ciphertextType == AuthCertCiphertextTypeEnum.Public.value) {
        getPublicAuthCerts();
    }
};

const { form, btnLoading } = toRefs(state);

const changeCiphertextType = (val: any) => {
    if (val == AuthCertCiphertextTypeEnum.Public.value) {
        getPublicAuthCerts();
    }
};

const changePublicAuthCert = (val: string) => {
    // 使用公共授权凭证名称赋值username
    state.form.username = val;
};

const getPublicAuthCerts = async () => {
    const res = await resourceAuthCertApi.listByQuery.request({
        type: AuthCertTypeEnum.Public.value,
        pageNum: 1,
        pageSize: 100,
    });
    state.publicAuthCerts = res.list;
};

const getCiphertext = async () => {
    const res = await resourceAuthCertApi.detail.request({ name: state.form.name });
    state.form.ciphertext = res.ciphertext;
    state.form.extra.passphrase = res.extra?.passphrase;
};

const cancelEdit = () => {
    dialogVisible.value = false;
    setTimeout(() => {
        state.form = { ...DefaultForm };
    }, 300);
};

const btnOk = async () => {
    acForm.value.validate(async (valid: boolean) => {
        if (valid) {
            emit('confirm', state.form);
        }
    });
};
</script>
<style lang="scss"></style>
