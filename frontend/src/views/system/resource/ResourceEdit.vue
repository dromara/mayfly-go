<template>
    <div>
        <el-dialog :title="title" :destroy-on-close="true" v-model="visible" width="800px">
            <el-form :model="form" :inline="true" ref="menuFormRef" :rules="rules" label-width="auto">
                <el-row :gutter="35">
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <el-form-item class="w-full!" prop="type" :label="$t('common.type')" required>
                            <enum-select :enums="ResourceTypeEnum" v-model="form.type" :disabled="typeDisabled" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <el-form-item class="w-full!" prop="name" :label="$t('common.name')" required>
                            <el-input v-model.trim="form.name" auto-complete="off"></el-input>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12">
                        <FormItemTooltip class="w-full!" label="path|code" prop="code" :tooltip="$t('system.menu.menuCodeTips')">
                            <el-input v-model.trim="form.code" :placeholder="$t('system.menu.menuCodePlaceholder')" auto-complete="off"></el-input>
                        </FormItemTooltip>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w-full!" :label="$t('system.menu.icon')">
                            <icon-selector v-model="form.meta.icon" />
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <FormItemTooltip
                            class="w-full!"
                            :label="$t('system.menu.routerName')"
                            prop="meta.routeName"
                            :tooltip="$t('system.menu.routerNameTips')"
                        >
                            <el-input v-model.trim="form.meta.routeName"></el-input>
                        </FormItemTooltip>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <FormItemTooltip class="w-full!" :label="$t('system.menu.isCache')" prop="meta.isKeepAlive" :tooltip="$t('system.menu.isCacheTips')">
                            <el-select v-model="form.meta.isKeepAlive" class="w-full!">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </FormItemTooltip>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <FormItemTooltip class="w-full!" :label="$t('system.menu.isHide')" prop="meta.isHide" :tooltip="$t('system.menu.isHideTips')">
                            <el-select v-model="form.meta.isHide" class="w-full!">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </FormItemTooltip>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <el-form-item class="w-full!" prop="meta.isAffix" :label="$t('system.menu.tagIsDelete')">
                            <el-select v-model="form.meta.isAffix" class="w-full!">
                                <el-option v-for="item in trueFalseOption" :key="item.value" :label="item.label" :value="item.value"> </el-option>
                            </el-select>
                        </el-form-item>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue">
                        <FormItemTooltip
                            class="w-full!"
                            :label="$t('system.menu.externalLink')"
                            prop="meta.linkType"
                            :tooltip="$t('system.menu.externalLinkTips')"
                        >
                            <el-select class="w-full!" @change="onChangeLinkType" v-model="form.meta.linkType">
                                <el-option :key="0" :label="$t('system.menu.no')" :value="0"> </el-option>
                                <el-option :key="1" :label="$t('system.menu.inline')" :value="LinkTypeEnum.Iframes.value"> </el-option>
                                <el-option :key="2" :label="$t('system.menu.externalLink')" :value="LinkTypeEnum.Link.value"> </el-option>
                            </el-select>
                        </FormItemTooltip>
                    </el-col>
                    <el-col :xs="24" :sm="12" :md="12" :lg="12" :xl="12" v-if="form.type === menuTypeValue && form.meta.linkType > 0">
                        <el-form-item prop="meta.link" :label="$t('system.menu.linkAddress')" class="w-full!">
                            <el-input v-model.trim="form.meta.link" :placeholder="$t('system.menu.linkPlaceholder')"></el-input>
                        </el-form-item>
                    </el-col>
                </el-row>
            </el-form>

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
import EnumSelect from '@/components/enum-select/EnumSelect.vue';
import FormItemTooltip from '@/components/form/FormItemTooltip.vue';
import iconSelector from '@/components/icon-selector/index.vue';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { reactive, toRefs, useTemplateRef, watch } from 'vue';
import type { FormInstance } from 'element-plus';
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

const menuFormRef = useTemplateRef<FormInstance>('menuFormRef');

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

const rules = {
    name: [Rules.requiredInput('common.name')],
    code: [Rules.requiredInput('code')],
};

const trueFalseOption = [
    {
        label: t('system.menu.yes'),
        value: true,
    },
    {
        label: t('system.menu.no'),
        value: false,
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

// 改变外链类型
const onChangeLinkType = (linkType: number) => {};

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
