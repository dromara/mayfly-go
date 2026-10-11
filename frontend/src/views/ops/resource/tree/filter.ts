import { matchPrefixSubsequence } from '@/common/utils/string';
import { i18n } from '@/i18n';

import { ERROR_KIND, LOADING_KIND, type TreeNode } from './types';

/** 标签的高亮片段：text 为原文切片，hit 表示该片段是否由查询词命中 */
export interface LabelSegment {
    text: string;
    hit: boolean;
}

/** 懒加载骨架的 UI 伪节点（占位/错误行）：不是资源，永不参与命中 */
const isPseudoNode = (node: { kind?: unknown }): boolean => node.kind === LOADING_KIND || node.kind === ERROR_KIND;

/**
 * 节点展示文本：默认 `t(label)`，贡献者可用 `displayLabel` 字段覆盖。
 *
 * 为什么默认要翻译：label 既可能是原始名称（表名、实例名），也可能是 i18n key（'db.table' 等菜单节点），
 * 而行渲染走的是 $t(label)。过滤拿裸 key 做子序列匹配会出现「搜中文『表』搜不到、而 key 里的字母 tab 反而命中」。
 *
 * 为什么是数据字段而不是贡献者函数入 register：加载节点时贡献者手里就握着原始 VO，算好一段文本是它本有的职责；
 * 放在数据上能让过滤/高亮/计数/行渲染四条路径取到同一个值（它们全调本函数），且核心依旧零 kind 知识。
 */
export function nodeDisplayLabel(node: { label?: unknown; displayLabel?: unknown }): string {
    const override = node.displayLabel;
    if (typeof override === 'string' && override) return override;

    const raw = String(node.label ?? '');
    // 不含 '.' 的串不可能是 i18n key（字典顶层全是命名空间），直接返回：
    // 过滤谓词与计数都是逐节点热路径，这一步省掉表名/主机名等绝大多数节点的一次消息解析
    if (!raw.includes('.')) return raw;
    return i18n.global.t(raw);
}

/**
 * 查询词在展示文本上的命中下标（未命中返回 null）。
 *
 * 空查询返回 null 而非「全命中」：本函数只服务高亮与计数（「没有词就没有命中可言」），
 * 引擎过滤谓词对空查询必须放行所有节点，见 nodeMatchesQuery。
 */
export function matchQueryOnText(query: string, text: string): number[] | null {
    return query ? matchPrefixSubsequence(query, text, true) : null;
}

/**
 * 资源树搜索谓词（前缀子序列匹配展示文本，与树库无关，由容器接线到引擎 filter）。
 *
 * 占位/错误行是懒加载水合层的 UI 伪节点，其 label 存的是 i18n key（如 'common.processing'），
 * 必须先于文本匹配排除：搜索词恰为该 key 的子序列时（如 'org' ⊆ 'common.processing'），
 * el-tree-v2 的 doFilter 会把命中的占位节点整条父链标记为展开，使未水合的节点在搜索时
 * 集体停在「加载中」且不发任何请求（引擎内部展开态不回传业务层，无人 hydrate）。
 * 伪节点非真实资源，本就不该被搜到。
 *
 * 空查询放行所有节点：语义是「不过滤」，其展开副作用由 VirtualTree 清空过滤后复位受控展开态抵消。
 */
export function nodeMatchesQuery(query: string, node: { kind?: unknown; label?: unknown; displayLabel?: unknown }): boolean {
    if (!query) return true;
    if (isPseudoNode(node)) return false;
    return matchQueryOnText(query, nodeDisplayLabel(node)) !== null;
}

/**
 * 命中片段切分（供行内标色）：无查询词或未命中返回 null，调用方据此回落纯文本渲染。
 *
 * 与 nodeMatchesQuery 共用 matchQueryOnText，保证「被标色的字」就是「让这个节点留下的字」。
 */
export function labelHitSegments(query: string, text: string): LabelSegment[] | null {
    const hits = matchQueryOnText(query, text);
    if (!hits?.length) return null;

    const segments: LabelSegment[] = [];
    let cursor = 0;
    for (const index of hits) {
        if (index > cursor) {
            segments.push({ text: text.slice(cursor, index), hit: false });
        }
        // 相邻命中字符合并进同一片段，少切几段 DOM
        const last = segments[segments.length - 1];
        if (last?.hit) {
            last.text += text[index];
        } else {
            segments.push({ text: text[index], hit: true });
        }
        cursor = index + 1;
    }
    if (cursor < text.length) {
        segments.push({ text: text.slice(cursor), hit: false });
    }
    return segments;
}

/**
 * 视图级附加可见性判定（如「按资源类型筛选」）：与文本命中 AND 组合。
 *
 * 谓词由宿主页面注入（容器与 filter 都不认识具体资产语义）；计数、行序、引擎过滤三处必须
 * 用同一个组合判定，否则会出现「计数说 N 个，Enter 只能跳 M 个」。
 */
