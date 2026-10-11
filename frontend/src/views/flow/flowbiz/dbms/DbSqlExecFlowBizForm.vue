<template>
    <auto-form ref="formRef" v-model="bizForm" :items="bizItems" label-position="top">
        <template #dbId>
            <db-select-tree
                :placeholder="$t('flow.selectDbPlaceholder')"
                v-model:db-id="bizForm.dbId"
                v-model:db-name="bizForm.dbName"
                v-model:inst-name="bizForm.instName"
                v-model:db-type="bizForm.dbType"
                v-model:tag-path="bizForm.tagPath"
                @select-db="changeResourceCode"
            />
        </template>
        <template #sql>
            <div class="w-full!">
                <monaco-editor ref="monacoEditorRef" height="300px" language="sql" v-model="bizForm.sql" @ready="onMonacoReady" />
            </div>
        </template>
        <template #impact>
            <div class="w-full!">
                <div class="mb-2 flex flex-wrap items-center gap-2">
                    <el-button size="small" :loading="plansLoading" @click="capturePlans">{{ $t('flow.refreshImpact') }}</el-button>
                    <span class="text-[12px] text-[var(--el-text-color-secondary)]">{{ $t('flow.impactSnapshotTip') }}</span>
                </div>
                <SqlPlanList :plans="bizForm.plans ?? []" :loading="plansLoading" :empty-text="$t('flow.impactEmpty')" />
            </div>
        </template>
    </auto-form>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, nextTick, onMounted, ref, useTemplateRef, watch } from 'vue';
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import type { MonacoEditorExpose } from '@/components/monaco/types';
// completion 桶只能经惰性作用域触达（见 db/completion/lazy.ts 的边界约束）
import { createSqlCompletionScope } from '@/views/ops/db/completion/lazy';
import { dbApi } from '@/views/ops/db/api';
import SqlPlanList from '@/views/ops/db/sql-editor/SqlPlanList.vue';
import { useSqlExplain } from '@/views/ops/db/sql-editor/useSqlExplain';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Rules } from '@/common/rule';

/** DB SQL 执行业务表单声明（库选择与 SQL 编辑器为 custom 插槽） */
const bizItems: AutoFormItem[] = [
    { prop: 'dbId', label: 'tag.db', type: 'custom', required: true, rules: Rules.requiredSelect('db.db') },
    { prop: 'sql', label: 'SQL', type: 'custom', required: true, rules: Rules.requiredInput('flow.runSql') },
    { prop: 'impact', label: 'flow.sqlImpact', type: 'custom' },
];

const emit = defineEmits(['changeResourceCode']);

const formRef = ref<{ validate: (...args: unknown[]) => unknown; resetFields?: () => void } | null>(null);
const monacoEditorRef = useTemplateRef<MonacoEditorExpose>('monacoEditorRef');
let editorUri: string | undefined;

// 取数收敛到 useSqlExplain（接口调用/并发序号守卫/失败降级都在那一处），本表单只把结果写进快照
const { loading: plansLoading, fetchPlans } = useSqlExplain();

/**
 * 抓取当前 SQL 的逐条执行计划快照并写入 bizForm.plans。
 *
 * 只读、best-effort：无库/无 SQL/方言不支持/库不可达时 fetchPlans 返回空列表，面板自然显示空态，不阻断提单。
 * 提交人本就持有该库的运行权限（否则也不会提这条 SQL），故在提交表单里拉计划是安全的。
 */
const capturePlans = async () => {
    bizForm.value.plans = await fetchPlans(bizForm.value.dbId, bizForm.value.dbName, bizForm.value.sql);
};

// 预估影响自动抓取：库/SQL 变化后防抖重取（预填打开即出，无需手点）；按钮保留作手动刷新
let captureTimer: ReturnType<typeof setTimeout> | null = null;
const scheduleAutoCapture = (delay = 700) => {
    if (captureTimer) {
        clearTimeout(captureTimer);
    }
    captureTimer = setTimeout(capturePlans, delay);
};

