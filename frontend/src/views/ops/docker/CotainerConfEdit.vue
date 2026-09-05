<template>
    <div>
        <auto-form-drawer v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-loading="saveBtnLoading" @confirm="onConfirm" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import { Msg } from '@/hooks/useI18n';
import { computed, type PropType } from 'vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { dockerApi } from './api';
import type { Container } from './types';

const props = defineProps({
    container: {
        type: Object as PropType<Container | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['val-change', 'cancel']);

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；关联标签走 custom 插槽） */
const items: AutoFormItem[] = [
    { prop: 'tagCodePaths', label: 'tag.relateTag', type: 'custom', rules: [Rules.requiredSelect('tag.relateTag')] },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'addr', label: 'docker.addr', type: 'textarea', placeholder: 'docker.addrTips', required: true },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
];

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData>(() => {
    const container = props.container as Record<string, unknown> | null;
    if (container) {
        return { ...container } as AutoFormData;
    }
    return { id: null, code: '', tagCodePaths: [], name: null, addr: '', remark: '' } as AutoFormData;
});

const { isFetching: saveBtnLoading, execute: saveConfExec } = dockerApi.saveConf.useApi();

// @confirm 触发前 AutoFormDrawer 已完成表单校验
const onConfirm = async (form: AutoFormData) => {
    await saveConfExec(form);
    Msg.saveSuccess();
    emit('val-change', form);
    dialogVisible.value = false;
    emit('cancel');
};
</script>
<style lang="scss"></style>
