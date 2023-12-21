<template>
    <div>
        <el-dialog title="待执行SQL" v-model="dialogVisible" :show-close="false" width="600px" @close="cancel">
            <monaco-editor height="300px" class="codesql" language="sql" v-model="sqlValue" />
            <el-input @keyup.enter="runSql" ref="remarkInputRef" v-model="remark" placeholder="请输入执行备注" class="mt5" />
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
import { toRefs, ref, nextTick, reactive } from 'vue';
import { dbApi } from '@/views/ops/db/api';
import { ElDialog, ElButton, ElInput, ElMessage, InputInstance } from 'element-plus';
// import base style
import MonacoEditor from '@/components/monaco/MonacoEditor.vue';
import { format as sqlFormatter } from 'sql-formatter';

import { SqlExecProps } from './SqlExecBox';

const props = defineProps({
    visible: {
        type: Boolean,
    },
    dbId: {
        type: [Number],
    },
    db: {
        type: String,
    },
    sql: {
        type: String,
    },
});

const remarkInputRef = ref<InputInstance>();
const state = reactive({
    dialogVisible: false,
    sqlValue: '',
    dbId: 0,
    db: '',
    remark: '',
    btnLoading: false,
});

const { dialogVisible, sqlValue, remark, btnLoading } = toRefs(state);

state.sqlValue = props.sql as any;
let runSuccessCallback: any;
let cancelCallback: any;
let runSuccess: boolean = false;

/**
 * 执行sql
 */
const runSql = async () => {
    if (!state.remark) {
        ElMessage.error('请输入执行的备注信息');
        return;
    }

    try {
        state.btnLoading = true;
        const res = await dbApi.sqlExec.request({
            id: state.dbId,
            db: state.db,
            remark: state.remark,
            sql: state.sqlValue.trim(),
        });

        for (let re of res.res) {
            if (re.result !== 'success') {
                ElMessage.error(`${re.sql} \n执行失败: ${re.result}`);
                throw new Error(re.result);
            }
        }

        runSuccess = true;
        ElMessage.success('执行成功');
    } catch (e) {
        runSuccess = false;
    }
    if (runSuccess) {
        if (runSuccessCallback) {
            runSuccessCallback();
        }
        cancel();
    }
    state.btnLoading = false;
};

const cancel = () => {
    state.dialogVisible = false;
    // 没有执行成功，并且取消回调函数存在，则执行
    if (!runSuccess && cancelCallback) {
        cancelCallback();
    }
    setTimeout(() => {
        state.dbId = 0;
        state.sqlValue = '';
        state.remark = '';
        runSuccessCallback = null;
        cancelCallback = null;
        runSuccess = false;
    }, 200);
};

const open = (props: SqlExecProps) => {
    runSuccessCallback = props.runSuccessCallback;
    cancelCallback = props.cancelCallback;
    props.dbType = props.dbType || 'mysql';
    state.sqlValue = sqlFormatter(props.sql, { language: props.dbType });
    state.dbId = props.dbId;
    state.db = props.db;
    state.dialogVisible = true;
    nextTick(() => {
        setTimeout(() => {
            remarkInputRef.value?.focus();
        });
    });
};

defineExpose({ open });
</script>
<style lang="scss">
.codesql {
    font-size: 9pt;
    font-weight: 600;
}
</style>
