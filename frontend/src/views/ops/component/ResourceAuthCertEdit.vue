<template>
    <div class="auth-cert-edit">
        <el-dialog :title="props.title" v-model="dialogVisible" :show-close="false" width="600px" :destroy-on-close="true" :close-on-click-modal="false">
            <auto-form ref="acFormRef" v-model="form" :items="items">
                <!-- 密文输入（密码 / 私钥两种形态，含查看密文入口） -->
                <template #ciphertext>
                    <el-input
                        v-if="form.ciphertextType == AuthCertCiphertextTypeEnum.Password.value"
                        type="password"
                        show-password
                        clearable
                        v-model.trim="form.ciphertext"
                        autocomplete="new-password"
                    >
                        <template #suffix>
                            <SvgIcon v-if="form.id" v-auth="'authcert:showciphertext'" @click="getCiphertext" name="search" />
                        </template>
                    </el-input>
                    <div v-else class="w-full!" style="position: relative">
                        <SvgIcon
                            v-if="form.id"
                            v-auth="'authcert:showciphertext'"
                            @click="getCiphertext"
                            name="search"
                            style="position: absolute; top: 5px; right: 5px; cursor: pointer; z-index: 1"
                        />
                        <el-input type="textarea" :rows="5" v-model="form.ciphertext" :placeholder="$t('ac.privateKeyPlaceholder')"> </el-input>
                    </div>
                </template>

                <!-- 公共凭证选择（自定义 option 内容） -->
                <template #publicAuthCert>
                    <el-select default-first-option filterable v-model="form.ciphertext" @change="changePublicAuthCert" class="w-full">
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
                </template>
            </auto-form>
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
import { reactive, toRefs, computed, watch, useTemplateRef } from 'vue';
import { AuthCertTypeEnum, AuthCertCiphertextTypeEnum } from '../tag/enums';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import EnumTag from '@/components/enum-tag/EnumTag.vue';
import { resourceAuthCertApi } from '../tag/api';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { useI18nFormValidate } from '@/hooks/useI18n';
import type { ResourceAuthCert } from '@/types/common';

/** 表单类型，extra 必填（初始化时保证存在） */
interface AuthCertForm extends Omit<ResourceAuthCert, 'extra'> {
    extra: Record<string, unknown>;
}

const props = defineProps({
    title: {
        type: String,
        default: '',
    },
    authCert: {
        type: Object as () => ResourceAuthCert | null,
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

const DefaultForm: AuthCertForm = {
    id: null,
    name: '',
    username: '',
    ciphertextType: AuthCertCiphertextTypeEnum.Password.value,
    type: AuthCertTypeEnum.Private.value,
    resourceType: TagResourceTypeEnum.PublicAuthCert.value,
    resourceCode: '',
    ciphertext: '',
    extra: {} as Record<string, unknown>,
    remark: '',
};

const emit = defineEmits(['confirm', 'cancel']);

const dialogVisible = defineModel<boolean>('visible', { default: false });

const acFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown; resetFields: () => void }>('acFormRef');

const state = reactive({
    form: { ...DefaultForm } as AuthCertForm,
    btnLoading: false,
    publicAuthCerts: [] as ResourceAuthCert[],
});

const showResourceEdit = computed(() => {
    return state.form.type != AuthCertTypeEnum.Public.value && !props.resourceEdit;
});

watch(dialogVisible, (val: boolean) => {
    if (val) {
        setForm(props.authCert ?? undefined);
    } else {
        cancelEdit();
    }
});

const setForm = (val: ResourceAuthCert | undefined) => {
    const formData = { ...val } as AuthCertForm;
    if (!formData.extra) {
        formData.extra = {};
    }
    state.form = formData;
    if (state.form.ciphertextType == AuthCertCiphertextTypeEnum.Public.value) {
        getPublicAuthCerts();
    }
};

const { form, btnLoading } = toRefs(state);

const changeType = (val: number) => {
    // 如果选择了公共凭证，则需要保证密文类型不能为公共凭证
    if (val == AuthCertTypeEnum.Public.value && state.form.ciphertextType == AuthCertCiphertextTypeEnum.Public.value) {
        state.form.ciphertextType = AuthCertCiphertextTypeEnum.Password.value;
    }
};

const changeCiphertextType = (val: number) => {
    if (val == AuthCertCiphertextTypeEnum.Public.value) {
        getPublicAuthCerts();
    }
};

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；下拉直接声明枚举，ciphertext/publicAuthCert 走插槽承载复杂控件） */
const items: AutoFormItem[] = [
    {
        prop: 'type',
        label: 'ac.credentialType',
        type: 'enum',
        enums: AuthCertTypeEnum,
        required: true,
        // 按 disableType 剔除不可选凭证类型
        excludeValues: props.disableType,
        onChange: (val: unknown) => changeType(val as number),
    },
    {
        prop: 'ciphertextType',
        label: 'ac.ciphertextType',
        type: 'enum',
        enums: AuthCertCiphertextTypeEnum,
        required: true,
        // 按 disableCiphertextType 剔除；公共凭证选项在凭证类型为公共时禁用（选项级联动禁用）
        excludeValues: props.disableCiphertextType,
        optionDisabled: (val, f) => val == AuthCertCiphertextTypeEnum.Public.value && f.type == AuthCertTypeEnum.Public.value,
        onChange: (val: unknown) => changeCiphertextType(val as number),
    },
    {
        prop: 'resourceType',
        label: 'ac.resourceType',
        type: 'enum',
        // 仅允许资源型枚举子集
        enums: [TagResourceTypeEnum.Machine, TagResourceTypeEnum.DbInstance, TagResourceTypeEnum.Redis],
        required: true,
        disabled: (f) => !!f.id,
        when: () => showResourceEdit.value,
    },
    { prop: 'resourceCode', label: 'ac.resourceCode', required: true, disabled: (f) => !!f.id, when: () => showResourceEdit.value },
    {
        prop: 'name',
        label: 'common.name',
        required: true,
        rules: Rules.resourceCode,
        disabled: (f) => !!f.id,
        placeholder: 'ac.namePlaceholder',
        // 仅公共凭证需要命名
        when: (f) => f.type == AuthCertTypeEnum.Public.value,
    },
    { prop: 'username', label: 'common.username', when: (f) => f.ciphertextType != AuthCertCiphertextTypeEnum.Public.value },
    { prop: 'ciphertext', label: 'common.password', type: 'custom', when: (f) => f.ciphertextType == AuthCertCiphertextTypeEnum.Password.value },
    { prop: 'ciphertext', label: 'ac.privateKey', type: 'custom', when: (f) => f.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value },
    { prop: 'extra.passphrase', label: 'ac.privateKeyPwd', type: 'password', when: (f) => f.ciphertextType == AuthCertCiphertextTypeEnum.PrivateKey.value },
    { prop: 'ciphertext', label: 'ac.publicAc', type: 'custom', slot: 'publicAuthCert', when: (f) => f.ciphertextType == AuthCertCiphertextTypeEnum.Public.value },
    { prop: 'remark', label: 'common.remark', type: 'textarea', rows: 2 },
];

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
    state.form.extra.passphrase = res.extra?.passphrase ?? '';
};

const cancelEdit = () => {
    dialogVisible.value = false;

    setTimeout(() => {
        state.form = { ...DefaultForm };
        acFormRef.value?.resetFields();
        emit('cancel');
    }, 300);
};

const btnOk = async () => {
    await useI18nFormValidate(acFormRef);
    emit('confirm', { ...state.form });
};
</script>
<style lang="scss"></style>
