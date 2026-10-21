<template>
    <auto-form-drawer
        v-model:visible="visible"
        :title="title"
        :items="items"
        :data="editData"
        size="55%"
        :cols="3"
        :confirm-api="agentTaskApi.save.request"
        @submitted="emit('submitSuccess')"
        @cancel="emit('cancel')"
    >
        <!-- cron 表达式（自定义控件插槽：留空则不自动调度） -->
        <template #cron="{ form: f }">
            <CrontabInput v-model="f.cron" />
        </template>
        <!-- 通知模板（自定义控件插槽：按编码选择消息模板，不允许手填） -->
        <template #notifyTmplCode="{ form: f }">
            <MsgTmplSelect v-model="f.notifyTmplCode" value-field="code" clearable />
        </template>
        <!-- 任务指令（自定义控件插槽）：与 AI 助手同一个提示词编辑器，
             `/` 引用技能、`@` 引用资源；引用段随指令一并保存，派发时展开成可直接取参的定位信息。
             Enter 是换行而不是提交（表单里回车即保存会误发） -->
        <template #instruction="{ form: f }">
            <PromptEditor
                v-model:text="f.instruction"
                :segments="f.instructionSegments"
                :placeholder="t('agentops.instructionPlaceholder')"
                @update:segments="(segs) => (f.instructionSegments = segs)"
            />
        </template>
    </auto-form-drawer>
</template>

<script lang="ts" setup>
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import CrontabInput from '@/components/crontab/CrontabInput.vue';
import MsgTmplSelect from '@/views/msg/components/MsgTmplSelect.vue';
import PromptEditor from '@/views/ai/input/PromptEditor.vue';
import { profileFormOptions } from './useAgentProfiles';
import { computed, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { agentTaskApi } from './api';
import { AgentTaskNotifyTypeEnum, AgentTaskStatusEnum, MisfirePolicyEnum, ProfileEnum } from './enums';
import type { AgentTask, AgentTaskForm } from './types';

const props = defineProps({
    data: {
        type: Object as PropType<AgentTask | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const { t } = useI18n();

const emit = defineEmits(['cancel', 'submitSuccess']);

const visible = defineModel<boolean>('visible', { default: false });

/**
 * 表单声明：分「基础配置」「调度与通知」两组（group 类型），
 * cron / notifyTmplCode 为自定义控件插槽，其余由 AutoForm 按字段类型渲染。
 *
 * 档位选项来自后端注册表（useAgentProfiles）：需一次性运行许可的档（受控执行档）置灰——
 * 定时/手动派发不会下发许可，那个组合只会跑出「成功但什么都没做」的空诊断（后端保存期也会拒）。
 * 置灰而不是隐藏：能力确实存在，只是不适用于无人值守场景。
 */
const items = defineFormItems<AgentTaskForm>([
    { type: 'group', label: 'agentops.basicGroup' },
    { prop: 'name', label: 'common.name', required: true },
    { prop: 'profileId', label: 'agentops.profile', type: 'select', options: profileFormOptions, required: true },
    { prop: 'status', label: 'common.status', type: 'enum', enums: AgentTaskStatusEnum, required: true },
    // 指令不再是裸 textarea：控件由同名插槽的 PromptEditor 承载（引用技能/资源）
    { prop: 'instruction', label: 'agentops.instruction', slot: 'instruction', required: true, span: 24, tooltip: 'agentops.instructionTip' },

    { type: 'group', label: 'agentops.scheduleGroup', groupDescription: 'agentops.ownerTip' },
    { prop: 'cron', label: 'agentops.cron', slot: 'cron', span: 16, tooltip: 'agentops.cronTip' },
    { prop: 'misfirePolicy', label: 'agentops.misfirePolicy', type: 'enum', enums: MisfirePolicyEnum, tooltip: 'agentops.misfireTip' },
    { prop: 'timeoutSeconds', label: 'agentops.timeout', type: 'number', min: 0 },
    { prop: 'maxStep', label: 'agentops.maxStep', type: 'number', min: 0, tooltip: 'agentops.maxStepTip' },
    { prop: 'tokenBudget', label: 'agentops.tokenBudget', type: 'number', min: 0, tooltip: 'agentops.tokenBudgetTip' },
    { prop: 'dryRun', label: 'agentops.dryRun', type: 'switch', tooltip: 'agentops.dryRunTip' },
    { prop: 'retryTimes', label: 'agentops.retryTimes', type: 'number', min: 0, tooltip: 'agentops.retryTimesTip' },
    { prop: 'notifyType', label: 'agentops.notifyType', type: 'enum', enums: AgentTaskNotifyTypeEnum },
    {
        prop: 'notifyTmplCode',
        label: 'agentops.notifyTmpl',
        // 声明为 select 使必填提示为「请选择」，实际控件由同名插槽的消息模板选择器承载
        type: 'select',
        tooltip: 'agentops.notifyTmplTip',
        // 选了「不通知」就没有可绑定的模板；要通知则必须有模板，否则后端静默不发
        when: (f) => f.notifyType !== AgentTaskNotifyTypeEnum.None.value,
        required: (f) => f.notifyType !== AgentTaskNotifyTypeEnum.None.value,
    },
    { prop: 'remark', label: 'common.remark', span: 24 },
]);

/**
 * 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成）。
 * 新建态预置默认档位/状态/通知，ownerAccountId 留空由后端以当前登录账号为运行身份（谁建谁负责）。
 */
const editData = computed<Partial<AgentTaskForm>>(() => {
    if (props.data) {
        return { ...props.data };
    }
    return {
        profileId: ProfileEnum.Diagnostician.value as string,
        status: AgentTaskStatusEnum.Enable.value,
        timeoutSeconds: 0,
        maxStep: 0,
        retryTimes: 0,
        notifyType: 0,
        misfirePolicy: 0,
    };
});
</script>
