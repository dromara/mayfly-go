<template>
    <div class="alert-condition-edit">
        <el-select v-model="localCondition.operator" :placeholder="$t('alert.conditionOperator')" style="width: 150px; margin-bottom: 10px">
            <el-option v-for="item in AlertOperatorEnumValues" :key="item.value" :label="$t(item.label)" :value="item.value" />
        </el-select>

        <div v-for="(item, index) in localCondition.items" :key="index" class="condition-item">
            <el-select v-model="item.metric" :placeholder="$t('alert.metric')" style="width: 140px" @change="onMetricChange(item)">
                <el-option v-for="m in metrics" :key="m.key" :label="$t(m.label)" :value="m.key" />
            </el-select>
            <el-select v-model="item.compare" :placeholder="$t('alert.compare')" style="width: 90px">
                <el-option v-for="c in AlertCompareEnumValues" :key="c.value" :label="c.label" :value="c.value" />
            </el-select>
            <el-input-number
                v-model="item.value"
                :placeholder="$t('alert.threshold')"
                :precision="metricPrecision(item.metric)"
                v-bind="metricBounds(item.metric)"
                :controls="false"
                style="width: 120px"
            />
            <el-input-number v-model="item.duration" :placeholder="$t('alert.duration')" :min="0" :step="30" :controls="false" style="width: 110px" />
            <el-button type="danger" link @click="removeItem(index)">{{ $t('alert.delete') }}</el-button>
        </div>

        <div class="edit-tip">{{ tips }}</div>

        <el-button type="primary" link @click="addItem">+ {{ $t('alert.addCondition') }}</el-button>
    </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import type { AlertCondition, ConditionItem, MetricDefinition } from '../types';
import { AlertCompareEnum, AlertOperatorEnum } from '../enums';
import { resourceMetricsOf } from '../utils';

const props = defineProps<{
    modelValue: AlertCondition;
    resourceType?: number;
}>();

const emit = defineEmits<{
    (e: 'update:modelValue', value: AlertCondition): void;
}>();

const { t } = useI18n();

const AlertOperatorEnumValues = Object.values(AlertOperatorEnum);
const AlertCompareEnumValues = Object.values(AlertCompareEnum);

// 指标完全由后端按资源类型下发，不再兜底到机器指标：
// 写死机器指标会让 Redis/DB 等其他资源类型配置出永不生效的规则
const metrics = ref<MetricDefinition[]>([]);

const loadMetrics = async (resourceType?: number) => {
    metrics.value = await resourceMetricsOf(resourceType);
};

watch(
    () => props.resourceType,
    (val) => loadMetrics(val),
    { immediate: true }
);

function findMetric(metric: string): MetricDefinition | undefined {
    return metrics.value.find((m) => m.key === metric);
}

function metricPrecision(metric: string): number {
    return findMetric(metric)?.isPercent ? 2 : 0;
}

/** 无取值区间的指标不限制 min/max，交由后端校验 */
function metricBounds(metric: string): Record<string, number> {
    const def = findMetric(metric);
    return def?.hasRange ? { min: def.min, max: def.max } : {};
}

/** 切换指标后把阈值收敛回该指标的合法区间，避免残留越界阈值被后端拒绝 */
const onMetricChange = (item: ConditionItem) => {
    const def = findMetric(item.metric);
    if (!def?.hasRange) {
        return;
    }
    if (item.value < def.min) {
        item.value = def.min;
    } else if (item.value > def.max) {
        item.value = def.max;
    }
};

const localCondition = computed({
    get: () => props.modelValue || { operator: 'and', items: [] },
    set: (val) => emit('update:modelValue', val),
});

// 提示行：持续时间语义 + 首个条件项指标的阈值区间（避免逐项重复）
const tips = computed(() => {
    const def = findMetric(localCondition.value?.items?.[0]?.metric || '');
    if (!def?.hasRange) {
        return t('alert.durationTip');
    }
    return `${t('alert.durationTip')} | ${t('alert.metricRangeTip', { min: def.min, max: def.max })}`;
});

const addItem = () => {
    if (!localCondition.value.items) {
        localCondition.value.items = [];
    }
    const first = metrics.value[0];
    localCondition.value.items.push({
        metric: first?.key || '',
        compare: 'gt',
        value: first?.hasRange ? first.min : 0,
        duration: 0,
    } as ConditionItem);
};

const removeItem = (index: number) => {
    localCondition.value.items.splice(index, 1);
};
</script>

<style lang="scss" scoped>
.alert-condition-edit {
    .condition-item {
        display: flex;
        gap: 8px;
        margin-bottom: 8px;
        align-items: center;
    }

    .edit-tip {
        font-size: 12px;
        color: var(--el-text-color-secondary);
        margin-bottom: 8px;
    }
}
</style>
