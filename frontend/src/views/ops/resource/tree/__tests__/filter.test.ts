import { describe, expect, it, vi } from 'vitest';

// 只 mock 一小份词典：本文件要钉的是「匹配用翻译后的文本」这条规则，
// 依赖真实字典会把「词条被改写」这种无关变更也变成红灯
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => ({ 'db.table': '表', 'db.view': '视图' })[key] ?? key } },
}));

import { collectMatchKeys, collectVisibleRowKeys, labelHitSegments, nextMatchIndex, nodeDisplayLabel, nodeMatchesQuery } from '../filter';
import { ERROR_KIND, LOADING_KIND, type TreeNode } from '../types';

/**
 * 资源树搜索契约单测：谓词（谁留下）、片段（哪几个字命中）、计数（哪些命中真的可见）三件套。
 *
 * 三者必须出自同一次匹配实现，因此也在同一份用例里互相印证：
 * 界面上标了色的字，就是让这行没被过滤掉的字；计数报几个，Enter 就能跳几个。
 */

/** 真实资源节点（label 为原始名称，i18n 取不到 key 时原样返回） */
const resourceNode = (key: string, label: string, children: TreeNode[] = []): TreeNode => ({
    key,
    kind: 'table',
    label,
    params: {},
    hasChildren: children.length > 0,
    children,
});

describe('resource/tree/filter nodeMatchesQuery', () => {
    it('空查询放行所有节点（含伪节点，交由引擎不过滤）', () => {
        expect(nodeMatchesQuery('', { kind: LOADING_KIND, label: 'common.processing' })).toBe(true);
        expect(nodeMatchesQuery('', { kind: 'db', label: 'mysql' })).toBe(true);
    });

    it('占位/错误伪节点永不命中，即使查询词是其 i18n key 的子序列', () => {
        // 'org' ⊆ 'common.processing'、'com' ⊆ 'common.retry'：修复前会被误命中并撑开父链
        expect(nodeMatchesQuery('org', { kind: LOADING_KIND, label: 'common.processing' })).toBe(false);
        expect(nodeMatchesQuery('com', { kind: ERROR_KIND, label: 'common.retry' })).toBe(false);
    });

    it('真实节点按前缀子序列匹配（排除只针对伪 kind，不误伤真实命中）', () => {
        // 'org' ⊆ 't_flow_rule_segment'（o→r→g 依序出现）
        expect(nodeMatchesQuery('org', { kind: 'table', label: 't_flow_rule_segment' })).toBe(true);
        expect(nodeMatchesQuery('zzz', { kind: 'table', label: 't_flow_rule_segment' })).toBe(false);
        // 未水合的真实节点若 label 不含查询词，本就不命中，与占位排除无关
        expect(nodeMatchesQuery('org', { kind: 'db', label: 'local_pgsql' })).toBe(false);
    });

    it('匹配对象是界面展示文本，不是裸 i18n key', () => {
        // 'db.table' 在界面上是「表」：搜中文要能命中，而 key 里的字母不该反过来命中
        expect(nodeDisplayLabel({ label: 'db.table' })).toBe('表');
        expect(nodeMatchesQuery('表', { kind: 'db-table-menu', label: 'db.table' })).toBe(true);
        expect(nodeMatchesQuery('tab', { kind: 'db-table-menu', label: 'db.table' })).toBe(false);
    });

    it('贡献者声明 displayLabel 时，行渲染/过滤/高亮/计数跟着它走同一个值', () => {
        // 这是开闭口：新增资产渲染另一套文本时只需声明字段，不必去改核心谓词或加 kind 分支
        const node = { kind: 'table', label: 't_user', displayLabel: '用户表' };
        expect(nodeDisplayLabel(node)).toBe('用户表');
        expect(nodeMatchesQuery('用户', node)).toBe(true);
        expect(nodeMatchesQuery('t_user', node)).toBe(false);
        expect(labelHitSegments('用户', nodeDisplayLabel(node))).toEqual([{ text: '用户', hit: true }, { text: '表', hit: false }]);
    });

    it('大小写不敏感（表名/实例名多为小写，用户输入大写不该零命中）', () => {
        expect(nodeMatchesQuery('LOCAL', { kind: 'db', label: 'local_pgsql' })).toBe(true);
        expect(nodeMatchesQuery('Local', { kind: 'db', label: 'local_pgsql' })).toBe(true);
        expect(nodeMatchesQuery('pgsql', { kind: 'db', label: 'local_pgsql' })).toBe(true);
    });
});

describe('resource/tree/filter labelHitSegments', () => {
    it('按命中下标切段，相邻命中字符合并进同一段', () => {
        // 'uname' ⊆ 'username'：u@0，name@4..7
        expect(labelHitSegments('uname', 'username')).toEqual([
            { text: 'u', hit: true },
            { text: 'ser', hit: false },
            { text: 'name', hit: true },
        ]);
    });

    it('片段拼回原文，大小写按原文呈现（不改写用户看到的字）', () => {
        const segments = labelHitSegments('US', 'username') ?? [];
        expect(segments.map((s) => s.text).join('')).toBe('username');
        expect(segments[0]).toEqual({ text: 'us', hit: true });
    });

    it('无查询词或未命中返回 null（调用方回落纯文本，不产生多余节点）', () => {
        expect(labelHitSegments('', 'username')).toBeNull();
        expect(labelHitSegments('zzz', 'username')).toBeNull();
    });
});

