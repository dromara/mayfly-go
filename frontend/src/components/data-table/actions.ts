import { getTextWidth } from '@/common/utils/string';
import { hasPerm } from '@/components/auth/auth';
import { i18n } from '@/i18n';
import { CELL_PADDING } from './constants';

// 操作列格与格之间的间距：由 RowActions 的 flex gap 读（模板内联变量注入），计宽按同一值累加
export const ACTION_BUTTON_GAP = 12;
const ACTION_ICON_WIDTH = 20; // el-button 带 icon 时图标本身 + 与文字的间距
const ACTION_LINK_PADDING = 4; // .el-button.is-link 的 padding: 2px（左右共 4px）
const ACTION_INLINE_MAX = 3; // 行内最多直显几格操作按钮，其余折进「更多」
/** 折叠触发器的文案 key：计宽与渲染都读它，两处不会各自写一份字面量 */
export const ACTION_MORE_LABEL = 'common.more';

/**
 * 行内操作项：由表格统一渲染（RowActions.vue），列宽也从同一份声明推出来。
 *
 * 为什么不让页面自己写按钮模板 + 宽度：两者必然分叉——文案改了宽度没改，按钮被硬裁成半个词；
 * 而按钮是 .cell 的 inline-flex 子元素，被裁时连省略号都不出。收进这里之后，
 * 「渲染什么」与「占多宽」是同一个数组的两种投影，权限/折叠规则也只需写一遍。
 *
 * 新增一种形态（新按钮样式/新占位）只需往本接口加可选字段并在计宽与渲染两处各读它一次，
 * 已有页面无需改动（开闭原则）。
 *
 * 泛型 Row = 该列的行数据类型：默认 `any`（不标注时与旧行为一致，零波及）；页面在 `setActions<MachineVO>([...])`
 * 处绑定一次，本列所有行回调（show/disabled/loading/onClick/labelParams/confirm.params）即获得真实行类型与补全。
 * 行类型只在「定义处」参与校验，存进 TableColumn.actions 后按 `ActionItem<any>` 擦除（渲染期行数据本就是动态形状）。
 */
export interface ActionItem<Row = any> {
    /** 唯一标识（折叠菜单的 command 值） */
    key: string;
    /** 按钮文案的 i18n key */
    label: string;
    /** 文案插值参数（按行生成），如「RDP(新标签页)」这类组合文案。计宽按文案模板原文量，带占位符时实际更宽，由渲染后实测回填补上 */
    labelParams?: (row: Row) => Record<string, unknown>;
    type?: 'primary' | 'success' | 'warning' | 'danger' | 'info';
    /** Element Plus 图标名（已全局注册）：kebab-case / PascalCase 都可，如 'edit'、'video-play' */
    icon?: string;
    /** 只显图标（需有 icon，否则自动退回文字）：文案降为 aria-label/tooltip，不计入列宽 */
    iconOnly?: boolean;
    /** 悬浮提示的 i18n key（图标按钮靠它自解释） */
    tooltip?: string;
    /** 账号级权限码：无权限则既不渲染也不参与计宽（受限账号不必为看不到的按钮留宽） */
    perm?: string;
    /**
     * 互斥分组名：同组按钮不会在同一行同时出现（如按状态切「停止/运行」、按模式切「单机/集群信息」）。
     * 计宽时同组只取最宽的一格，避免把整列算宽一格；哪个更长与语言有关，交给计宽时再量，页面不自行猜。
     */
    group?: string;
    /** 行级显隐：参与计宽（任一行可能显示就得留宽），只影响该行渲染 */
    show?: (row: Row) => boolean;
    disabled?: (row: Row) => boolean;
    /** 行级进行中状态（如终端回放）：渲染为按钮 loading */
    loading?: (row: Row) => boolean;
    /**
     * 危险/不可逆动作的确认：title 为 i18n key，params 按行生成插值参数。
     * pop=true 用就地气泡确认（el-popconfirm）；折叠进「更多」的项固定用确认框（菜单里弹不下气泡）。
     */
    confirm?: { title: string; params?: (row: Row) => Record<string, unknown>; pop?: boolean };
    /** event 仅直显按钮有值（如在新窗口按点击位置开终端）；折叠菜单项为 undefined */
    onClick: (row: Row, event?: MouseEvent) => void;
}

