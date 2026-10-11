import { beforeEach, describe, expect, it, vi } from 'vitest';

// getTextWidth 依赖真实布局引擎（offsetWidth），单测环境量不到宽度，mock 成确定性函数：每个字符 10px；
// vi.fn 包装以便断言调用参数（如超长文本不量宽的性能短路）
vi.mock('@/common/utils/string', () => ({
    getTextWidth: vi.fn((str: string) => str.length * 10),
}));

import EnumValue from '@/common/Enum';
import { getTextWidth } from '@/common/utils/string';
import { TableColumn } from '../index';

describe('TableColumn.setSlotText', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('链式返回且登记 measureContentWidth', () => {
        const col = TableColumn.new('targetTable', 'db.targetDbTable');
        const ret = col.setSlotText((row: { a: string }) => row.a);
        expect(ret).toBe(col);
        expect(col.measureContentWidth).toBeTypeOf('function');
    });

    it('按提取文本量宽（不叠加额外占位，渲染占位由 DOM 回填精修）', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb').setSlotText((row: { name: string }) => row.name);
        // mayfly_uib_it 13 字符 × 10px
        expect(col.measureContentWidth!({ name: 'mayfly_uib_it' })).toBe(130);
    });

    it('插槽渲染形态与原始字段不同时按渲染文本计宽（dbName.tableName 场景）', () => {
        const col = TableColumn.new('targetTable', 'db.targetDbTable').setSlotText(
            (row: { dbName: string; tableName: string }) => `${row.dbName}.${row.tableName}`
        );
        // 渲染文本 "mayfly_pg_it.t_bs" 17 字符 × 10；行数据里不存在 targetTable 字段也不影响
        expect(col.measureContentWidth!({ dbName: 'mayfly_pg_it', tableName: 't_bs' })).toBe(170);
    });

    it('autoCalculateMinWidth 采用 measureContentWidth 结果，与表头宽取大者并受上限约束', () => {
        const col = TableColumn.new('updFieldVal', 'db.syncCursor').setSlotText((row: { v: string }) => row.v);
        // 内容 20 字符 × 10 + 30 = 230；表头 'db.syncCursor' 13 字符 × 10 + 60 = 190 → 取 230
        col.autoCalculateMinWidth([{ v: 'x'.repeat(20) }, { v: 'short' }]);
        expect(col.minWidth).toBe(230);
    });

    it('未声明 setSlotText 的插槽列退回按原始字段值计宽', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb').isSlot();
        col.autoCalculateMinWidth([{ srcDb: 'mayfly_uib_it' }]);
        // 13 字符 × 10 + 30 = 160；表头 8 字符 × 10 + 60 = 140 → 取 160
        expect(col.minWidth).toBe(160);
    });
});

describe('TableColumn.refineMinWidthByDom', () => {
    it('实测内容宽(+内边距)大于当前 minWidth 时向上回填', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        col.minWidth = 100;
        // 文本估算 100，DOM 实测内容 100 → 100+30=130 > 100 → 回填
        col.refineMinWidthByDom(100);
        expect(col.minWidth).toBe(130);
    });

    it('实测小于当前 minWidth 时不缩小，仅向上修正', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        col.minWidth = 200;
        col.refineMinWidthByDom(100);
        expect(col.minWidth).toBe(200);
    });

    it('width 已固化时向上修正 width 而非 minWidth（固化列生效约束是 width）', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        col.minWidth = 180;
        col.width = 180;
        col.refineMinWidthByDom(200);
        // 200+30=230 > 180 → width 上修；minWidth 保持固化时的原值
        expect(col.width).toBe(230);
        expect(col.minWidth).toBe(180);
    });

    it('setFlexible 标记弹性列', () => {
        expect(TableColumn.new('name', 'common.name').flexible).toBe(false);
        expect(TableColumn.new('name', 'common.name').setFlexible().flexible).toBe(true);
    });

    it('受自动计宽上限约束，且固定宽列的大值不被实测覆盖', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        col.minWidth = 500;
        // 1000+30 → 上限 400 < 500 → 保持不动
        col.refineMinWidthByDom(1000);
        expect(col.minWidth).toBe(500);
    });

    it('隐藏态量到 0 不写脏值', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        col.minWidth = 100;
        col.refineMinWidthByDom(0);
        expect(col.minWidth).toBe(100);
    });

    it('显式字符串 minWidth 是调用方明确意图，不被回填覆盖', () => {
        const strCol = TableColumn.new('x', 'x');
        strCol.minWidth = '10%';
        strCol.refineMinWidthByDom(50);
        expect(strCol.minWidth).toBe('10%');
    });

    it('minWidth 未初始化时按 0 起步回填', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb');
        expect(col.minWidth).toBeUndefined();
        col.refineMinWidthByDom(50);
        expect(col.minWidth).toBe(80);
    });
});

