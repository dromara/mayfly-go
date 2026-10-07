<template>
    <auto-form ref="formRef" v-model="bizForm" :items="bizItems" label-width="auto">
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
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { AutoForm, type AutoFormItem } from '@/components/auto-form';
import { Rules } from '@/common/rule';

/** DB SQL 执行业务表单声明（库选择与 SQL 编辑器为 custom 插槽） */
const bizItems: AutoFormItem[] = [
    { prop: 'dbId', label: 'tag.db', type: 'custom', required: true, rules: Rules.requiredSelect('db.db') },
    { prop: 'sql', label: 'SQL', type: 'custom', required: true, rules: Rules.requiredInput('flow.runSql') },
];

const emit = defineEmits(['changeResourceCode']);

const formRef = ref<{ validate: (...args: unknown[]) => unknown; resetFields?: () => void } | null>(null);
const monacoEditorRef = useTemplateRef<MonacoEditorExpose>('monacoEditorRef');
let editorUri: string | undefined;

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
    }),
});

// 本表单的 SQL 联想使用方作用域（多使用方共存时按计数释放，见 db/completion/lazy.ts）
const sqlCompletion = createSqlCompletionScope();

// 选库后注册联想上下文；编辑器的补全注册表按语言全局唯一，故释放必须与申请成对
onBeforeUnmount(() => {
    sqlCompletion.release();
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

const changeResourceCode = async (db: any) => {
    emit('changeResourceCode', TagResourceTypeEnum.Db.value, db.dbCode);
};

const validateBizForm = async () => {
    return formRef.value?.validate();
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
