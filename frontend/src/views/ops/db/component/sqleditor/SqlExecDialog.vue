<template>
    <div>
        <el-dialog title="待执行SQL" v-model="dialogVisible" :show-close="false" width="600px" :close-on-click-modal="false">
            <monaco-editor height="300px" class="codesql" language="sql" v-model="sqlValue" />
            <el-input @keyup.enter="runSql" ref="remarkInputRef" v-model="remark" placeholder="执行备注" class="mt5" />

            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="cancel">取 消</el-button>
                    <el-button @click="runSql" type="primary" :loading="btnLoading">执 行</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { toRefs, ref, reactive, onMounted } from 'vue';
import { dbApi } from '@/views/ops/db/api';
import { ElDialog, ElButton, ElInput, ElMessage, InputInstance } from 'element-plus';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { format as sqlFormatter } from 'sql-formatter';

import { SqlExecProps } from './SqlExecBox';
import { isTrue } from '@/common/assert';

const props = withDefaults(defineProps<SqlExecProps>(), {});

const remarkInputRef = ref<InputInstance>();
const state = reactive({
    dialogVisible: false,
    sqlValue: '',
    remark: '',
    btnLoading: false,
});

const { dialogVisible, sqlValue, remark, btnLoading } = toRefs(state);

let runSuccess: boolean = false;

onMounted(() => {
    open();
});

/**
 * 执行sql
 */
const runSql = async () => {
    try {
        state.btnLoading = true;
        runSuccess = true;

        const res = await dbApi.sqlExec.request({
            id: props.dbId,
            db: props.db,
            remark: state.remark,
            sql: state.sqlValue.trim(),
        });

        let isSuccess = true;
        for (let re of res) {
            if (re.errorMsg) {
                isSuccess = false;
                ElMessage.error(`${re.sql} \n执行失败: ${re.errorMsg}`);
            }
        }

        isTrue(isSuccess, '存在执行失败sql');
        ElMessage.success('执行成功');
    } catch (e) {
        runSuccess = false;
    } finally {
        if (runSuccess) {
            if (props.runSuccessCallback) {
                props.runSuccessCallback();
            }
            cancel();
        }
        state.btnLoading = false;
    }
};

const cancel = () => {
    state.dialogVisible = false;
    props.cancelCallback && props.cancelCallback();
    setTimeout(() => {
        state.sqlValue = '';
        state.remark = '';
        runSuccess = false;
    }, 200);
};

const open = () => {
    state.sqlValue = sqlFormatter(props.sql, { language: (props.dbType || 'mysql') as any });
    state.dialogVisible = true;
    setTimeout(() => {
        remarkInputRef.value?.focus();
    }, 200);
};

defineExpose({ open });
</script>
<style lang="scss">
.codesql {
    font-size: 9pt;
    font-weight: 600;
}
</style>
