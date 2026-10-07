<template>
    <div>
        <auto-form-drawer
            v-model:visible="visible"
            :title="title"
            :items="items"
            :data="editData"
            :size="FLOW_DRAWER.form"
            :confirm-api="onConfirm"
            :close-guard="closeGuard"
            @submitted="emit('cancel')"
            @opened="loadDetailOnOpened"
            @cancel="emit('cancel')"
        >
            <!-- 消息模板选择 -->
            <template #msgTmplId="{ form }">
                <MsgTmplSelect v-model="form.msgTmplId" clearable />
            </template>

            <!-- 触发策略：内置检查项 + 自定义条件树，元素清单由后端 policy-schema 下发 -->
            <template #triggerPolicy="{ form }">
                <ProcdefPolicyEditor ref="policyEditorRef" v-if="form.triggerPolicy" :policy="form.triggerPolicy" :schema="policySchema" />
                <div v-else class="policy-pending">{{ $t('flow.policy.loading') }}</div>

                <!-- 变更时间线只在已保存的流程上出现：新建时还没有可回溯的历史 -->
                <ProcdefPolicyHistory v-if="form.id" :procdef-id="form.id" :schema="policySchema" />
            </template>

            <!-- 关联标签 -->
            <template #codePaths="{ form }">
                <!-- schema 到货前不渲染：拿不到「哪些资源可被治理」就先去拉标签树，
                     只能靠一份会过期的保底清单，结果就是「首次打开缺节点、重开才对」 -->
                <tag-tree-check v-if="governTagTypes" height-mode="fixed" height="300px" v-model="form.codePaths" :tag-type="governTagTypes" />
            </template>
        </auto-form-drawer>
    </div>
</template>

<script lang="ts" setup>
import { Rules } from '@/common/rule';
import { Msg, useI18nConfirm } from '@/hooks/useI18n';
import { createPolicy, usePolicySchema } from '@/components/policy-builder';
import { AutoFormDrawer, defineFormItems } from '@/components/auto-form';
import { useAutoFormModel } from '@/hooks/useAutoFormModel';
import { computed, onMounted, ref, useTemplateRef, type PropType } from 'vue';
import MsgTmplSelect from '../msg/components/MsgTmplSelect.vue';
import TagTreeCheck from '../ops/component/TagTreeCheck.vue';
import { procdefApi } from './api';
import ProcdefPolicyEditor from './components/ProcdefPolicyEditor.vue';
import ProcdefPolicyHistory from './components/ProcdefPolicyHistory.vue';
import { governTagTypesOf } from './governance';
import { ProcdefStatus } from './enums';
import type { ProcdefVO } from './types';
import { FLOW_DRAWER } from '@/views/flow/drawerSize';

const props = defineProps({
    data: {
        type: Object as PropType<ProcdefVO | null>,
        default: null,
    },
    title: {
        type: String,
    },
});

const visible = defineModel<boolean>('visible', { default: false });

//定义事件
const emit = defineEmits(['cancel', 'val-change']);

interface ProcdefForm extends Partial<ProcdefVO> {
    msgTmplId?: number | null;
    codePaths?: string[];
}

const { policySchema, load: loadPolicySchema } = usePolicySchema(() => procdefApi.policySchema.request());

/** 表单声明（defineFormItems<ProcdefForm>，渲染 + 校验唯一数据源；触发策略/消息模板/关联标签走 custom 插槽） */
const items = defineFormItems<ProcdefForm>([
    { prop: 'name', label: 'common.name', rules: [Rules.requiredInput('common.name')] },
    { prop: 'defKey', label: 'Key', disabled: (f) => !!f.id, rules: [Rules.requiredInput('key')] },
    { prop: 'status', label: 'common.status', type: 'enum', enums: ProcdefStatus },
    { prop: 'triggerPolicy', label: 'flow.triggeringCondition', type: 'custom', tooltip: 'flow.triggeringConditionTips' },
    { prop: 'remark', label: 'common.remark' },
    { prop: 'msgTmplId', label: 'flow.notify', type: 'custom' },
    { prop: 'codePaths', label: 'tag.relateTag', type: 'custom' },
]);