describe('TableColumn.autoCalculateMinWidth', () => {
    beforeEach(() => {
        vi.clearAllMocks();
    });

    it('超长文本（>200 字符）直接按上限封顶，不再逐条量宽', () => {
        const col = TableColumn.new('sql', 'common.sql');
        col.autoCalculateMinWidth([{ sql: 'x'.repeat(500) }]);
        expect(col.minWidth).toBe(400);
        // 性能短路：超长文本不得进入逐字符量宽
        const longTextCalls = vi.mocked(getTextWidth).mock.calls.filter(([str]) => str.length > 200);
        expect(longTextCalls).toHaveLength(0);
    });

    it('空数据不写列宽', () => {
        const col = TableColumn.new('name', 'common.name');
        expect(col.autoCalculateMinWidth([])).toBe(0);
        expect(col.minWidth).toBeUndefined();
    });

    it('对象值按 JSON 序列化文本计宽', () => {
        const col = TableColumn.new('ext', 'common.ext');
        // '{"a":1}' 7 字符 → 7*10+30=100；表头 'common.ext' 10 字符 → 10*10+60=160 → 取大者
        col.autoCalculateMinWidth([{ ext: { a: 1 } }]);
        expect(col.minWidth).toBe(160);
    });

    it('tag 列按枚举翻译后的标签文本 + 标签外框计宽', () => {
        const col = TableColumn.new('status', 'common.status').typeTag({ enabled: new EnumValue(1, 'Enabled') });
        // 'Enabled' 7 字符 × 10 + tag 外框 22
        expect(col.measureContentWidth!({ status: 1 })).toBe(92);
    });

    it('tag 列未知枚举值按空文本计宽（仅标签外框）', () => {
        const col = TableColumn.new('status', 'common.status').typeTag({ enabled: new EnumValue(1, 'Enabled') });
        expect(col.measureContentWidth!({ status: 999 })).toBe(22);
    });

    it('autoCalculateMinWidthByHeader 仅按表头估宽，固化列同步向上写 width', () => {
        const col = TableColumn.new('name', 'test.colName');
        col.autoCalculateMinWidthByHeader();
        // 'test.colName' 12 字符 → 120+60=180，内容未参与
        expect(col.minWidth).toBe(180);

        const fixed = TableColumn.new('prop', 'test.colProp');
        fixed.width = 100; // 模拟已固化的紧凑列
        fixed.autoCalculateMinWidthByHeader();
        // 表头估宽 180 > 固化 100 → width 上修，minWidth 同步
        expect(fixed.minWidth).toBe(180);
        expect(fixed.width).toBe(180);
    });

    it('autoCalculateMinWidth 对已固化列同步向上写 width', () => {
        const col = TableColumn.new('name', 'test.colName');
        col.width = 100;
        col.autoCalculateMinWidth([{ name: 'x'.repeat(20) }]);
        // 内容 20 字符 → 230 > 固化 100 → width 上修 230
        expect(col.width).toBe(230);
        expect(col.minWidth).toBe(230);
    });
});
