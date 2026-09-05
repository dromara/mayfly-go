<template>
    <div>
        <auto-form-drawer ref="drawerRef" v-model:visible="visible" :title="title" :items="items" :data="editData" size="40%" :confirm-loading="saveBtnLoading" @confirm="btnOk" @cancel="emit('cancel')">
            <!-- 渠道类型扩展配置（动态组件） -->
            <template #extra="{ form: f }">
                <component v-if="channelTypeComp(f.type)" :is="channelTypeComp(f.type)" v-model:extra="f.extra" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import EnumValue from '@/common/Enum';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg, useI18nFormValidate } from '@/hooks/useI18n';
import { computed, useTemplateRef, type Component, type PropType } from 'vue';
import { channelApi } from '../api';
import { ChannelStatusEnum, ChannelTypeEnum } from '../enums';
import ChannelDing from './ChannelDing.vue';
import ChannelEmail from './ChannelEmail.vue';
import type { MsgChannel } from '@/views/system/msg/types';

/** 消息通道编辑表单类型 */
interface ChannelForm extends Omit<Partial<MsgChannel>, 'id' | 'name' | 'type' | 'status'> {
    id?: number | null;
    name?: string | null;
    type?: string | null;
    status?: string | number;
}

const props = defineProps({
    form: {
        type: Object as PropType<MsgChannel | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const channels: Record<string, Component> = {
    ChannelEmail,
    ChannelDing,
};

//定义事件
const emit = defineEmits(['cancel', 'success']);

const visible = defineModel<boolean>('visible', { default: false });

const drawerRef = useTemplateRef<{ validate: (...args: unknown[]) => unknown }>('drawerRef');

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；extra 走插槽承载渠道类型扩展配置） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'msg.name', required: true },
    { prop: 'status', label: 'common.status', type: 'enum', enums: ChannelStatusEnum },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
    {
        prop: 'type',
        label: 'common.type',
        type: 'enum',
        enums: ChannelTypeEnum,
        required: true,
        // 切换渠道类型时重置扩展配置
        onChange: (_val: unknown, form: AutoFormData) => {
            (form as ChannelForm).extra = {};
        },
    },
    { prop: 'url', label: 'URL', required: true },
    { prop: 'extra', type: 'custom' },
];

const defaultForm = (): ChannelForm => {
    return {
        id: null,
        name: null,
        type: null,
        url: '',
        status: ChannelStatusEnum.Enable.value,
        remark: '',
        extra: {},
    };
};

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    const form = props.form as ChannelForm | null;
    return (form ? { ...form } : defaultForm()) as unknown as AutoFormData;
});

const { isFetching: saveBtnLoading, execute: saveFormExec } = channelApi.save.useApi();

/** 渠道类型对应的扩展配置组件 */
const channelTypeComp = (type?: string | null): Component | undefined => {
    return channels[EnumValue.getEnumByValue(ChannelTypeEnum, type ?? '')?.extra?.component];
};

const btnOk = async (rawForm: AutoFormData) => {
    await useI18nFormValidate(drawerRef);
    await saveFormExec(rawForm);
    Msg.saveSuccess();
    emit('success', rawForm);
    visible.value = false;
};
</script>
<style lang="scss"></style>
