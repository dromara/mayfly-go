<template>
    <div>
        <el-dialog :title="title" :destroy-on-close="true" v-model="visible" width="800px">
            <auto-form ref="menuFormRef" v-model="form" :items="items" :cols="2" label-width="auto">
                <template #iconSelector>
                    <icon-selector v-model="form.meta.icon" />
                </template>
            </auto-form>

            <template #footer>
                <el-button @click="onCancel()">{{ $t('common.cancel') }}</el-button>
                <el-button type="primary" :loading="saveBtnLoading" @click="onConfirm">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { notEmpty } from '@/common/assert';
import { LinkTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import iconSelector from '@/components/icon-selector/index.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { resourceApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import type { ResourceMeta } from '../types';

const { t } = useI18n();

const props = defineProps({
    data: {
        type: [Boolean, Object],
    },
    title: {
        type: String,
    },
    typeDisabled: {
        type: Boolean,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

interface ResourceForm {
    id?: number | null;
    name?: string | null;
    pid?: number | null;
    code?: string | null;
    type?: number | null;
    meta: ResourceMeta;
    [key: string]: unknown;
}

const menuFormRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown; resetFields: () => void }>('menuFormRef');

const menuTypeValue = ResourceTypeEnum.Menu.value;

const defaultMeta: ResourceMeta = {
    routeName: '',
    icon: 'Menu',
    redirect: '',
    component: '',
    isKeepAlive: true,
    isHide: false,
    isAffix: false,
    linkType: 0,
    link: '',
};

// 选项 label 传 i18n key：AutoForm 内部对 option.label 做 $t()，此处预先 t() 会被二次翻译（控制台报缺 key，且切换语言后文案不随之更新）
const trueFalseOption = [
    {
        label: 'system.menu.yes',
        value: true,
    },
    {
        label: 'system.menu.no',
        value: false,
    },
];

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；菜单类型才有 meta 相关字段） */
const items: AutoFormItem[] = [
    { prop: 'type', label: 'common.type', type: 'enum', enums: ResourceTypeEnum, required: true, disabled: () => props.typeDisabled },
    { prop: 'name', label: 'common.name', required: true },
    {
        prop: 'code',
        label: 'path|code',
        tooltip: 'system.menu.menuCodeTips',
        placeholder: 'system.menu.menuCodePlaceholder',
        rules: [Rules.requiredInput('code')],
    },
    { prop: 'meta.icon', label: 'system.menu.icon', type: 'custom', slot: 'iconSelector', when: (f) => f.type === menuTypeValue },
    { prop: 'meta.routeName', label: 'system.menu.routerName', tooltip: 'system.menu.routerNameTips', when: (f) => f.type === menuTypeValue },
    {
        prop: 'meta.isKeepAlive',
        label: 'system.menu.isCache',
        type: 'select',
        tooltip: 'system.menu.isCacheTips',
        options: trueFalseOption,
        when: (f) => f.type === menuTypeValue,
    },
    {
        prop: 'meta.isHide',
        label: 'system.menu.isHide',
        type: 'select',
        tooltip: 'system.menu.isHideTips',
        options: trueFalseOption,
        when: (f) => f.type === menuTypeValue,
    },
    {
        prop: 'meta.isAffix',
        label: 'system.menu.tagIsDelete',
        type: 'select',
        options: trueFalseOption,
        when: (f) => f.type === menuTypeValue,
    },
    {
        prop: 'meta.linkType',
        label: 'system.menu.externalLink',
        type: 'select',
        tooltip: 'system.menu.externalLinkTips',
        options: [
            { label: 'system.menu.no', value: 0 },
            { label: 'system.menu.inline', value: LinkTypeEnum.Iframes.value },
            { label: 'system.menu.externalLink', value: LinkTypeEnum.Link.value },
        ],
        when: (f) => f.type === menuTypeValue,
    },
    {
        prop: 'meta.link',
        label: 'system.menu.linkAddress',
        placeholder: 'system.menu.linkPlaceholder',
        when: (f) => f.type === menuTypeValue && f.meta.linkType > 0,
    },
];

const state = reactive<{
    form: ResourceForm;
}>({
    form: {
        id: null,
        name: null,
        pid: null,
        code: null,
        type: null,
        meta: {
            routeName: '',
            icon: '',
            redirect: '',
            isKeepAlive: true,
            isHide: false,
            isAffix: false,
            linkType: 0,
            link: '',
        },
    },
});

const { form } = toRefs(state);

const { isFetching: saveBtnLoading, execute: saveResouceExec } = resourceApi.save.useApi();

watch(visible, () => {
    if (!visible.value) {
        return;
    }
    if (props.data) {
        const data = props.data as ResourceForm;
        state.form = { ...data, meta: data.meta ?? { ...defaultMeta } };
    } else {
        state.form = { meta: { ...defaultMeta } };
    }

    // 不存在或false，都为false
    const meta = state.form.meta;
    state.form.meta.isKeepAlive = meta.isKeepAlive ? true : false;
    state.form.meta.isHide = meta.isHide ? true : false;
    state.form.meta.isAffix = meta.isAffix ? true : false;
    state.form.meta.linkType = meta.linkType;
});

const onConfirm = async () => {
    await useI18nFormValidate(menuFormRef);

    let meta: Partial<ResourceMeta> | undefined;
    if (state.form.type == 1) {
        // 如果是菜单，则解析meta，如果值为false或者''则去除该值
        meta = parseMenuMeta(state.form.meta);
    }
    const submitForm = { ...state.form, meta };

    await saveResouceExec(submitForm);

    emit('val-change', submitForm);
    Msg.saveSuccess();
    onCancel();
};

const parseMenuMeta = (meta: ResourceMeta): Partial<ResourceMeta> => {
    const metaForm: Partial<ResourceMeta> = {};
    // 如果是菜单，则校验meta
    notEmpty(meta.routeName, t('system.menu.routeNameNotEmpty'));
    metaForm.routeName = meta.routeName;
    if (meta.isKeepAlive) {
        metaForm.isKeepAlive = true;
    }
    if (meta.isHide) {
        metaForm.isHide = true;
    }
    if (meta.isAffix) {
        metaForm.isAffix = true;
    }
    if (meta.linkType) {
        metaForm.linkType = meta.linkType;
    }
    if (meta.link) {
        metaForm.link = meta.link;
    }
    if (meta.redirect) {
        metaForm.redirect = meta.redirect;
    }
    if (meta.component) {
        metaForm.component = meta.component;
    }
    if (meta.icon) {
        metaForm.icon = meta.icon;
    }
    return metaForm;
};

const onCancel = () => {
    visible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
