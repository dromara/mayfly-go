<template>
    <el-form :model="bizForm" ref="formRef" :rules="rules" label-width="auto">
        <el-form-item prop="dbId" :label="$t('tag.db')" required>
            <db-select-tree
                :placeholder="$t('flow.selectDbPlaceholder')"
                v-model:db-id="bizForm.dbId"
                v-model:db-name="bizForm.dbName"
                v-model:inst-name="bizForm.instName"
                v-model:db-type="bizForm.dbType"
                v-model:tag-path="bizForm.tagPath"
                @select-db="changeResourceCode"
            />
        </el-form-item>

        <el-form-item prop="sql" label="SQL" required>
            <div class="w-full!">
                <monaco-editor height="300px" language="sql" v-model="bizForm.sql" />
            </div>
        </el-form-item>
    </el-form>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from 'vue';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { registerDbCompletionItemProvider } from '@/views/ops/db/db';
import { TagResourceTypeEnum } from '@/common/commonEnum';
import { Rules } from '@/common/rule';

const rules = {
    dbId: [Rules.requiredSelect('db.db')],
    sql: [Rules.requiredInput('flow.runSql')],
};

const emit = defineEmits(['changeResourceCode']);

const formRef: any = ref(null);

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
    emit('changeResourceCode', TagResourceTypeEnum.Db.value, db.code);
};

const validateBizForm = async () => {
    return formRef.value.validate();
};

const resetBizForm = () => {
    //重置表单域
    formRef.value.resetFields();
    bizForm.value.dbId = 0;
    bizForm.value.dbName = '';
};

defineExpose({ validateBizForm, resetBizForm });
</script>
<style lang="scss"></style>
