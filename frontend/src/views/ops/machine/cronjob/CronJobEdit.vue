<template>
    <div class="mock-data-dialog">
        <auto-form-drawer v-model:visible="visible" :title="title" :items="items" :data="editData" size="50%" :confirm-api="cronJobApi.save.request" @submitted="emit('submitSuccess')" @cancel="emit('cancel')">
            <!-- cron 表达式（自定义控件插槽） -->
            <template #cron="{ form: f }">
                <CrontabInput v-model="f.cron" />
            </template>
            <!-- 关联机器标签（自定义控件插槽） -->
            <template #codePaths="{ form: f }">
                <TagTreeCheck height="200px" :tag-type="`${TagResourceTypeEnum.Machine.value}`" v-model="f.codePaths" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { AutoFormDrawer, type AutoFormData, type AutoFormItem } from '@/components/auto-form';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import { computed } from 'vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { cronJobApi } from '../api';
import { CronJobSaveExecResTypeEnum, CronJobStatusEnum } from '../enums';
import type { MachineCronJob, MachineCronJobForm } from '../types';
import type { ResourceTag } from '@/types/common';

const props = defineProps({
    data: {
        type: Object as () => MachineCronJob | null,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['cancel', 'submitSuccess']);

const visible = defineModel<boolean>('visible', { default: false });

/** 表单声明（AutoFormItem[]，渲染 + 校验唯一数据源；cron/codePaths 为自定义控件插槽） */
const items: AutoFormItem[] = [
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'cron', label: 'machine.cronExpression', required: true, slot: 'cron' },
    { prop: 'status', label: 'common.status', type: 'enum', enums: CronJobStatusEnum, required: true },
    { prop: 'saveExecResType', label: 'machine.execResRecordType', type: 'enum', enums: CronJobSaveExecResTypeEnum, required: true },
    { prop: 'remark', label: 'common.remark' },
    { prop: 'script', label: 'machine.script', type: 'monaco', required: true, props: { language: 'shell', height: '200px' } },
    { prop: 'codePaths', label: 'machine.relateMachine', slot: 'codePaths' },
];

/** 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成） */
const editData = computed<AutoFormData | null>(() => {
    if (props.data) {
        return {
            ...props.data,
            codePaths: props.data.tags?.map((tag: ResourceTag) => tag.codePath),
        } as unknown as AutoFormData;
    }
    return { script: '', status: 1 } as unknown as AutoFormData;
});

// 统一提交：confirmApi 由 AutoFormDrawer 内置逻辑驱动（校验 → 保存 → 成功提示 → submitted → 关闭抽屉，全程 loading 防重复提交）
</script>
<style lang="scss"></style>
