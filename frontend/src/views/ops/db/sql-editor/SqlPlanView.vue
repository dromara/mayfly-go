<template>
    <div class="sql-plan-view">
        <div class="mb-2 break-all rounded bg-[var(--el-fill-color-light)] px-2 py-1 font-mono text-[12px]">{{ plan.sql }}</div>

        <!-- 拿不到计划就明说：画一张空图或只列 0 个节点，会被读成「这条语句没有扫描」 -->
        <el-alert v-if="plan.note" type="warning" :title="plan.note" :closable="false" show-icon />

        <template v-else-if="result">
            <div class="mb-2 flex flex-wrap items-center gap-2">
                <el-tag size="small" type="info">{{ $t('db.planRowsEstimated', { rows: rowText(result.rows) }) }}</el-tag>
                <el-tag v-if="result.fullScan" size="small" type="danger">{{ $t('db.planFullScan') }}</el-tag>
                <el-tag v-if="result.usedIndexes?.length" size="small" type="success">
                    {{ $t('db.planUsedIndexes', { indexes: result.usedIndexes.join(', ') }) }}
                </el-tag>
                <span class="text-[12px] text-gray-400">{{ $t('db.planNotExecuted') }}</span>
            </div>

            <!-- 平铺不是「都是叶子」，而是该方言没给出父子关系：不说明就会被当成一棵单链树 -->
            <div v-if="!hasTree && nodes.length > 1" class="mb-1 text-[12px] text-gray-400">{{ $t('db.planFlatHint') }}</div>

            <div class="plan-nodes">
                <div
                    v-for="row in rows"
                    :key="row.index"
                    class="plan-node"
                    :class="{ 'is-full-scan': row.node.fullScan }"
                    :style="{ marginLeft: `${row.depth * 16}px`, '--plan-heat': String(row.heat) }"
                >
                    <span class="plan-node__kind">{{ row.node.nodeKind || '-' }}</span>
                    <span v-if="row.node.table" class="plan-node__table">{{ row.node.table }}</span>
                    <span v-if="row.node.indexName" class="plan-node__index">{{ row.node.indexName }}</span>
                    <span class="plan-node__rows">{{ $t('db.planNodeRows', { rows: rowText(row.node.estRows) }) }}</span>
                    <span v-if="row.droppedEdge" class="plan-node__warn">{{ $t('db.planDroppedEdge') }}</span>
                </div>
            </div>

            <div v-for="warn in result.warnings ?? []" :key="warn" class="mt-1 text-[12px] text-[var(--el-color-warning)]">{{ warn }}</div>
        </template>

        <el-empty v-else :description="$t('db.planNone')" :image-size="60" />
    </div>
</template>

<script lang="ts" setup>
/**
 * 单条语句的执行计划视图。
 *
 * 呈现的全部事实来自后端：哪个节点算全表扫描、行数是估算还是真实、有没有父子关系，
 * 都由方言与后端给出，这里不按 nodeKind 文本再判一遍（那会复制出一份只对本方言正确的判定）。
 * 行热区只是相对深浅（本计划内最大估算行数归一），刻意不带绝对数值语义。
 */
import { computed } from 'vue';
import type { SqlExplainPlan } from '../types';
import { flattenPlan, maxPlanRows, planHasTree, planHeat } from './planModel';

const props = defineProps<{ plan: SqlExplainPlan }>();

const result = computed(() => props.plan.result);
const nodes = computed(() => result.value?.plan ?? []);
const hasTree = computed(() => planHasTree(nodes.value));

const rows = computed(() => {
    const list = nodes.value;
    const max = maxPlanRows(list);
    return flattenPlan(list).map((row) => ({ ...row, heat: max > 0 ? planHeat(row.node, list) : 0 }));
});

/** 行数是浮点估算值：整数直接显示，小数保留 1 位，避免出现 1234.5678 这种假精确 */
function rowText(rows: number): string {
    if (!rows) {
        return '0';
    }
    return Number.isInteger(rows) ? rows.toLocaleString() : rows.toFixed(1);
}
</script>

<style lang="scss" scoped>
.plan-nodes {
    max-height: 360px;
    overflow: auto;
}

.plan-node {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 3px 8px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    font-size: 12px;
    line-height: 18px;

    // 热区底色由 --plan-heat 驱动（0~1 的相对强度），随主题主色派生，不写死色值
    background-color: color-mix(in srgb, var(--el-color-primary) calc(var(--plan-heat, 0) * 14%), transparent);

    &__kind {
        min-width: 96px;
        font-family: var(--el-font-family-mono, ui-monospace, monospace);
        color: var(--el-text-color-primary);
    }

    &__table {
        color: var(--el-text-color-regular);
    }

    &__index {
        color: var(--el-color-success);
    }

    &__rows {
        margin-left: auto;
        color: var(--el-text-color-secondary);
        white-space: nowrap;
    }

    &__warn {
        color: var(--el-color-warning);
        white-space: nowrap;
    }

    // 全表扫描的判定来自方言，这里只负责让它显眼
    &.is-full-scan .plan-node__kind {
        color: var(--el-color-danger);
        font-weight: 600;
    }
}
</style>
