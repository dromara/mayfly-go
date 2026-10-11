import type { ECOption } from '@/components/echarts/config';

/**
 * 查询结果集 → 图表配置的模型。
 *
 * 两条口径写在这里而不是散在组件里：
 * - 点数上限：超过上限只画前 N 条并回报，不回报就等于用一张局部图冒充全貌；
 * - 度量列判定：一列只有在「所有非空值都是数字」时才算度量，混进文本的列不静默按 0 画
 *   （0 是真实数值，画出来会被读成「这些行确实是 0」）。
 */

export type ChartKind = 'bar' | 'line' | 'pie';

/** 单次成图最多呈现的数据点数 */
export const CHART_POINT_LIMIT = 200;

/** 值为 NULL 的分类标签：留白会被看成图坏了，写 NULL 才是库里的原话 */
const NULL_LABEL = 'NULL';

/** 该值能否当数字用（驱动常把 bigint/decimal 回传成字符串形态） */
function numericValue(value: unknown): number | null {
    if (value === null || value === undefined || value === '') {
        return null;
    }
    if (typeof value === 'number') {
        return Number.isFinite(value) ? value : null;
    }
    if (typeof value === 'string') {
        const trimmed = value.trim();
        if (!trimmed) {
            return null;
        }
        const parsed = Number(trimmed);
        return Number.isFinite(parsed) ? parsed : null;
    }
    return null;
}

/**
 * 该列是否可作度量：非空值全部可当数字。
 *
 * 一列非空值都没有时返回 false：全 NULL 的列画出来是一条零线，那是把「没有数据」画成「数据都是 0」
 */
export function isMeasureColumn(rows: Record<string, unknown>[], key: string): boolean {
    let nonNull = 0;
    for (const row of rows) {
        const raw = row[key];
        if (raw === null || raw === undefined || raw === '') {
            continue;
        }
        nonNull++;
        if (numericValue(raw) === null) {
            return false;
        }
    }
    return nonNull > 0;
}

/** 可作度量的列（保持结果集的列顺序，不按名称重排） */
export function measureColumns(columns: string[], rows: Record<string, unknown>[]): string[] {
    return columns.filter((column) => isMeasureColumn(rows, column));
}

export interface ChartBuild {
    option: ECOption;
    /** 实际参与成图的点数 */
    points: number;
    /** 结果集总行数，用于判断是否被上限截断 */
    total: number;
    /** 真正成图的度量列（饼图只用第一个） */
    usedMeasures: string[];
}

/**
 * 组装 echarts option。
 *
 * category 为空、或没有任何数值度量时返回 null，由界面给出「为什么成不了图」，
 * 而不是渲染一张空坐标系让人以为查询没数据。
 */
export function buildChartOption(kind: ChartKind, category: string, measures: string[], rows: Record<string, unknown>[]): ChartBuild | null {
    const used = kind === 'pie' ? measures.slice(0, 1) : measures;
    if (!category || !used.length || !rows.length) {
        return null;
    }

    const limited = rows.slice(0, CHART_POINT_LIMIT);
    const labels = limited.map((row) => {
        const raw = row[category];
        return raw === null || raw === undefined || raw === '' ? NULL_LABEL : String(raw);
    });

    if (kind === 'pie') {
        const measure = used[0];
        const data: { name: string; value: number }[] = [];
        limited.forEach((row, index) => {
            const value = numericValue(row[measure]);
            // 无值的行直接不入饼：0 是一个真实占比，拿它填位会多出一个看不见的扇区
            if (value !== null) {
                data.push({ name: labels[index], value });
            }
        });
        return {
            option: {
                // appendToBody：tooltip 挂到 body 层，不被结果面板的 overflow 裁切、也不被上方编辑器盖住
                tooltip: { trigger: 'item', appendToBody: true },
                legend: { top: 8, type: 'scroll' },
                series: [{ type: 'pie', radius: ['30%', '60%'], top: 24, data }],
            },
            points: data.length,
            total: rows.length,
            usedMeasures: [measure],
        };
    }

    const series = used.map((measure) => ({
        type: kind,
        name: measure,
        // 缺失值留空而非补 0：折线断点/柱子缺位才是「这里没有值」的正确表达
        data: limited.map((row) => numericValue(row[measure])),
    }));

    return {
        option: {
            // appendToBody：tooltip 挂到 body 层，不被结果面板的 overflow 裁切、也不被上方编辑器盖住
            tooltip: { trigger: 'axis', appendToBody: true },
            legend: { top: 8, type: 'scroll' },
            grid: { top: 44, left: 60, right: 24, bottom: 32, containLabel: true },
            xAxis: { type: 'category', data: labels },
            yAxis: { type: 'value' },
            series,
        },
        points: limited.length,
        total: rows.length,
        usedMeasures: used,
    };
}
