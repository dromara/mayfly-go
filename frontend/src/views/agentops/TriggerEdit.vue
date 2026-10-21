<template>
    <auto-form-drawer
        v-model:visible="visible"
        :title="title"
        :items="items"
        :data="editData"
        size="55%"
        :cols="3"
        :confirm-api="agentTriggerApi.save.request"
        @submitted="emit('submitSuccess')"
        @cancel="emit('cancel')"
    >
        <!-- 内联指令：与任务编辑、AI 助手共用同一个提示词编辑器（引用技能/资源） -->
        <template #instruction="{ form: f }">
            <PromptEditor
                v-model:text="f.instruction"
                :segments="f.instructionSegments"
                :placeholder="t('agentops.instructionPlaceholder')"
                @update:segments="(segs) => (f.instructionSegments = segs)"
            />
        </template>
        <!-- 匹配条件：可视化四维度控件（类型/资源/严重级/指标），落库仍是后端契约的 JSON 字符串 -->
        <template #matchCond="{ form: f }">
            <MatchCondForm v-model="f.matchCond" />
        </template>
        <!-- 通知模板（自定义控件插槽：按编码选择消息模板，不允许手填）——与任务/计划任务表单同源 -->
        <template #notifyTmplCode="{ form: f }">
            <MsgTmplSelect v-model="f.notifyTmplCode" value-field="code" clearable />
        </template>
    </auto-form-drawer>
</template>

<script lang="ts" setup>
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import PromptEditor from '@/views/ai/input/PromptEditor.vue';
import MsgTmplSelect from '@/views/msg/components/MsgTmplSelect.vue';
import MatchCondForm from './MatchCondForm.vue';
import { computed, type PropType } from 'vue';
import { useI18n } from 'vue-i18n';
import { agentTaskApi, agentTriggerApi } from './api';
import { AgentTaskNotifyTypeEnum, AgentTaskStatusEnum, ProfileEnum, TriggerSourceEnum } from './enums';
import { profileFormOptions } from './useAgentProfiles';
import { triggerSourceFormOptions } from './useTriggerSources';
import type { AgentTrigger, AgentTriggerForm } from './types';

