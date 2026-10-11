<template>
    <el-dialog v-model="visible" :title="$t('db.sqlPlanTab')" width="1000px" destroy-on-close append-to-body>
        <SqlPlanList :plans="plans" :loading="loading" />
        <template #footer>
            <el-button type="primary" @click="visible = false">{{ $t('common.close') }}</el-button>
        </template>
    </el-dialog>
</template>

<script lang="ts" setup>
/**
 * SQL 执行计划查看对话框（可复用）。
 *
 * 给定「库 + SQL 文本」，打开时经 useSqlExplain 自取数（只读、逐条 EXPLAIN 不带 ANALYZE），
 * 多条由 SqlPlanList 直接列表逐条展示。方言不支持 / 库不可达时逐条给出说明，前端如实展示不粉饰。
 * 供编辑器执行计划入口、表编辑执行确认框、编辑器执行确认框三处复用（经全局 openSqlPlan 唤起）。
 */
import { watch } from 'vue';
import SqlPlanList from './SqlPlanList.vue';
import { useSqlExplain } from './useSqlExplain';

const props = defineProps<{ dbId: number; db: string; sql: string }>();

const visible = defineModel<boolean>('visible', { default: false });

const { plans, loading, fetchPlans } = useSqlExplain();

// 每次打开按当前 SQL 重新拉取（destroy-on-close 保证状态干净）
watch(visible, (opened) => {
    if (opened) fetchPlans(props.dbId, props.db, props.sql);
});
</script>