/** 本行可见且有权操作按声明顺序取前 maxInline 格直显，其余折进「更多」 */
export type RowActionPlan = { inline: ActionItem[]; folded: ActionItem[] };

/**
 * 行内操作折叠方案：先按账号权限过滤，再把超出内联上限的次要动作折进「更多」。
 * 入参应是「本行可能显示的候选」（调用方先过行级 show）：先折叠后过滤会把「运行」这类主动作藏进菜单。
 *
 * 而若折完只剩一条，就不折了全部直显——「⋯ 里塞一个动作」等于把功能藏起来。
 */
export function planRowActions(items: ActionItem[], maxInline = ACTION_INLINE_MAX): RowActionPlan {
    const available = items.filter((item) => !item.perm || hasPerm(item.perm));
    if (available.length <= maxInline) {
        return { inline: available, folded: [] };
    }
    const folded = available.slice(maxInline);
    return folded.length === 1 ? { inline: available, folded: [] } : { inline: available.slice(0, maxInline), folded };
}

/**
 * 计宽用的候选收敛：同 group 的互斥按钮只保留最宽的一格（按声明位置保留，不影响其他格顺序）。
 * 渲染不走这里——渲染按行数据决定显示哪个，计宽要的是「最坏情况」。
 */
function widestPerGroup(items: ActionItem[], options: ActionOptions = {}): ActionItem[] {
    const widest = new Map<string, ActionItem>();
    for (const item of items) {
        if (!item.group) continue;
        const current = widest.get(item.group);
        if (!current || actionCellWidth(item, options) > actionCellWidth(current, options)) {
            widest.set(item.group, item);
        }
    }
    return items.filter((item) => !item.group || widest.get(item.group) === item);
}

/**
 * 按钮外观（对齐 Element Plus 的四种写法）：
 * link＝无边框文字链（表格行内惯例）；text＝无边框但 hover 有底色；
 * plain＝浅底描边按钮；bordered＝默认实心/纯色按钮。
 */
export type ActionAppearance = 'link' | 'text' | 'plain' | 'bordered';
export type ActionSize = 'large' | 'default' | 'small';

/** `setActions` / 裸表格的展示选项；以后加展示维度只往这里加字段，不再扩位置参数与逐层 prop */
export interface ActionOptions {
    /** 直显上限，超出折进「更多」（默认 ACTION_INLINE_MAX） */
    maxInline?: number;
    /** 整列收成纯图标 + 悬停提示（单个按钮要图标名时在本格写 icon，无需整列开关） */
    iconOnly?: boolean;
    /** 按钮尺寸（面板表常用 small）；同时影响渲染与占位宽 */
    size?: ActionSize;
    /** 按钮外观，默认 link */
    appearance?: ActionAppearance;
}

/**
 * el-button 的左右内边距合计（Element Plus 事实）：
 * link 是 `padding: 2px`，其余按 size 取 `8px 15px` / `5px 11px` / `12px 19px`。
 * 不计内边距会整体少算 n×(4~30)：link 三格差 12px 刚好把最后一格顶到第二行，
 * 带边框按钮一格就差 30px。
 */
const BUTTON_PADDING_X: Record<ActionSize, number> = { large: 19, default: 15, small: 11 };

function actionButtonPadding(options: ActionOptions): number {
    return !options.appearance || options.appearance === 'link' ? ACTION_LINK_PADDING : BUTTON_PADDING_X[options.size ?? 'default'] * 2;
}

