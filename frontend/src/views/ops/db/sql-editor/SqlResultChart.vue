<template>
    <div class="sql-result-chart">
        <div class="mb-1 flex flex-wrap items-center gap-2">
            <el-radio-group v-model="kind" size="small">
                <el-radio-button value="bar">{{ $t('db.chartBar') }}</el-radio-button>
                <el-radio-button value="line">{{ $t('db.chartLine') }}</el-radio-button>
                <el-radio-button value="pie">{{ $t('db.chartPie') }}</el-radio-button>
            </el-radio-group>

            <el-select v-model="category" size="small" style="width: 200px" :placeholder="$t('db.chartCategoryPlaceholder')">
                <el-option v-for="col in columns" :key="col.key" :value="col.key" :label="col.label" />
            </el-select>

            <el-select
                v-if="kind !== 'pie'"
                v-model="measures"
                multiple
                collapse-tags
                size="small"
                style="width: 260px"
                :placeholder="$t('db.chartMeasurePlaceholder')"
            >
                <el-option v-for="col in measureCols" :key="col.key" :value="col.key" :label="col.label" />
            </el-select>
        </div>

        <!-- 成不了图/被上限截断都要说出口，静默画一张局部图是在骗人 -->
        <div v-if="notice" class="mb-1 text-[12px] text-[var(--el-color-warning)]">{{ notice }}</div>

        <ECharts v-if="build && active" :option="build.option" class="sql-result-chart__canvas" />
        <el-empty v-else :description="$t('db.chartNoData')" :image-size="60" />
    </div>
</template>

<script lang="ts" setup>
/**
 * 查询结果的图表视图（柱/折/饼 + 轴选择）。
 *
 * 由结果页签按需异步加载：echarts 体积大，静态引入会把它拽进 SQL 编辑器的首屏 chunk
 * （与 monaco 同一类边界，守卫见 db/__tests__/monacoBoundary.test.ts 的同款思路）。
 * 取数一律用列的 key，展示一律用 label：结果集里存在「标题≠数据键」的合成列。
 */
import { computed, ref, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import ECharts from '@/components/echarts/ECharts.vue';
import { CHART_POINT_LIMIT, buildChartOption, measureColumns, type ChartKind } from './chartModel';

const props = defineProps<{
    /** 结果集列：key 用于取行数据，label 用于展示 */
    columns: { key: string; label: string }[];
    rows: Record<string, unknown>[];
    /**
     * 该结果页签是否为当前激活页签。非激活时 el-tab-pane 处于 display:none，容器宽高为 0，
     * 此时初始化 ECharts 会得到 0x0 画布，切回时靠 ResizeObserver 的防抖 resize 才恢复，
     * 出现「先塌一下再正常」。故仅在激活时挂载 ECharts，保证 init 时容器已是真实尺寸。
     * 图表配置（kind/列选择）仍存于本组件（随页签 v-show 常驻），切换不丢失。
     */
    active: boolean;
}>();

const { t } = useI18n();

const kind = ref<ChartKind>('bar');

/** 可作度量的列（非空值全为数字），保持结果集列序 */
const measureCols = computed(() => {
    const usable = new Set(
        measureColumns(
            props.columns.map((col) => col.key),
            props.rows
        )
    );
    return props.columns.filter((col) => usable.has(col.key));
});

/** 分类列默认取第一个非度量列（没有就取第一列） */
const defaultCategory = computed(() => {
    const measureKeys = new Set(measureCols.value.map((col) => col.key));
    return props.columns.find((col) => !measureKeys.has(col.key))?.key ?? props.columns[0]?.key ?? '';
});

const category = ref('');
const measures = ref<string[]>([]);

// 结果集变化（切换页签/重新执行）时重算默认选择，但保留用户已选且仍然有效的列
watch(
    [() => props.rows, () => props.columns],
    () => {
        if (!category.value || !props.columns.some((col) => col.key === category.value)) {
            category.value = defaultCategory.value;
        }
        const usable = measureCols.value.map((col) => col.key);
        measures.value = measures.value.filter((key) => usable.includes(key));
        if (!measures.value.length) {
            measures.value = usable.slice(0, 2);
        }
    },
    { immediate: true }
);

const build = computed(() => buildChartOption(kind.value, category.value, measures.value, props.rows));

const notice = computed(() => {
    if (!build.value) {
        return '';
    }
    if (kind.value === 'pie' && measures.value.length > 1) {
        // 饼图只能表达一个度量，不说明会让人以为其余列被一起画进来了
        return t('db.chartPieSingleMeasure');
    }
    if (build.value.total > build.value.points) {
        return t('db.chartPointLimited', { limit: String(CHART_POINT_LIMIT), total: String(build.value.total) });
    }
    return '';
});
</script>

<style lang="scss" scoped>
.sql-result-chart {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
}

/* ECharts 以容器尺寸初始化画布，这里必须给出可计算的高度，否则画布是 0x0 */
.sql-result-chart__canvas {
    flex: 1;
    min-height: 200px;
}
</style>
