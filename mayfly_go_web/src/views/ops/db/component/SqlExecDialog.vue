<template>
    <div>
        <el-dialog title="待执行SQL" v-model="dialogVisible" :show-close="false" width="600px">
            <codemirror height="350px" class="codesql" ref="cmEditor" language="sql" v-model="sql" :options="cmOptions" />
            <div class="footer mt10">
                <el-button @click="runSql" type="primary" :loading="btnLoading">执 行</el-button>
                <el-button @click="cancel">取 消</el-button>
            </div>
        </el-dialog>
    </div>
</template>

<script lang="ts">
import { toRefs, reactive, defineComponent } from 'vue';
import { dbApi } from '../api';
import { ElDialog, ElButton } from 'element-plus';
// import base style
import 'codemirror/lib/codemirror.css';
// 引入主题后还需要在 options 中指定主题才会生效
import 'codemirror/theme/base16-light.css';
import 'codemirror/addon/selection/active-line';
import { codemirror } from '@/components/codemirror';
import { format as sqlFormatter } from 'sql-formatter';

import { SqlExecProps } from './SqlExecBox';

export default defineComponent({
    name: 'SqlExecDialog',
    components: {
        codemirror,
        ElButton,
        ElDialog,
    },
    props: {
        visible: {
            type: Boolean,
        },
        dbId: {
            type: [Number],
        },
        sql: {
            type: String,
        },
    },
    setup(props: any) {
        const state = reactive({
            dialogVisible: false,
            sql: '',
            dbId: 0,
            btnLoading: false,
            cmOptions: {
                tabSize: 4,
                mode: 'text/x-sql',
                lineNumbers: true,
                line: true,
                indentWithTabs: true,
                smartIndent: true,
                matchBrackets: true,
                theme: 'base16-light',
                autofocus: true,
                extraKeys: { Tab: 'autocomplete' }, // 自定义快捷键
            },
        });

        let runSuccessCallback: any;
        let cancelCallback: any;
        let runSuccess: boolean = false;

        /**
         * 执行sql
         */
        const runSql = async () => {
            try {
                state.btnLoading = true;
                await dbApi.sqlExec.request({
                    id: state.dbId,
                    sql: state.sql.trim(),
                });
                runSuccess = true;
            } catch (e) {
                runSuccess = false;
            }
            if (runSuccess && runSuccessCallback) {
                runSuccessCallback();
            }
            state.btnLoading = false;
            cancel();
        };

        const cancel = () => {
            state.dialogVisible = false;
            // 没有执行成功，并且取消回调函数存在，则执行
            if (!runSuccess && cancelCallback) {
                cancelCallback();
            }
            setTimeout(() => {
                state.dbId = 0;
                state.sql = '';
                runSuccessCallback = null;
                cancelCallback = null;
                runSuccess = false;
            }, 200);
        };

        const open = (props: SqlExecProps) => {
            runSuccessCallback = props.runSuccessCallback;
            cancelCallback = props.cancelCallback;
            state.sql = sqlFormatter(props.sql);
            state.dbId = props.dbId;
            state.dialogVisible = true;
        };

        return {
            ...toRefs(state),
            open,
            runSql,
            cancel,
        };
    },
});
</script>
<style lang="scss">
.codesql {
    font-size: 9pt;
    font-weight: 600;
}
.footer {
    float: right;
}
</style>
