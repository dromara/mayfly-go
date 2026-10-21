import { beforeEach, describe, expect, it, vi } from 'vitest';

// getTextWidth 依赖真实布局引擎（offsetWidth），单测环境量不到宽度，mock 成确定性函数：每个字符 10px；
// vi.fn 包装以便断言调用参数（如超长文本不量宽的性能短路）
vi.mock('@/common/utils/string', () => ({
    getTextWidth: vi.fn((str: string) => str.length * 10),
}));

// 行内操作的权限过滤不能依赖登录态：约定权限码以 ':denied' 结尾即视为无权限
vi.mock('@/components/auth/auth', () => ({
    hasPerm: (code: string) => !code.endsWith(':denied'),
}));

import EnumValue from '@/common/Enum';
import { getTextWidth } from '@/common/utils/string';
import { i18n } from '@/i18n';
import { TableColumn, planRowActions, isIconOnlyAction, actionTooltip, actionsColumnWidth, actionColumnProps, type ActionItem } from '../index';

/** 与 data-table/index.ts 里的 ACTION_ICON_WIDTH 同值（带 icon 的按钮额外占位） */
const ACTION_ICON = 20;

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
        // 内容 20 字符 × 10 + 30 = 230；表头 'db.syncCursor' 13 字符 × 10 + 30 = 160 → 取 230
        col.autoCalculateMinWidth([{ v: 'x'.repeat(20) }, { v: 'short' }]);
        expect(col.minWidth).toBe(230);
    });

    it('未声明 setSlotText 的插槽列退回按原始字段值计宽', () => {
        const col = TableColumn.new('srcDb', 'db.srcDb').isSlot();
        col.autoCalculateMinWidth([{ srcDb: 'mayfly_uib_it' }]);
        // 13 字符 × 10 + 30 = 160；表头 8 字符 × 10 + 30 = 110 → 取 160
        expect(col.minWidth).toBe(160);
    });

    it('操作列：与行数据无关的常量按钮文案也参与计宽', () => {
        // 行数据里没有 action 字段，若按原始字段值过滤空值则该列永远量不到内容，
        // 只能靠表头宽打底，而列宽首轮定型即冻结（首页无按钮行就会把该列永久算窄）
        const col = TableColumn.new('action', 'test.op')
            .isSlot()
            .setSlotText(() => 'x'.repeat(20));
        col.autoCalculateMinWidth([{ fileKey: 'a.zip' }]);
        // 内容 20 字符 → 200+30=230 > 表头 'test.op' 7 字符 → 130
        expect(col.minWidth).toBe(230);
    });
});

describe('TableColumn.setAddWidth', () => {
    it('链式返回并登记 addWidth', () => {
        const col = TableColumn.new('status', 'common.status');
        expect(col.setAddWidth(20)).toBe(col);
        expect(col.addWidth).toBe(20);
    });

    it('autoCalculateMinWidth 在封顶宽度之上叠加 addWidth（为定宽控件预留）', () => {
        // 内容 'mayfly_uib_it' 13 字符 ×10 + 30 = 160；表头 'test.colName' 12 字符 → 150；取大 160，+addWidth 20 = 180
        const col = TableColumn.new('name', 'test.colName').setAddWidth(20);
        col.autoCalculateMinWidth([{ name: 'mayfly_uib_it' }]);
        expect(col.minWidth).toBe(180);
    });

    it('addWidth 在 MAX_AUTO_WIDTH 封顶之后叠加（不占用内容计宽上限）', () => {
        // 内容 50 字符 ×10 + 30 = 530 > 400 → 封顶 400，再 +addWidth 20 = 420
        const col = TableColumn.new('name', 'test.colName').setAddWidth(20);
        col.autoCalculateMinWidth([{ name: 'x'.repeat(50) }]);
        expect(col.minWidth).toBe(420);
    });
});

