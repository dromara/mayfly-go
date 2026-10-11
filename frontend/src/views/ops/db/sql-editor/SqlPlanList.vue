<template>
    <div v-loading="loading" class="min-h-[120px] space-y-3">
        <!-- 多条逐条列出：一屏扫完全部语句的计划才是目的，不做下拉一条条切 -->
        <div v-for="(plan, index) in plans" :key="index" class="rounded border border-solid border-[var(--el-border-color-lighter)] p-2">
            <SqlPlanView :plan="plan" />
        </div>
        <!-- 一条都没有时不能留白：留白与「还没加载」在用户眼里长得一样 -->
        <el-empty v-if="!loading && !plans.length" :description="emptyText ?? $t('db.planNotLoaded')" :image-size="60" />
    </div>
</template>

<script lang="ts" setup>
/**
 * 执行计划列表：把「多条逐条卡片 + 加载态 + 空态」这一层外壳收敛到一处。
 *
 * 只负责呈现传入的 plans，不取数（取数由 useSqlExplain 或调用方的快照负责）；
 * 单条计划的具体渲染仍委托 SqlPlanView。供执行计划对话框、提单预估面板、审批详情快照三处复用。
 */
import type { SqlExplainPlan } from '../types';
import SqlPlanView from './SqlPlanView.vue';

defineProps<{ plans: SqlExplainPlan[]; loading?: boolean; emptyText?: string }>();
</script>