export type NodeVisibilityFilter = (node: {
    kind?: unknown;
    label?: unknown;
    displayLabel?: unknown;
    parentKey?: unknown;
    params?: unknown;
}) => boolean;

/** 文本命中 + 视图级过滤的组合判定（伪节点已由 nodeMatchesQuery 排除） */
function hitOf(query: string, extra?: NodeVisibilityFilter): (node: Parameters<NodeVisibilityFilter>[0]) => boolean {
    return (node) => nodeMatchesQuery(query, node) && (extra?.(node) ?? true);
}

/**
 * 收集「业务展开链上可见」且命中查询词的节点 key，按文档序（即界面自上而下的顺序）。
 *
 * 为何只沿展开集下钻（而不是数整棵已水合树）：引擎的 flattenTree 只包含「祖先都已在
 * expandedKeySet 里」的节点，而 doFilter 替命中节点强开的那份展开态活不过下一次 data 变化
 * （useTree  watch data 会 `setExpandedKeys(defaultExpandedKeys)` 把受控集刷回去）。
 * 把收起分支的命中也计进来，会得到一个数不上、也跳不到的虚高计数：
 * 实测「Enter 逐跳」走到这类命中时 `flattenTree.indexOf(node)` 返回 -1，
 * `scrollToItem(-1)` 不仅静默跳空，还会把列表 scrollTop 复位成 0（把用户看的位置也弄丢）。
 *
 * 未水合（收起即释放子树）的节点不在 data 里，同样算不到——这就是「只能搜已加载资源」的边界，
 * 靠计数而非 placeholder 文案说清楚。
 */
export function collectMatchKeys(
    query: string,
    nodes: TreeNode[],
    expandedKeys: Set<string>,
    extra?: NodeVisibilityFilter,
    acc: string[] = [],
): string[] {
    const hit = hitOf(query, extra);
    if (!query) return acc;
    for (const node of nodes) {
        if (isPseudoNode(node)) continue;
        if (hit(node)) {
            acc.push(node.key);
        }
        // 只有父节点在展开集里，它的子级才在 flattenTree 里（才看得到、才跳得到）
        if (node.children?.length && expandedKeys.has(node.key)) {
            collectMatchKeys(query, node.children, expandedKeys, extra, acc);
        }
    }
    return acc;
}

/**
 * 收集引擎在该查询词下真的会渲染的行 key（文档序），与 el-tree-v2 的 flattenTree 等价。
 *
 * 规则抄自 useFilter.doFilter + useTree 的 flattenTree：
 * - 只下钻已展开节点（flattenTree 按 expandedKeySet 决定是否 push 子级）；
 * - 无过滤：展开链上每一行都渲染，含占位/错误行（它们本来就是界面上看得见的一行，算行序不能漏）；
 * - 有过滤：只有「自身命中」或「后代里有命中」的节点不被隐藏，命中节点的未命中子级整体消失。
 *
 * 为何业务层要自己再算一份行序：定位跳转需要「目标行是第几行」，而引擎的 scrollToItem
 * 在过滤态下对合法下标也会返回错位偏移（实测 32 行列表 idx=26 → scrollTop 0，
 * 同公式无过滤的 113 行列表 idx=69 → 1502 正确），偏移只能由持有行序与行高的一侧自己算。
 */
export function collectVisibleRowKeys(
    query: string,
    nodes: TreeNode[],
    expandedKeys: Set<string>,
    extra?: NodeVisibilityFilter,
    acc: string[] = [],
): string[] {
    const hit = hitOf(query, extra);
    // 「有过滤」= 有查询词 **或** 有视图级判定：引擎两者都会隐藏行，行序必须同样两者都算，
    // 否则只选类型不搜字时 visibleRowKeys 会多算被隐藏的行，scrollToIndex 落到错行
    const filtering = !!query || !!extra;
    for (const node of nodes) {
        const childRows =
            node.children?.length && expandedKeys.has(node.key) ? collectVisibleRowKeys(query, node.children, expandedKeys, extra) : [];
        // 完全无过滤时引擎渲染展开链上的每一行（含占位/错误行）；有过滤时只留命中及其祖先链
        const selfVisible = filtering ? hit(node) && !isPseudoNode(node) : true;
        if (selfVisible || childRows.length) {
            acc.push(node.key);
            acc.push(...childRows);
        }
    }
    return acc;
}

/**
 * 下一个跳转位序（纯函数，供查找条 n/N 与循环跳转共用）。
 *
 * cursor 为 -1 表示「还没跳过，位置在第 1 个之前」：此时向前应绕到最后一个，
 * 而不是按 (cursor + delta) 的算式落在倒数第二个。total 为 0 时回 -1（无可跳转）。
 */
export function nextMatchIndex(cursor: number, total: number, delta: number): number {
    if (total <= 0) return -1;
    const from = cursor >= 0 ? Math.min(cursor, total - 1) : delta > 0 ? -1 : total;
    return (from + delta + total) % total;
}