const bizForm = defineModel<any>('bizForm', {
    // 对象默认值必须是工厂函数：字面量只创建一次，会被多个实例共享
    default: () => ({
        dbId: 0,
        instName: '',
        dbName: '',
        dbType: '',
        tagPath: '',
        // 库的资源编码：从被拦下的操作提单时由宿主带入，用于自动解析审批流程
        dbCode: '',
        sql: '',
        // 提交时抓取的逐条执行计划快照（审批页只读渲染，审批人无需连库）
        plans: [],
    }),
});

// 本表单的 SQL 联想使用方作用域（多使用方共存时按计数释放，见 db/completion/lazy.ts）
const sqlCompletion = createSqlCompletionScope();

// 选库后注册联想上下文；编辑器的补全注册表按语言全局唯一，故释放必须与申请成对
onBeforeUnmount(() => {
    sqlCompletion.release();
    if (captureTimer) {
        clearTimeout(captureTimer);
    }
});

const registerCompletion = () => {
    sqlCompletion.register(bizForm.value.dbId, bizForm.value.dbName, [bizForm.value.dbName], bizForm.value.dbType, editorUri);
};

/** 编辑器就绪后上报 editorUri，使多编辑器共存时补全按编辑器路由 */
const onMonacoReady = () => {
    const editor = monacoEditorRef.value?.getEditor();
    if (editor) {
        editorUri = editor.getModel()?.uri.toString();
        if (editorUri && bizForm.value.dbId) {
            registerCompletion();
        }
    }
};

onMounted(() => {
    if (bizForm.value.dbId) {
        registerCompletion();
    }
    announceResource();
    // 被拦下的一键提单会预填好库与完整 SQL：打开即自动估算，不必再点按钮
    if (bizForm.value.dbId && bizForm.value.sql?.trim()) {
        capturePlans();
    }
});

/**
 * 预填完库时主动上报一次资源标识。
 *
 * 正常提单是用户选库后由 select-db 触发的，而被策略拦下的一键提单不会再有选库动作：
 * 不上报，宿主就解析不出流程定义，抽屉会停在「不存在审批节点」且确定按钮不可点。
 * 宿主只知 dbId/dbName 而不知资源编码时（如函数式执行确认框），按 id 反查补齐——
 * 与 DbSelectTree 预填回显同一数据源；反查失败不阻断，抽屉里重选库仍可触发 select-db 上报。
 * 等一个 tick 再抛：宿主要先完成表单接管才能接住这次变更
 */
const announceResource = async () => {
    if (!bizForm.value.dbCode && bizForm.value.dbId) {
        try {
            const res = await dbApi.dbs.request({ id: bizForm.value.dbId });
            // 反查期间用户可能已手选库（select-db 已带真实 code），不覆盖
            if (!bizForm.value.dbCode) {
                bizForm.value.dbCode = res.list?.[0]?.code ?? '';
            }
        } catch {
            // 保持 dbCode 为空：下方 return，交给用户重选
        }
    }
    const code = bizForm.value.dbCode;
    if (!code) {
        return;
    }
    nextTick(() => emit('changeResourceCode', TagResourceTypeEnum.Db.value, code));
};

watch(
    () => bizForm.value.dbId,
    () => {
        registerCompletion();
    }
);

// 库或 SQL 变化后防抖自动重取预估影响（只读、best-effort，失败自然降级为“无法获取”）
watch([() => bizForm.value.dbId, () => bizForm.value.sql], () => {
    scheduleAutoCapture();
});

const changeResourceCode = async (db: any) => {
    emit('changeResourceCode', TagResourceTypeEnum.Db.value, db.dbCode);
};

const validateBizForm = async () => {
    const ok = await formRef.value?.validate();
    // 提交时确保有一份与最终 SQL 一致的计划快照（用户没手动点过也补抓一次）；best-effort，失败不阻断
    if (ok) {
        await capturePlans();
    }
    return ok;
};

const resetBizForm = () => {
    //重置表单域
    formRef.value?.resetFields?.();
    bizForm.value.dbId = 0;
    bizForm.value.dbName = '';
};

defineExpose({ validateBizForm, resetBizForm });
</script>
<style lang="scss"></style>