/**
 * 这一格最终是否只显图标：声明了 iconOnly 且本格确实有 icon——没图标时退回文字，
 * 否则会渲染出一个完全空白的“隐形按钮”。计宽与渲染必须走同一个判定，
 * 不然会“按文字算宽、按图标渲染”白留一截，或反过来把文字挤掉。
 */
export function isIconOnlyAction(item: ActionItem, options: ActionOptions = {}): boolean {
    return (!!options.iconOnly || !!item.iconOnly) && !!item.icon;
}

/** 纯图标格必须有自解释提示：没显式声明 tooltip 就拿文案当提示（图标语义不能让用户猜） */
export function actionTooltip(item: ActionItem, options: ActionOptions = {}): string | undefined {
    return item.tooltip ?? (isIconOnlyAction(item, options) ? item.label : undefined);
}

/** 一格按钮的占位宽：只显图标时不算文案，但图标位与按钮内边距照计 */
function actionCellWidth(item: ActionItem, options: ActionOptions): number {
    const iconOnly = isIconOnlyAction(item, options);
    const text = iconOnly ? 0 : maxLabelWidth(item.label);
    // small 按钮字距比正文窄，但这里仍按正文字号量：宁偏宽不可裁切（裸表格没有实测回填兜底）
    return text + (item.icon || iconOnly ? ACTION_ICON_WIDTH : 0) + actionButtonPadding(options);
}

/**
 * 操作列的内容需求宽（不含单元格内边距）。
 * DataTable 的 measureContentWidth 与裸 `el-table-column` 共用同一个算法——
 * 宽度只能有一份算法，否则两条路径下同一组按钮会算出两个宽度。
 */
export function actionsColumnWidth(items: ActionItem[], options: ActionOptions = {}): number {
    const { inline, folded } = planRowActions(widestPerGroup(items, options), options.maxInline ?? ACTION_INLINE_MAX);
    const cellsWidth = inline.reduce((sum, cell, index) => sum + actionCellWidth(cell, options) + (index ? ACTION_BUTTON_GAP : 0), 0);
    if (!folded.length) {
        return cellsWidth;
    }
    // 多出的「更多」那一格永远带文字+下拉箭头（触发器不参与图标模式，否则整列收图标时算不出它的宽）
    return cellsWidth + ACTION_BUTTON_GAP + maxLabelWidth(ACTION_MORE_LABEL) + ACTION_ICON_WIDTH + actionButtonPadding({});
}

/**
 * 不走 DataTable 的裸 `el-table-column`（面板表/抽屉表）用这个直接 v-bind 出操作列属性，
 * 单元格里再放 `<RowActions :items="同一份 items" :data="scope.row" />`。
 * 与 DataTable 的区别：这里要自己把单元格内边距算进去（DataTable 由 autoCalculateMinWidth 补）。
 */
export function actionColumnProps(
    items: ActionItem[],
    options: ActionOptions & { label?: string; fixed?: boolean } = {}
): { label: string; minWidth: number; align: string; fixed: string | undefined } {
    return {
        // 默认列头必须过翻译：这里以前给的是裸键 'common.operation'，而返回值是直接 v-bind 到
        // el-table-column 的 label 上的，结果所有用本接缝的裸表格列头直接显示英文原词
        // （本文件算按钮宽度时倒是用了 i18n.global.t，唯独这个默认值漏了）。
        // 调用方显式传的 label 视为已解析好的展示文本，不在这里二次翻译
        label: options.label ?? i18n.global.t('common.operation'),
        minWidth: actionsColumnWidth(items, options) + CELL_PADDING,
        align: 'center',
        fixed: options.fixed === false ? undefined : 'right',
    };
}

/** 一格按钮的需求宽：互斥候选（数组）取最宽的那个，单文案直接量 */
function maxLabelWidth(button: string | string[]): number {
    const keys = Array.isArray(button) ? button : [button];
    return keys.reduce((max, key) => Math.max(max, getTextWidth(i18n.global.t(key))), 0);
}