describe('resource/tree/filter collectMatchKeys', () => {
    it('空查询不收集', () => {
        expect(collectMatchKeys('', [resourceNode('a', 'ab')], new Set())).toEqual([]);
    });

    it('按文档序返回命中 key，父先于子', () => {
        const tree = [resourceNode('root', 'order', [resourceNode('root.a', 't_order'), resourceNode('root.b', 'order_id')])];
        expect(collectMatchKeys('order', tree, new Set(['root']))).toEqual(['root', 'root.a', 'root.b']);
    });

    it('收起分支的命中不计入：不在引擎 flattenTree 里的节点，跳到时 indexOf 为 -1 会静默跳空', () => {
        // 计数口径必须跟「真能跳到的集合」一致，否则「N 个命中」与 Enter 可跳个数对不上
        const collapsed = resourceNode('ac', 'ac', [resourceNode('ac.x', 'abx')]);
        const tree = [resourceNode('a', 'root', [resourceNode('ab', 'ab'), collapsed, resourceNode('bb', 'bb')])];
        // 'ac' 收起 → 其命中子级 'ac.x' 看不到也跳不到，不计
        expect(collectMatchKeys('ab', tree, new Set(['a']))).toEqual(['ab']);
        // 展开后它才真的在可见行里
        expect(collectMatchKeys('ab', tree, new Set(['a', 'ac']))).toEqual(['ab', 'ac.x']);
    });

    it('伪节点永不计入命中', () => {
        const loading: TreeNode = { key: 'l', kind: LOADING_KIND, label: 'common.processing', params: {}, hasChildren: false };
        const tree = [resourceNode('a', 'a', [loading])];
        expect(collectMatchKeys('org', tree, new Set(['a']))).toEqual([]);
    });

    it('视图级判定（extra）与文本命中 AND 组合：计数不会把被筛掉的类型算进去', () => {
        const tree = [resourceNode('m1', 'machine-a'), resourceNode('d1', 'machine-b')];
        const onlyMachines = (node: { kind?: unknown }) => node.kind === 'table';
        expect(collectMatchKeys('machine', tree, new Set(), onlyMachines)).toEqual(['m1', 'd1']);
        // 换成只看另一种类型：同一文本命中的行全部不算可跳转命中
        expect(collectMatchKeys('machine', tree, new Set(), (node) => node.kind === 'other')).toEqual([]);
    });
});

describe('resource/tree/filter collectVisibleRowKeys', () => {
    const loading: TreeNode = { key: 'a__loading', kind: LOADING_KIND, label: 'common.processing', params: {}, hasChildren: false };

    it('无过滤：展开链上每一行都算，含懒加载占位行（漏了它行序就会整体上移一行）', () => {
        const tree = [resourceNode('a', 'alpha', [resourceNode('a.b', 'beta'), loading])];
        expect(collectVisibleRowKeys('', tree, new Set(['a']))).toEqual(['a', 'a.b', 'a__loading']);
    });

    it('无过滤但父未展开：不收集其子级', () => {
        const tree = [resourceNode('a', 'alpha', [resourceNode('a.b', 'beta')])];
        expect(collectVisibleRowKeys('', tree, new Set())).toEqual(['a']);
    });

    it('有过滤：只保留命中节点及其祖先链，未命中兄弟与命中节点的未命中子级都不算行', () => {
        const menu = resourceNode('m', 'db.table', [resourceNode('m.hit', 't_alert'), resourceNode('m.miss', 'other')]);
        const tree = [resourceNode('root', 'root', [menu, resourceNode('x', 'zzz')])];
        expect(collectVisibleRowKeys('alert', tree, new Set(['root', 'm']))).toEqual(['root', 'm', 'm.hit']);
    });

    it('有过滤时占位/错误行永不混举行序（引擎谓词也不让它们命中）', () => {
        const tree = [resourceNode('a', 'alpha', [loading, resourceNode('a.b', 't_alert')])];
        expect(collectVisibleRowKeys('alert', tree, new Set(['a']))).toEqual(['a', 'a.b']);
    });

    it('视图级判定同样作用于行序：被筛掉的类型既不占行也不能被跳到', () => {
        const tree = [resourceNode('m', 't_alert'), resourceNode('d', 't_alert_db')];
        const rows = (extra?: (node: { kind?: unknown }) => boolean) => collectVisibleRowKeys('t_alert', tree, new Set(), extra);
        expect(rows()).toEqual(['m', 'd']);
        expect(rows((node) => node.kind !== 'table')).toEqual([]);
    });
});

describe('resource/tree/filter nextMatchIndex', () => {
    it('无命中回 -1（无可跳转）', () => {
        expect(nextMatchIndex(-1, 0, 1)).toBe(-1);
        expect(nextMatchIndex(2, 0, -1)).toBe(-1);
    });

    it('首次向前从第 1 个开始，向后绕到最后一个', () => {
        expect(nextMatchIndex(-1, 5, 1)).toBe(0);
        expect(nextMatchIndex(-1, 5, -1)).toBe(4);
    });

    it('中段双向环绕', () => {
        expect(nextMatchIndex(4, 5, 1)).toBe(0);
        expect(nextMatchIndex(0, 5, -1)).toBe(4);
        expect(nextMatchIndex(2, 5, 1)).toBe(3);
    });

    it('命中集收缩后旧位序先钳回末尾再走（不会越界取到 undefined）', () => {
        // 只剩 3 个命中时位序 9 已越界：向前应从头循环，而不是跳到不存在的第 10 个
        expect(nextMatchIndex(9, 3, 1)).toBe(0);
        expect(nextMatchIndex(9, 3, -1)).toBe(1);
    });
});
