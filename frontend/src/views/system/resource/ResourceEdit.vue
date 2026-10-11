<template>
    <AutoFormDialog
        v-model:visible="visible"
        :title="title"
        :items="formItems"
        :data="editData"
        :cols="2"
        label-width="100px"
        :confirm-api="onConfirm"
        @opened="onOpened"
        @submitted="emit('val-change')"
        @cancel="emit('cancel')"
        width="800px"
    >
        <!-- icon 选择器自定义插槽 -->
        <template #iconSelector="{ form }">
            <icon-selector v-model="form.meta.icon" />
        </template>
    </AutoFormDialog>
</template>

<script lang="ts" setup>
import { notEmpty } from '@/common/assert';
import { LinkTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';
import { AutoFormDialog, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import iconSelector from '@/components/icon-selector/index.vue';
import { Msg } from '@/hooks/useI18n';
import { computed, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { resourceApi } from '../api';
import { ResourceTypeEnum } from '../enums';
import type { ResourceMeta, SysResource } from '../types';

const { t } = useI18n();

const props = defineProps({
    data: Object as PropType<Partial<SysResource> | boolean | null>,
    title: String,
    typeDisabled: Boolean,
});

const visible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['cancel', 'val-change']);

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

// 选项 label 传 i18n key：AutoForm 内部对 option.label 做 $t()
const trueFalseOption = [
    { label: 'system.menu.yes', value: true },
    { label: 'system.menu.no', value: false },
];

/** 表单声明（渲染 + 校验唯一数据源；菜单类型才有 meta 相关字段） */
const formItems = defineFormItems<ResourceForm>([
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
]);

interface ResourceForm {
    id?: number | null;
    name?: string | null;
    pid?: number | null;
    code?: string | null;
    type?: number | null;
    meta: ResourceMeta;
}

/** 将后端返回的 meta 归一为对象（可能是 ''、JSON 字符串、或已解析对象） */
const normalizeMeta = (raw: unknown): ResourceMeta => {
    if (raw && typeof raw === 'object') return { ...defaultMeta, ...(raw as Partial<ResourceMeta>) };
    if (typeof raw === 'string' && raw !== '') {
        try {
            return { ...defaultMeta, ...JSON.parse(raw) };
        } catch {
            /* ignore */
        }
    }
    return { ...defaultMeta };
};

/** 传给 AutoFormDialog 的回填数据：在克隆前完成 meta 归一化，避免 AutoFormDialog 内部 cloneFormData 拿到未归一的原始值 */
const editData = computed<Partial<ResourceForm> | boolean>(() => {
    if (!props.data || typeof props.data !== 'object') {
        return props.data ?? false;
    }
    return { ...props.data, meta: normalizeMeta(props.data.meta) };
});

// 宿主内部表单在 @opened 接管
const { onOpened, requireForm } = useAutoFormModel<ResourceForm>();

const { execute: saveResourceExec } = resourceApi.save.useApi();

/** 提交时解析 menu meta（只保留非空/真值字段） */
const parseMenuMeta = (meta: ResourceMeta): Partial<ResourceMeta> => {
    const metaForm: Partial<ResourceMeta> = {};
    notEmpty(meta.routeName, t('system.menu.routeNameNotEmpty'));
    metaForm.routeName = meta.routeName;
    if (meta.isKeepAlive) metaForm.isKeepAlive = true;
    if (meta.isHide) metaForm.isHide = true;
    if (meta.isAffix) metaForm.isAffix = true;
    if (meta.linkType) metaForm.linkType = meta.linkType;
    if (meta.link) metaForm.link = meta.link;
    if (meta.redirect) metaForm.redirect = meta.redirect;
    if (meta.component) metaForm.component = meta.component;
    if (meta.icon) metaForm.icon = meta.icon;
    return metaForm;
};

// confirmApi 提交动作；成功提示与关闭弹窗由组件内置逻辑处理
const onConfirm = async () => {
    const form = requireForm();
    let meta: Partial<ResourceMeta> | undefined;
    if (form.type == menuTypeValue) {
        meta = parseMenuMeta(form.meta);
    }
    await saveResourceExec({ ...form, meta });
    Msg.saveSuccess();
};
</script>
<style lang="scss"></style>
