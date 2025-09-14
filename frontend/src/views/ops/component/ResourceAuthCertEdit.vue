<template>
    <div class="auth-cert-edit">
        <el-dialog :title="props.title" v-model="dialogVisible" :show-close="false" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form ref="acForm" :model="state.form" label-width="auto" :rules="rules">
                <el-form-item prop="type" :label="$t('ac.credentialType')" required>
                    <el-select @change="changeType" v-model="form.type">
                        <el-option
                            v-for="item in AuthCertTypeEnum"
                            :key="item.value"
                            :label="$t(item.label)"
                            :value="item.value"
                            v-show="!props.disableType?.includes(item.value)"
                        >
                        </el-option>
                    </el-select>
                </el-form-item>

                <el-form-item prop="ciphertextType" :label="$t('ac.ciphertextType')" required>
                    <el-select v-model="form.ciphertextType" @change="changeCiphertextType">
                        <el-option
                            v-for="item in AuthCertCiphertextTypeEnum"
                            :key="item.value"
                            :label="$t(item.label)"
                            :value="item.value"
                            v-show="!props.disableCiphertextType?.includes(item.value)"
                            :disabled="item.value == AuthCertCiphertextTypeEnum.Public.value && form.type == AuthCertTypeEnum.Public.value"
                        >
                        </el-option>
                    </el-select>
                </el-form-item>

                <template v-if="showResourceEdit">
                    <el-form-item prop="type" :label="$t('ac.resourceType')" required>
                        <el-select :disabled="form.id" v-model="form.resourceType">
                            <el-option
                                :key="TagResourceTypeEnum.Machine.value"
                                :label="$t(TagResourceTypeEnum.Machine.label)"
                                :value="TagResourceTypeEnum.Machine.value"
                            />
                            <el-option
                                :key="TagResourceTypeEnum.DbInstance.value"
                                :label="$t(TagResourceTypeEnum.DbInstance.label)"
                                :value="TagResourceTypeEnum.DbInstance.value"
                            />
                            <el-option
                                :key="TagResourceTypeEnum.Redis.value"
                                :label="$t(TagResourceTypeEnum.Redis.label)"
                                :value="TagResourceTypeEnum.Redis.value"
                            />
                        </el-select>
                    </el-form-item>
                    <el-form-item prop="resourceCode" :label="$t('ac.resourceCode')" required>
                        <el-input :disabled="form.id" v-model="form.resourceCode"></el-input>
                    </el-form-item>
                </template>

                <el-form-item v-if="form.type == AuthCertTypeEnum.Public.value" prop="name" :label="$t('common.name')" required>
                    <el-input :disabled="form.id" v-model="form.name" :placeholder="$t('ac.namePlaceholder')"></el-input>
                </el-form-item>

                <template v-if="form.ciphertextType != AuthCertCiphertextTypeEnum.Public.value">
                    <el-form-item prop="username" :label="$t('common.username')">
                        <el-input v-model="form.username"></el-input>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.Password.value" prop="ciphertext" :label="$t('common.password')">
                        <el-input type="password" show-password clearable v-model.trim="form.ciphertext" autocomplete="new-password">
                            <template #suffix>
                                <SvgIcon v-if="form.id" v-auth="'authcert:showciphertext'" @click="getCiphertext" name="search" />
                            </template>
                        </el-input>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="ciphertext" :label="$t('ac.privateKey')">
                        <div class="!w-full" style="position: relative">
                            <SvgIcon
                                v-if="form.id"
                                v-auth="'authcert:showciphertext'"
                                @click="getCiphertext"
                                name="search"
                                style="position: absolute; top: 5px; right: 5px; cursor: pointer; z-index: 1"
                            />
                            <el-input type="textarea" :rows="5" v-model="form.ciphertext" :placeholder="$t('ac.privateKeyPlaceholder')"> </el-input>
                        </div>
                    </el-form-item>

                    <el-form-item v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value" prop="passphrase" :label="$t('ac.privateKeyPwd')">
                        <el-input type="password" show-password v-model="form.extra.passphrase"> </el-input>
                    </el-form-item>
                </template>

                <template v-else>
                    <el-form-item :label="$t('ac.publicAc')">
                        <el-select default-first-option filterable v-model="form.ciphertext" @change="changePublicAuthCert">
                            <el-option v-for="item in state.publicAuthCerts" :key="item.name" :label="item.name" :value="item.name">
                                {{ item.name }}
                                <el-divider direction="vertical" border-style="dashed" />
                                {{ item.username }}
                                <el-divider direction="vertical" border-style="dashed" />
                                <EnumTag :value="item.ciphertextType" :enums="AuthCertCiphertextTypeEnum" />
                                <el-divider direction="vertical" border-style="dashed" />
                                {{ item.remark }}
                            </el-option>
                        </el-select>
                    </el-form-item>
                </template>

                <el-form-item :label="$t('common.remark')">
                    <el-input v-model="form.remark" type="textarea" :rows="2"></el-input>
                </el-form-item>
            </el-form>
            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelEdit">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" :loading="btnLoading" @click="btnOk">{{ $t('common.confirm') }}</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { reactive, ref, toRefs, computed, watch } from 'vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
import EnumTag from '@/components/enumtag/EnumTag.vue';
import { resourceAuthCertApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';

const props = defineProps({
    title: {
        type: String,
        default: '',
    },
    authCert: {
        type: Object,
    },
    disableCiphertextType: {
        type: Array,
    },
    disableType: {
        type: Array,
    },
    // 是否为资源编辑该授权凭证，即机器编辑等页面等
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
    resourceType: TagResourceTypeEnum.PublicAuthCert.value,
    resourceCode: '',
    ciphertext: '',
    extra: {} as any,
    remark: '',
};

const rules = {
    name: [Rules.requiredInput('common.name'), Rules.resourceCode],
    resourceCode: [Rules.requiredInput('ac.resourceCode')],
};

const emit = defineEmits(['confirm', 'cancel']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const acForm: any = ref(null);

const state = reactive({
    form: { ...DefaultForm },
    btnLoading: false,
    publicAuthCerts: [] as any,
});

const showResourceEdit = computed(() => {
    return state.form.type != AuthCertTypeEnum.Public.value && !props.resourceEdit;
});

watch(dialogVisible, (val: any) => {
    if (val) {
        setForm(props.authCert);
    } else {
        cancelEdit();
    }
});

const setForm = (val: any) => {
    val = { ...val };
    if (!val.extra) {
        val.extra = {};
    }
    state.form = val;
    if (state.form.ciphertextType == AuthCertCiphertextTypeEnum.Public.value) {
        getPublicAuthCerts();
    }
};

const { form, btnLoading } = toRefs(state);

const changeType = (val: any) => {
    // 如果选择了公共凭证，则需要保证密文类型不能为公共凭证
    if (val == AuthCertTypeEnum.Public.value && state.form.ciphertextType == AuthCertCiphertextTypeEnum.Public.value) {
        state.form.ciphertextType = AuthCertCiphertextTypeEnum.Password.value;
    }
};

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
        acForm.value?.resetFields();
        emit('cancel');
    }, 300);
};

const btnOk = async () => {
    acForm.value.validate(async (valid: boolean) => {
        if (valid) {
            emit('confirm', { ...state.form });
        }
    });
};
</script>
<style lang="scss"></style>
