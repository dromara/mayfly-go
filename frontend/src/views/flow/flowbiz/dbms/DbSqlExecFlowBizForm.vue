<template>
    <el-form :model="bizForm" ref="formRef" :rules="rules" label-width="auto">
        <el-form-item prop="dbId" label="数据库" required>
            <db-select-tree
                placeholder="请选择数据库"
                v-model:db-id="bizForm.dbId"
                v-model:db-name="bizForm.dbName"
                v-model:db-type="dbType"
                @select-db="changeResourceCode"
            />
        </el-form-item>

        <el-form-item prop="sql" label="SQL" required>
            <div class="w100">
                <monaco-editor height="300px" language="sql" v-model="bizForm.sql" />
            </div>
        </el-form-item>
    </el-form>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue';
import DbSelectTree from '@/views/ops/db/component/DbSelectTree.vue';
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { registerDbCompletionItemProvider } from '@/views/ops/db/db';
import { TagResourceTypeEnum } from '@/common/commonEnum';

const rules = {
    dbId: [
        {
            required: true,
            message: '请选择数据库',
            trigger: ['change', 'blur'],
        },
    ],
    sql: [
        {
            required: true,
            message: '请输入执行SQL',
            trigger: ['change', 'blur'],
        },
    ],
};

const emit = defineEmits(['changeResourceCode']);

const formRef: any = ref(null);

const bizForm = defineModel<any>('bizForm', {
    default: {
        dbId: 0,
        dbName: '',
        sql: '',
    },
});

const dbType = ref('');

watch(
    () => bizForm.value.dbId,
    () => {
        registerDbCompletionItemProvider(bizForm.value.dbId, bizForm.value.dbName, [bizForm.value.dbName], dbType.value);
    }
);

const changeResourceCode = async (db: any) => {
    emit('changeResourceCode', TagResourceTypeEnum.DbName.value, db.code);
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
