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
import { onMounted, ref, watch } from 'vue';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { registerDbCompletionItemProvider } from '@/views/ops/db/db';
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
    default: {
        dbId: 0,
        instName: '',
        dbName: '',
        dbType: '',
        tagPath: '',
        sql: '',
    },
});

onMounted(() => {
    if (bizForm.value.dbId) {
        registerDbCompletionItemProvider(bizForm.value.dbId, bizForm.value.dbName, [bizForm.value.dbName], bizForm.value.dbType);
    }
});

watch(
    () => bizForm.value.dbId,
    () => {
        registerDbCompletionItemProvider(bizForm.value.dbId, bizForm.value.dbName, [bizForm.value.dbName], bizForm.value.dbType);
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
