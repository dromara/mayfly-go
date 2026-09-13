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
                <monaco-editor height="300px" language="sql" v-model="bizForm.sql" />
            </div>
        </template>
    </auto-form>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue';
import DbSelectTree from '@/views/ops/db/widgets/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
// completion 桶只能经惰性作用域触达（见 db/completion/lazy.ts 的边界约束）
import { createSqlCompletionScope } from '@/views/ops/db/completion/lazy';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import type { AutoFormItem } from '@/components/auto-form';
import { Rules } from '@/common/rule';

/** DB SQL 执行业务表单声明（库选择与 SQL 编辑器为 custom 插槽） */
const bizItems: AutoFormItem[] = [
    { prop: 'dbId', label: 'tag.db', type: 'custom', required: true, rules: Rules.requiredSelect('db.db') },
    { prop: 'sql', label: 'SQL', type: 'custom', required: true, rules: Rules.requiredInput('flow.runSql') },
];

const emit = defineEmits(['changeResourceCode']);

const formRef = ref<{ validate: (...args: unknown[]) => unknown; resetFields?: () => void } | null>(null);

const bizForm = defineModel<any>('bizForm', {
    // 对象默认值必须是工厂函数：字面量只创建一次，会被多个实例共享
    default: () => ({
        dbId: 0,
        instName: '',
        dbName: '',
        dbType: '',
        tagPath: '',
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
    sqlCompletion.register(bizForm.value.dbId, bizForm.value.dbName, [bizForm.value.dbName], bizForm.value.dbType);
};

onMounted(() => {
    if (bizForm.value.dbId) {
        registerCompletion();
    }
});

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