/** 关联标签路径（列表行与详情均经 FillTagInfo 携带 tags） */
const codePathsOf = (procdef?: ProcdefVO | null) => procdef?.tags?.map((tag) => tag.codePath) ?? [];

/**
 * 「生效资源」树里可勾选的节点类型，取后端下发的治理路径并集（见 governance.ts）。
 *
 * 以前这里写死为「数据库 + Redis」，机器命令场景在后端早就注册完毕，
 * 管理员却根本选不到机器节点：策略配得再全也永远不会命中。
 * 这里不再内置一份保底清单：保底清单就是第二份「谁能被治理」的真源，一定会过期。
 */
const governTagTypes = computed<(number | string)[] | null>(() => governTagTypesOf(policySchema.value?.governPaths));

/** 传给 AutoFormDrawer 的回填数据（编辑态完整详情由 loadDetailOnOpened 异步补充回填） */
const editData = computed<ProcdefForm>(() => {
    if (props.data) {
        return {
            ...props.data,
            // 草稿在数据入口补齐，插槽只读不写：模板里写表单对象属于渲染期副作用
            triggerPolicy: props.data.triggerPolicy ?? createPolicy(),
            msgTmplId: null,
            codePaths: codePathsOf(props.data),
        };
    }
    return {
        status: ProcdefStatus.Enable.value,
        triggerPolicy: createPolicy(),
        msgTmplId: null,
        codePaths: [],
    };
});

// policy-schema 走共享缓存：条件组增删后失效重取的是同一份。本组件原先自己 request 到局部 ref 上，
// 就会停在挂载那一刻的旧字典（抽屉在 SPA 内不重新挂载，新建的条件组在界面上始终看不见）
onMounted(loadPolicySchema);

// 宿主抽屉的内部表单在 @opened 接管（提交载荷即它）
const { requireForm, openedWith } = useAutoFormModel<ProcdefForm>();

/** 编辑态异步补充回填详情（触发条件/消息模板等列表行未携带的字段） */
const loadDetailOnOpened = openedWith(async (form) => {
    openedForm = form;
    if (props.data?.id) {
        const detail = await procdefApi.detail.request({ id: props.data.id });
        Object.assign(form, detail, { triggerPolicy: detail.triggerPolicy ?? createPolicy(), codePaths: codePathsOf(detail) });
    }
    // 回填 settle 后拍基线快照：策略编辑器原地突变 form.triggerPolicy，关闭前与快照比对即知脏否
    formBaseline = JSON.stringify(form);
});

/** 打开期间的内部表单引用与基线快照（opened 前为 null，守卫直接放行） */
let openedForm: ProcdefForm | null = null;
let formBaseline = '';

// 误关防护：策略编辑器一改就是大段配置，Esc / 返回箭头 / 取消按钮误触丢全部未保存修改不可接受；
// 脏则弹确认，用户选择放弃才真正关闭
const closeGuard = async (): Promise<boolean> => {
    if (!openedForm || JSON.stringify(openedForm) === formBaseline) {
        return true;
    }
    return useI18nConfirm('flow.unsavedCloseConfirm');
};

const submitForm = computed(requireForm);

// 策略编辑器的问题面板已实时算出不合法项，提交前必须据它中止：
// 否则用户点保存后要等一次后端拒绝才知道哪里没填对
const policyEditorRef = useTemplateRef<{ validate: () => string | null }>('policyEditorRef');

const { execute: saveFlowDefExec } = procdefApi.save.useApi(submitForm);

// confirmApi 提交动作（无参，从内部表单读取提交数据）；成功提示与关闭抽屉由组件内置逻辑处理
const onConfirm = async () => {
    const issue = policyEditorRef.value?.validate();
    if (issue) {
        // 抛错而不是 return：宿主把「正常返回」当作提交成功，会弹保存成功并关掉抽屉
        Msg.error(issue);
        throw new Error(issue);
    }
    await saveFlowDefExec();
    emit('val-change', submitForm.value);
};
</script>
<style lang="scss" scoped>
.policy-pending {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}
</style>
