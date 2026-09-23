<template>
    <div>
        <auto-form-drawer v-model:visible="dialogVisible" :title="title" :items="items" :data="editData" size="40%" :confirm-api="onConfirm" @opened="onOpened" @submitted="emit('cancel')" @cancel="emit('cancel')">
            <!-- 关联标签 -->
            <template #tagCodePaths="{ form }">
                <TagTreeSelect multiple :code="form.code" v-model="form.tagCodePaths" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { computed, type PropType } from 'vue';
import TagTreeSelect from '../component/TagTreeSelect.vue';
import { dockerApi } from './api';
import type { Container, ContainerConfForm } from './types';

const props = defineProps({
    data: {
        type: Object as PropType<Container | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const dialogVisible = defineModel<boolean>('visible', { default: false });

const emit = defineEmits(['val-change', 'cancel']);

/** 表单声明（defineFormItems<ContainerConfForm>，渲染 + 校验唯一数据源；关联标签走 custom 插槽） */
const items = defineFormItems<ContainerConfForm>([
    { prop: 'tagCodePaths', label: 'tag.relateTag', type: 'custom', rules: [Rules.requiredSelect('tag.relateTag')] },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'addr', label: 'docker.addr', type: 'textarea', placeholder: 'docker.addrTips', required: true },
    { prop: 'remark', label: 'common.remark', type: 'textarea' },
]);

/** 新建态默认值（编辑态行数据不携带 tagCodePaths，由插槽控件按 code 自行 hydrate） */
const defaultForm: ContainerConfForm = { id: null, code: '', tagCodePaths: [], name: null, addr: '', remark: '' };

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<ContainerConfForm>(() => {
    return props.data ? { ...defaultForm, ...props.data, tagCodePaths: [] } : { ...defaultForm };
});

// 宿主内部表单在 @opened 接管（提交与回传均基于它）
const { onOpened, requireForm } = useAutoFormModel<ContainerConfForm>();

const { execute: saveConfExec } = dockerApi.saveConf.useApi();

// confirmApi 提交动作；成功提示、关闭抽屉由组件内置逻辑处理，submitted 时通知父组件
const onConfirm = async () => {
    const form = requireForm();
    await saveConfExec(form);
    emit('val-change', form);
};
</script>
<style lang="scss"></style>