const props = defineProps({
    data: {
        type: Object as PropType<AgentTrigger | null>,
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
 * 关联任务下拉选项：加载全部任务（含禁用——触发器可复用其运行配置），value=任务id、label=任务名。
 * pageSize 取后端上限，任务量通常远小于此；异步选项在表单值变化时由 AutoForm 自动重载。
 *
 * 尾部补一条「已删除的任务」选项：删任务只把绑定它的触发器置为禁用、并不解绑（见后端
 * AgentTaskService.Delete），于是该 id 在任务列表里已查不到；el-select 无匹配 option 时会直接
 * 退化成显示裸值（用户看到一个不知是什么、也不敢改的数字）。补上带文案的选项至少说清
 * 「绑的任务已不存在」；真的拿它去保存会被后端「绑定任务不存在」拒回，不会静默存下一个悬空引用
 */
const loadTaskOptions = async () => {
    const res = await agentTaskApi.list.request({ pageNum: 1, pageSize: 2000 });
    const options = (res.list ?? []).map((task) => ({ value: task.id, label: task.name }));
    const boundTaskId = props.data?.taskId;
    if (boundTaskId && !options.some((option) => option.value === boundTaskId)) {
        options.push({ value: boundTaskId, label: t('agentops.bindTaskMissing', { id: boundTaskId }) });
    }
    return options;
};

/**
 * 表单声明：分「基础配置」「匹配条件」「运行配置」三组。
 *
 * 运行配置二选一（与后端 TriggerService.resolveRunConfig 口径一致）：
 *   - 选定「关联任务」→ 复用任务的 instruction/profile/owner/timeout/retry/notify 全套配置，内联指令/档位隐藏；
 *   - 不选任务 → 内联 instruction 必填、profileId 可选（后端缺省 diagnostician 只读诊断档）。
 * webhook 令牌不在本表单里：它只由服务端生成，更新时保留库中现值；需要换地址时勾选
 * 「轮换推送令牌」（仅编辑态可见），保存后旧地址立即失效，需在外部监控系统重新配置推送地址。
 */
const items = defineFormItems<AgentTriggerForm>([
    { type: 'group', label: 'agentops.triggerBasicGroup' },
    { prop: 'name', label: 'common.name', required: true },
    // 来源选项取后端注册表（与档位目录同一理由）：静态枚举会让新注册的来源根本无法被创建
    { prop: 'sourceType', label: 'agentops.sourceType', type: 'select', options: triggerSourceFormOptions, required: true, tooltip: 'agentops.sourceTypeTip' },
    { prop: 'enabled', label: 'common.status', type: 'enum', enums: AgentTaskStatusEnum, required: true },

    { type: 'group', label: 'agentops.matchGroup', groupDescription: 'agentops.matchGroupTip' },
    {
        prop: 'matchCond',
        label: 'agentops.matchCond',
        slot: 'matchCond',
        span: 24,
    },

    { type: 'group', label: 'agentops.runConfigGroup', groupDescription: 'agentops.runConfigGroupTip' },
    { prop: 'taskId', label: 'agentops.bindTask', type: 'select', options: loadTaskOptions, span: 24, tooltip: 'agentops.bindTaskTip' },
    {
        prop: 'instruction',
        label: 'agentops.instruction',
        slot: 'instruction',
        span: 24,
        when: (f) => !f.taskId,
        required: (f) => !f.taskId,
        tooltip: 'agentops.instructionTip',
    },
    // 档位选项与任务表单同源（目录驱动）：需一次性许可的档位置灰，事件派发同样不下发许可
    { prop: 'profileId', label: 'agentops.profile', type: 'select', options: profileFormOptions, when: (f) => !f.taskId },
    // 内联模式的结果通知：关联任务时以任务配置为准（后端 resolveRunConfig 会用任务值覆盖），
    // 不填则 webhook/告警驱动的诊断运行结果永远无处可达
    { prop: 'notifyType', label: 'agentops.notifyType', type: 'enum', enums: AgentTaskNotifyTypeEnum, when: (f) => !f.taskId },
    {
        prop: 'notifyTmplCode',
        label: 'agentops.notifyTmpl',
        // 声明为 select 使必填提示为「请选择」，实际控件由同名插槽的消息模板选择器承载（不手填编码）
        type: 'select',
        tooltip: 'agentops.notifyTmplTip',
        // 选了「不通知」就没有可绑定的模板；要通知则必须有模板，否则后端静默不发（显示即必填）。
        // 「未填」与「不通知」同义（后端 notifyType 零值即 None），故用 != null 而非 !== None：
        // 否则新建态下 notifyType 还是 undefined 就会被判成「要通知」，把模板变成必填，
        // 用户无法在不配通知的情况下保存一个触发器（任务表单靠下面的默认值避免同一问题）
        when: (f) => !f.taskId && f.notifyType != null && f.notifyType !== AgentTaskNotifyTypeEnum.None.value,
        required: true,
    },
    { prop: 'cooldownSeconds', label: 'agentops.cooldown', type: 'number', min: 0, tooltip: 'agentops.cooldownTip' },
    {
        prop: 'rotateWebhookToken',
        label: 'agentops.rotateWebhookToken',
        type: 'switch',
        // 仅编辑态的 webhook 触发器可轮换（新建时服务端直接生成，无需轮换）
        when: (f) => f.sourceType === TriggerSourceEnum.Webhook.value && !!f.id,
        tooltip: 'agentops.rotateWebhookTokenTip',
    },
    { prop: 'remark', label: 'common.remark', span: 24 },
]);

/**
 * 传给 AutoFormDrawer 的回填数据（深拷贝由组件内部完成）。
 * 新建态预置默认来源/状态/档位/冷却，ownerAccountId 留空由后端以当前登录账号为运行身份（谁建谁负责）。
 */
const editData = computed<Partial<AgentTriggerForm>>(() => {
    if (props.data) {
        // taskId=0 是「内联模式」哨兵（不是真实任务 id），回填时必须归一化成未选，
        // 否则关联任务下拉因无匹配 option 会退化成显示裸 0（看着像选了个 id=0 的任务）
        const { taskId, ...rest } = props.data;
        return { ...rest, taskId: taskId || undefined };
    }
    return {
        sourceType: TriggerSourceEnum.Alert.value as string,
        enabled: AgentTaskStatusEnum.Enable.value as number,
        profileId: ProfileEnum.Diagnostician.value as string,
        cooldownSeconds: 0,
        // 结果通知必须显式预置为「不通知」：留空会被当成「非不通知」而把通知模板拖成必填，
        // 与后端 notifyType 零值语义（None）相反，也与任务表单不同源
        notifyType: AgentTaskNotifyTypeEnum.None.value as number,
    };
});
</script>
