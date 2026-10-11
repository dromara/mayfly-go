<template>
    <el-dialog v-model="visible" :title="$t('db.runSql')" width="460px" append-to-body>
        <el-input v-model="remark" type="textarea" :rows="2" :placeholder="$t('db.execRemarkPlaceholder')" />
        <template #footer>
            <div class="flex items-center justify-between">
                <!-- 执行前就地看计划：预估影响行数等，无需先跑到结果里再看 -->
                <el-button link type="primary" @click="openSqlPlan(dbId, db, sql)">{{ $t('db.viewPlan') }}</el-button>
                <div class="flex gap-2">
                    <el-button @click="cancel">{{ $t('common.cancel') }}</el-button>
                    <el-button type="primary" @click="confirm">{{ $t('db.run') }}</el-button>
                </div>
            </div>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * 编辑器执行非查询语句前的确认框：填执行备注 + 查看执行计划。
 *
 * open() 返回 Promise：确认 resolve 备注文本，取消 resolve null（调用方据此中止执行）。
 * 取代原生 ElMessageBox.prompt——原生框放不下「查看执行计划」这个入口。
 */
import { ref } from 'vue';
import { openSqlPlan } from './sqlPlanService';

const visible = ref(false);
const remark = ref('');
const dbId = ref(0);
const db = ref('');
const sql = ref('');

let resolver: ((value: string | null) => void) | null = null;

function open(id: number, dbName: string, sqlText: string): Promise<string | null> {
    dbId.value = id;
    db.value = dbName;
    sql.value = sqlText;
    remark.value = '';
    visible.value = true;
    return new Promise<string | null>((resolve) => {
        resolver = resolve;
    });
}

function settle(value: string | null) {
    visible.value = false;
    resolver?.(value);
    resolver = null;
}

const confirm = () => settle(remark.value);
const cancel = () => settle(null);

defineExpose({ open });
</script>
