<template>
    <div class="mock-data-dialog">
        <auto-form-drawer
            v-model:visible="visible"
            :title="title"
            :items="items"
            :data="editData"
            size="50%"
            :cols="3"
            :confirm-api="cronJobApi.save.request"
            @submitted="emit('submitSuccess')"
            @cancel="emit('cancel')"
        >
            <!-- cron 表达式（自定义控件插槽） -->
            <template #cron="{ form: f }">
                <CrontabInput v-model="f.cron" />
            </template>
            <!-- 通知模板（自定义控件插槽：按编码选择消息模板，不允许手填） -->
            <template #notifyTmplCode="{ form: f }">
                <MsgTmplSelect v-model="f.notifyTmplCode" value-field="code" clearable />
            </template>
            <!-- 关联机器标签（自定义控件插槽） -->
            <template #codePaths="{ form: f }">
                <TagTreeCheck height-mode="fixed" height="200px" :tag-type="`${TagResourceTypeEnum.Machine.value}`" v-model="f.codePaths" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import { computed, type PropType } from 'vue';
import MsgTmplSelect from '@/views/msg/components/MsgTmplSelect.vue';
import TagTreeCheck from '../../component/TagTreeCheck.vue';
import { cronJobApi } from '../api';
import { CronJobNotifyTypeEnum, CronJobSaveExecResTypeEnum, CronJobStatusEnum } from '../enums';
import type { MachineCronJob, MachineCronJobForm } from '../types';
import type { ResourceTag } from '@/types/common';

const props = defineProps({
    data: {
        type: Object as PropType<MachineCronJob | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const emit = defineEmits(['cancel', 'submitSuccess']);

const visible = defineModel<boolean>('visible', { default: false });

/**
 * 表单声明（cron/notifyTmplCode/codePaths 为自定义控件插槽，其余由 AutoForm 按字段类型渲染）
 *
 * 短字段按 3 列排布（cols=3 即每项默认 8 栅格），长内容字段单独给跨度：
 * cron 需容纳表达式与下次执行时间提示，备注/脚本/关联机器整行铺开。
 */
const items = defineFormItems<MachineCronJobForm>([
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'cron', label: 'machine.cronExpression', required: true, slot: 'cron', span: 16 },
    { prop: 'status', label: 'common.status', type: 'enum', enums: CronJobStatusEnum, required: true },
    { prop: 'saveExecResType', label: 'machine.execResRecordType', type: 'enum', enums: CronJobSaveExecResTypeEnum, required: true },
    { prop: 'timeoutSeconds', label: 'machine.cronJobTimeout', type: 'number', tooltip: 'machine.cronJobTimeoutTips' },
    { prop: 'retryTimes', label: 'machine.cronJobRetryTimes', type: 'number', tooltip: 'machine.cronJobRetryTips' },
    { prop: 'notifyType', label: 'machine.cronJobNotifyType', type: 'enum', enums: CronJobNotifyTypeEnum },
    {
        prop: 'notifyTmplCode',
        label: 'machine.cronJobNotifyTmpl',
        // 声明为 select 使必填提示为「请选择」，实际控件由同名插槽的消息模板选择器承载
        type: 'select',
        tooltip: 'machine.cronJobNotifyTmplTips',
        // 选了「不通知」就没有可绑定的模板；要通知则必须有模板，否则后端静默不发（配了通知却收不到）
        when: (f) => f.notifyType !== CronJobNotifyTypeEnum.None.value,
        required: (f) => f.notifyType !== CronJobNotifyTypeEnum.None.value,
    },
    { prop: 'remark', label: 'common.remark', span: 24 },
    { prop: 'script', label: 'machine.script', type: 'monaco', required: true, span: 24, props: { language: 'shell', height: '200px' } },
    { prop: 'codePaths', label: 'machine.relateMachine', slot: 'codePaths', span: 24 },
]);

/**
 * 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成）
 *
 * 声明为宽松形：新建态仅预置 script/status，其余必填项由用户输入或字段 defaultValue 提供。
 */
const editData = computed<Partial<MachineCronJobForm>>(() => {
    if (props.data) {
        return {
            ...props.data,
            codePaths: props.data.tags?.map((tag: ResourceTag) => tag.codePath),
        };
    }
    return { script: '', status: 1, timeoutSeconds: 0, retryTimes: 0, notifyType: 0 };
});

// 统一提交：confirmApi 由 AutoFormDrawer 内置逻辑驱动（校验 → 保存 → 成功提示 → submitted → 关闭抽屉，全程 loading 防重复提交）
</script>
<style lang="scss"></style>
