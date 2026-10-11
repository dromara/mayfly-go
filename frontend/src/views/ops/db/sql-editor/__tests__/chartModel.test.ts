import { describe, expect, it } from 'vitest';
import { CHART_POINT_LIMIT, buildChartOption, isMeasureColumn, measureColumns } from '../chartModel';

/**
 * 结果集成图模型的守卫
 *
 * 存在理由：图比表格更容易说谎。把缺失值补成 0、把全 NULL 列画成一条零线、
 * 只画前 N 行却不说明，这三种都会让一张「看着没问题」的图给出错误结论。
 */

const rows = (...items: Record<string, unknown>[]): Record<string, unknown>[] => items;

describe('度量列判定', () => {
    it('非空值全为数字才算度量，含数字字符串（驱动回传的 bigint/decimal 形态）', () => {
        expect(isMeasureColumn(rows({ n: 1 }, { n: '2' }), 'n')).toBe(true);
        expect(isMeasureColumn(rows({ n: 1 }, { n: 'abc' }), 'n')).toBe(false);
    });

    it('全 NULL 的列不算度量（画出来是一条零线，等于把没数据说成数据都是 0）', () => {
        expect(isMeasureColumn(rows({ n: null }, { n: undefined }), 'n')).toBe(false);
    });

    it('度量列保持结果集列序，不按名称重排', () => {
        const list = measureColumns(['b', 'a', 'name'], rows({ b: 1, a: 2, name: 'x' }));
        expect(list).toEqual(['b', 'a']);
    });
});

describe('buildChartOption', () => {
    it('缺分类列、缺度量列或没有行时返回 null，由界面说明原因而不是画空坐标系', () => {
        const data = rows({ name: 'a', n: 1 });
        expect(buildChartOption('bar', '', ['n'], data)).toBeNull();
        expect(buildChartOption('bar', 'name', [], data)).toBeNull();
        expect(buildChartOption('bar', 'name', ['n'], [])).toBeNull();
    });

    it('柱/折线：缺失值留 null 不补 0，NULL 分类标签写 NULL', () => {
        const built = buildChartOption('bar', 'name', ['n'], rows({ name: null, n: 5 }, { name: 'b', n: null }));
        expect(built).not.toBeNull();
        const option = built!.option as { xAxis: { data: string[] }; series: { data: (number | null)[] }[] };
        expect(option.xAxis.data).toEqual(['NULL', 'b']);
        expect(option.series[0].data).toEqual([5, null]);
    });

    it('饼图只取第一个度量，且无值的行不进饼（0 是真实占比）', () => {
        const built = buildChartOption('pie', 'name', ['n', 'm'], rows({ name: 'a', n: 1, m: 9 }, { name: 'b', n: null, m: 8 }));
        const option = built!.option as { series: { data: { name: string; value: number }[] }[] };
        expect(built!.usedMeasures).toEqual(['n']);
        expect(option.series[0].data).toEqual([{ name: 'a', value: 1 }]);
    });

    it('超过上限只画前 N 条，同时回报总行数（界面据此出提示）', () => {
        const many = Array.from({ length: CHART_POINT_LIMIT + 30 }, (_, i) => ({ name: `n${i}`, v: i }));
        const built = buildChartOption('line', 'name', ['v'], many);
        expect(built!.points).toBe(CHART_POINT_LIMIT);
        expect(built!.total).toBe(many.length);
    });

    it('不改动传入的行数据', () => {
        const data = rows({ name: 'a', n: 1 });
        buildChartOption('bar', 'name', ['n'], data);
        expect(data).toEqual([{ name: 'a', n: 1 }]);
    });
});
