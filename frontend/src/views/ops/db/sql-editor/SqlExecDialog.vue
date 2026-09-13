<template>
    <div>
        <el-dialog title="SQL" v-model="dialogVisible" :show-close="false" width="600px" :close-on-click-modal="false">
            <monaco-editor height="300px" class="codesql" language="sql" v-model="sqlValue" />
            <el-input @keyup.enter="runSql" ref="remarkInputRef" v-model="remark" :placeholder="i18n.global.t('common.remark')" class="mt-1" />

            <template #footer>
                <span class="dialog-footer">
                    <el-button @click="cancel">{{ i18n.global.t('common.cancel') }}</el-button>
                    <el-button @click="runSql" type="primary" :loading="btnLoading">{{ i18n.global.t('db.run') }}</el-button>
                </span>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { dbApi } from '@/views/ops/db/api';
import { ElButton, ElDialog, ElInput, InputInstance } from 'element-plus';
import { onMounted, reactive, ref, toRefs, defineAsyncComponent } from 'vue';

import { isTrue } from '@/common/assert';
import { Msg } from '@/hooks/useI18n';
import { i18n } from '@/i18n';
import { formatSql } from './utils/formatSql';
import type { SqlExecProps } from './SqlExecBox';

/**
 * 编辑器主体约 3.9M，而本弹窗只在用户真正执行 SQL 时出现。
 * SqlExecBox 由 db.ts（DbInst）静态引入，若此处静态引入编辑器，
 * 整份编辑器就会顺着「db 模块入口 → db.ts」回流到表数据页、资源树、实例列表等所有 DB 页面。
 */
const MonacoEditor = defineAsyncComponent(() => import('@/components/monaco/MonacoEditor.vue'));

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

onMounted(async () => {
    await open();
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
                Msg.error(`${re.sql} ==>: ${re.errorMsg}`);
            }
        }

        isTrue(isSuccess, 'exist run faild sql');
        Msg.success('db.execSuccess');
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

const open = async () => {
    // 先出弹窗再填内容：格式化器与编辑器都是按需加载的，弱网下也不至于点击无反馈
    state.dialogVisible = true;
    // 先填原始 sql，避免格式化器加载期间是个空编辑器（用户此刻点执行也不至于执行空语句）
    state.sqlValue = props.sql;
    state.sqlValue = await formatSql(props.sql, props.formatDialect);
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