describe('planRowActions / TableColumn.setActions', () => {
    /** .el-button.is-link 的左右内边距，每格都要计入（不计会整体少算 n×4，刚好导致最后一格折行） */
    const PAD = 4;

    /** 构造 n 格操作项；denied 下标的格带 ':denied' 权限码（视为无权限） */
    const items = (n: number, denied: number[] = []) =>
        Array.from({ length: n }, (_, i) => ({
            key: `k${i}`,
            label: `test.action${i}`, // 12 字符 → 120px，每格再 +PAD
            perm: denied.includes(i) ? 'a:denied' : 'a:ok',
            onClick: () => undefined,
        }));

    it('未超内联上限不折叠；折完只剩一条也不折叠', () => {
        expect(planRowActions(items(3)).folded).toHaveLength(0);
        // 4 格：超出 3 但尾巴只有一条 → 不折（「⋯ 里塞一个动作」等于把功能藏起来）
        const four = planRowActions(items(4));
        expect(four.folded).toHaveLength(0);
        expect(four.inline).toHaveLength(4);
        // 5 格：折 2 条进「更多」
        expect(planRowActions(items(5)).folded).toHaveLength(2);
    });

    it('无权限的按钮不参与渲染也不占宽度（受限账号不为看不到的按钮留宽）', () => {
        // 5 格里第 3、4 格无权限 → 实际可见 3 格 → 不折叠
        const plan = planRowActions(items(5, [3, 4]));
        expect(plan.inline).toHaveLength(3);
        expect(plan.folded).toHaveLength(0);
    });

    it('计宽按折叠后的实际格子：3 直显 + 1 格「更多」（下拉箭头占位另计）', () => {
        const col = TableColumn.new('action', 'common.operation').setActions(items(5));
        const moreWidth = i18n.global.t('common.more').length * 10;
        expect(col.measureContentWidth!({})).toBe((120 + PAD) * 3 + moreWidth + PAD + ACTION_ICON + 3 * 12);
    });

    it('不折叠时不计「更多」格，也不计图标占位', () => {
        const col = TableColumn.new('action', 'common.operation').setActions(items(2));
        expect(col.measureContentWidth!({})).toBe((120 + PAD) * 2 + 12);
    });

    it('声明了 icon 的按钮计入图标占位宽', () => {
        const col = TableColumn.new('action', 'common.operation').setActions([{ key: 'k', label: 'test.action0', icon: 'delete', onClick: () => undefined }]);
        expect(col.measureContentWidth!({})).toBe(120 + ACTION_ICON + PAD);
    });

    it('互斥组只按最宽一格计宽', () => {
        const col = TableColumn.new('action', 'common.operation').setActions([
            { key: 'a', label: 'test.short', group: 'state', onClick: () => undefined }, // 10 字符 → 100
            { key: 'b', label: 'test.longerr', group: 'state', onClick: () => undefined }, // 12 字符 → 120，同组只留它
            { key: 'c', label: 'test.action0', onClick: () => undefined }, // 120
        ]);
        // 互斥组收敛成两格（同组留最宽的 'test.longerr'）：(120+4) + (120+4) + 1 个间距 12
        expect(col.measureContentWidth!({})).toBe((120 + PAD) + (120 + PAD) + 12);
    });

    it('图标按钮不计文案宽', () => {
        const col = TableColumn.new('action', 'common.operation').setActions([
            { key: 'k', label: 'test.action0', icon: 'delete', iconOnly: true, onClick: () => undefined },
        ]);
        expect(col.measureContentWidth!({})).toBe(ACTION_ICON + PAD);
    });

    it('直显上限存到列上：计宽与渲染读同一个值，不会一边折一边不折', () => {
        const list = items(5);
        const col = TableColumn.new('action', 'common.operation').setActions(list, { maxInline: 4 });
        expect(col.actions).toBe(list);
        expect(col.actionsOptions?.maxInline).toBe(4);
        // 上限抬到 4 → 5 格只折一条 → 按规则不折，宽就是 5 格（文案+内边距）+ 4 个间距（不再多计「更多」那一格）
        expect(col.measureContentWidth!({})).toBe((120 + PAD) * 5 + 4 * 12);
    });

    it('列级图标模式：整列只按图标格计宽，不再算文案', () => {
        const col = TableColumn.new('action', 'common.operation').setActions(
            [
                { key: 'a', label: 'test.action0', icon: 'edit', onClick: () => undefined },
                { key: 'b', label: 'test.action1', icon: 'delete', onClick: () => undefined },
            ],
            { iconOnly: true }
        );
        expect(col.actionsOptions?.iconOnly).toBe(true);
        // 两格都只算图标位 20 + 内边距 4，加 1 个间距 12
        expect(col.measureContentWidth!({})).toBe((ACTION_ICON + PAD) * 2 + 12);
    });

    it('没图标的格不得进图标模式：否则渲染出完全空白的隐形按钮，计宽也必须按文字算', () => {
        expect(isIconOnlyAction({ key: 'a', label: 'test.action0', onClick: () => undefined }, { iconOnly: true })).toBe(false);
        expect(isIconOnlyAction({ key: 'a', label: 'test.action0', icon: 'edit', onClick: () => undefined }, { iconOnly: true })).toBe(true);
        expect(isIconOnlyAction({ key: 'a', label: 'test.action0', icon: 'edit', iconOnly: true, onClick: () => undefined })).toBe(true);
    });

    it('图标模式下的折叠格：「更多」触发器不参与图标模式，仍按文字+箭头计', () => {
        const iconItems: ActionItem[] = items(5).map((item) => ({ ...item, icon: 'edit' }));
        const col = TableColumn.new('action', 'common.operation').setActions(iconItems, { iconOnly: true });
        const moreWidth = i18n.global.t('common.more').length * 10;
        // 3 格图标（20+4）+ 更多（文案 + 箭头 20 + 4）+ 3 个间距 36
        expect(col.measureContentWidth!({})).toBe((ACTION_ICON + PAD) * 3 + moreWidth + ACTION_ICON + PAD + 3 * 12);
    });

    it('带边框按钮按 size 计入真实内边距（link 4px / small 22px / default 30px / large 38px）', () => {
        const one = [{ key: 'k', label: 'test.action0', onClick: () => undefined }];
        expect(actionsColumnWidth(one)).toBe(120 + PAD);
        expect(actionsColumnWidth(one, { appearance: 'bordered' })).toBe(120 + 30);
        expect(actionsColumnWidth(one, { appearance: 'plain', size: 'small' })).toBe(120 + 22);
        expect(actionsColumnWidth(one, { appearance: 'text', size: 'large' })).toBe(120 + 38);
    });

    it('裸表格接缝：actionColumnProps 与 setActions 用同一宽度算法，并自己补上单元格内边距', () => {
        const list = items(2);
        const props = actionColumnProps(list);
        // 两格（120+4）+ 1 个间距 12 + 单元格内边距 30
        expect(props.minWidth).toBe(actionsColumnWidth(list) + 30);
        // 不能断言成 'common.operation'：那是没翻译的裸键，而本返回值会直接 v-bind 到列头，
        // 把裸键当期望等于把缺陷钉成基线。现在钉两件事：取的是字典译文，且结果不再长得像个键
        expect(props.label).toBe(i18n.global.t('common.operation'));
        expect(props.label).not.toMatch(/^[a-z][\w]*\.[\w.]+$/);
        expect(props.label).not.toBe('common.operation');
        expect(props.align).toBe('center');
        expect(props.fixed).toBe('right');
        // 面板表靠右不固定时可选关掉 fixed
        expect(actionColumnProps(list, { fixed: false }).fixed).toBeUndefined();
    });

    it('纯图标格没写 tooltip 时拿文案当提示（图标语义不能让用户猜）', () => {
        const iconOnlyItem: ActionItem = { key: 'a', label: 'test.action0', icon: 'edit', iconOnly: true, onClick: () => undefined };
        expect(actionTooltip(iconOnlyItem)).toBe('test.action0');
        // 显式提示优先；带文字的格不强制提示
        expect(actionTooltip({ ...iconOnlyItem, tooltip: 'common.edit' })).toBe('common.edit');
        expect(actionTooltip({ key: 'b', label: 'test.action1', onClick: () => undefined })).toBeUndefined();
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
        // '{"a":1}' 7 字符 → 7*10+30=100；表头 'common.ext' 10 字符 → 10*10+30=130 → 取大者
        col.autoCalculateMinWidth([{ ext: { a: 1 } }]);
        expect(col.minWidth).toBe(130);
    });

    it('tag 列按枚举翻译后的标签文本 + 标签外框计宽', () => {
        const col = TableColumn.new('status', 'common.status').typeTag({ enabled: new EnumValue(1, 'Enabled') });
        // 'Enabled' 7 字符 × 10 + tag 外框 22
        expect(col.measureContentWidth!({ status: 1 })).toBe(92);
    });

    // 计宽必须跟着渲染走：EnumTag 对「有值但未登记展示元数据」刻意回显原始值（后端目录新增一项
    // 不该被渲成红「-」），旧用例却把「按空文本计宽」当期望钉住——那等于把双真源写成规范，
    // 实际这类列会渲出一长串原始值而只预留标签外框宽
    it('tag 列未知枚举值按回显的原始值计宽（与 EnumTag 渲染同源）', () => {
        const col = TableColumn.new('status', 'common.status').typeTag({ enabled: new EnumValue(1, 'Enabled') });
        // '999' 3 字符 × 10 + tag 外框 22
        expect(col.measureContentWidth!({ status: 999 })).toBe(52);
    });

    it('tag 列空值不计宽（整格不渲染标签，再算外框会无端撑出一段空白）', () => {
        const col = TableColumn.new('status', 'common.status').typeTag({ enabled: new EnumValue(1, 'Enabled') });
        expect(col.measureContentWidth!({ status: '' })).toBe(0);
        expect(col.measureContentWidth!({ status: null })).toBe(0);
        expect(col.measureContentWidth!({})).toBe(0);
    });

    it('autoCalculateMinWidthByHeader 仅按表头估宽，固化列同步向上写 width', () => {
        const col = TableColumn.new('name', 'test.colName');
        col.autoCalculateMinWidthByHeader();
        // 'test.colName' 12 字符 → 120+30=150，内容未参与
        expect(col.minWidth).toBe(150);

        const fixed = TableColumn.new('prop', 'test.colProp');
        fixed.width = 100; // 模拟已固化的紧凑列
        fixed.autoCalculateMinWidthByHeader();
        // 表头估宽 150 > 固化 100 → width 上修，minWidth 同步
        expect(fixed.minWidth).toBe(150);
        expect(fixed.width).toBe(150);
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
