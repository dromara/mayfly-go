import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as eleIcons from '@element-plus/icons-vue';
import en from '@/i18n/en/redis';
import zh from '@/i18n/zh-cn/redis';
import { commandNeedsConfirm, editingToken, isKeyArgument, splitCommand, suggestCommands, suggestKeys } from '../keyview/console';
import { insertKeysToTree, keysToTree } from '../utils';
import type { RedisCommandSpec } from '../types';
import { opSkins, viewSkins } from '../keyview/appearance';
import { columnValue, coversAll, formatTtlSeconds, hasMoreMembers, prefillForm, ttlCellText } from '../keyview/descriptor';
import type { RedisFormSchema, RedisKeyMember, RedisViewColumn } from '../types';

const commandRequestMock = vi.fn();
const scanRequestMock = vi.fn();
// commandCatalog / useKeyScan 内部依赖 redisApi，mock 掉 IO 才能单独测缓存与扫描续扫语义
vi.mock('../api', () => ({
    redisApi: {
        commands: { request: (...args: unknown[]) => commandRequestMock(...args) },
        scan: { request: (...args: unknown[]) => scanRequestMock(...args) },
    },
}));

/**
 * Redis 数据视角的跨语言契约守护。
 *
 * 后端 keyvalue 处理器用 i18n key 描述列名/按钮名，前端语言包负责出词：
 * 两侧各自新增而不同步时，界面会直接显示裸 key（如 redis.colFoo），这里在测试阶段就拦住
 */
const HANDLER_DIR = join(import.meta.dirname, '../../../../../../server/internal/redis/application/keyvalue');

const keys = (locale: { redis: Record<string, string> }) => new Set(Object.keys(locale.redis));

describe('命令目录的共享缓存', () => {
    const specs: RedisCommandSpec[] = [{ name: 'GET', arity: 2, flags: [], firstKey: 1, lastKey: 1, step: 1, needConfirm: false }];

    afterEach(() => {
        vi.clearAllMocks();
        // 模块级缓存是跨用例单例，reset 后动态 import 拿到干净实例
        vi.resetModules();
    });

    it('同一实例第二次读取直接命中缓存，不再发请求', async () => {
        commandRequestMock.mockResolvedValue(specs);
        const mod = await import('../keyview/commandCatalog');
        await mod.loadCommandCatalog(1, 0);
        await mod.loadCommandCatalog(1, 0);
        expect(commandRequestMock).toHaveBeenCalledTimes(1);
        expect(mod.cachedCommandCatalog(1)).toEqual(specs);
    });

    it('在途请求共享同一个 Promise，并发调用只发一次', async () => {
        let release!: (value: RedisCommandSpec[]) => void;
        commandRequestMock.mockReturnValue(new Promise((resolve) => (release = resolve)));
        const mod = await import('../keyview/commandCatalog');
        const first = mod.loadCommandCatalog(2, 0);
        const second = mod.loadCommandCatalog(2, 0);
        release(specs);
        await Promise.all([first, second]);
        expect(commandRequestMock).toHaveBeenCalledTimes(1);
    });

    it('拉取失败不写缓存，下次调用重新发起（可重试）', async () => {
        commandRequestMock.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce(specs);
        const mod = await import('../keyview/commandCatalog');
        await expect(mod.loadCommandCatalog(3, 0)).rejects.toThrow('boom');
        // 失败后、重试成功前，缓存必须仍然是空的
        expect(mod.cachedCommandCatalog(3)).toBeUndefined();
        await expect(mod.loadCommandCatalog(3, 0)).resolves.toEqual(specs);
        expect(commandRequestMock).toHaveBeenCalledTimes(2);
    });
});

describe('key 分组树', () => {
    const label = (node: { type?: number; name: string; keyCount?: number }) => `${node.type}:${node.name}:${node.keyCount ?? 1}`;

    it('按分隔符折叠目录并逐层统计 key 数（key 叶显示全名）', () => {
        const tree = keysToTree(['app:cache:a', 'app:cache:b', 'app:db', 'standalone'], ':', null);
        expect(tree.map(label)).toEqual(['1:app:3', '2:standalone:1']);
        expect(tree[0]?.children?.map(label)).toEqual(['1:cache:2', '2:app:db:1']);
    });

    it('目录与同名 key 共存时不互相吞并', () => {
        const tree = keysToTree(['aa', 'aa:x'], ':', null);
        expect(tree.filter((node) => node.type === 1).map((node) => node.name)).toEqual(['aa']);
        expect(tree.filter((node) => node.type === 2).map((node) => node.name)).toEqual(['aa']);
    });

    it('子段名恰为 keyNode 的目录不会被误判成 key 而丢子树', () => {
        const tree = keysToTree(['a:keyNode:x', 'a:keyNode:keyNode:y'], ':', null);
        const mid = tree[0]?.children?.find((node) => node.type === 1);
        expect(mid?.name).toBe('keyNode');
        expect(mid?.keyCount).toBe(2);
    });

    it('key 里出现 __proto__/constructor 段名不会污染对象原型', () => {
        const tree = keysToTree(['__proto__:x', 'constructor:y'], ':', null);
        expect(tree.map((node) => node.name).sort()).toEqual(['__proto__', 'constructor']);
        expect(Object.getOwnPropertyNames(Object.prototype).filter((name) => name.includes('`k`'))).toEqual([]);
    });

    it('展开过的目录内部按目录在前、key 在后排序', () => {
        const tree = keysToTree(['z:bb', 'z:aa', 'z:cc:x'], ':', new Set(['z:']));
        expect(tree[0]?.children?.map((node) => node.name)).toEqual(['cc', 'z:aa', 'z:bb']);
    });

    it('排序下沉数据层：未展开的目录也按序输出（不再依赖 DOM 补排）', () => {
        // openStatus 为 null（无目录展开）时，旧实现按插入序输出 [z:bb, z:aa, cc]，现在恒为有序
        const tree = keysToTree(['z:bb', 'z:aa', 'z:cc:x'], ':', null);
        expect(tree[0]?.children?.map((node) => node.name)).toEqual(['cc', 'z:aa', 'z:bb']);
    });
});

describe('key 分组树的增量插入', () => {
    const label = (node: { type?: number; name: string; keyCount?: number }) => `${node.type}:${node.name}:${node.keyCount ?? 1}`;

    it('落已有目录：插叶子并沿路 keyCount++，维持有序', () => {
        const tree = keysToTree(['app:b', 'app:d'], ':', null);
        insertKeysToTree(tree, ['app:c', 'app:a'], ':');
        expect(tree.map(label)).toEqual(['1:app:4']);
        expect(tree[0]?.children?.map(label)).toEqual(['2:app:a:1', '2:app:b:1', '2:app:c:1', '2:app:d:1']);
    });

    it('新路径：按序建目录链并逐层统计 keyCount', () => {
        const tree = keysToTree(['app:a'], ':', null);
        insertKeysToTree(tree, ['x:y:z'], ':');
        expect(tree.map(label)).toEqual(['1:app:1', '1:x:1']);
        expect(tree[1]?.children?.map(label)).toEqual(['1:y:1']);
        expect(tree[1]?.children?.[0]?.children?.map(label)).toEqual(['2:x:y:z:1']);
    });

    it('重复 key 不产生重复叶子，也不动 keyCount', () => {
        const tree = keysToTree(['app:a', 'app:b'], ':', null);
        insertKeysToTree(tree, ['app:a'], ':');
        expect(tree.map(label)).toEqual(['1:app:2']);
        expect(tree[0]?.children?.map(label)).toEqual(['2:app:a:1', '2:app:b:1']);
    });

    it('目录与同名 key 增量共存不互相吞并', () => {
        const tree = keysToTree(['aa'], ':', null);
        insertKeysToTree(tree, ['aa:x'], ':');
        expect(tree.filter((node) => node.type === 1).map((node) => node.name)).toEqual(['aa']);
        expect(tree.filter((node) => node.type === 2).map((node) => node.name)).toEqual(['aa']);
        expect(tree.find((node) => node.type === 1)?.children?.map(label)).toEqual(['2:aa:x:1']);
    });

    it('从空树增量插入与全量构建结果同序同形', () => {
        const sampleKeys = ['m:b', 'm:a', 'b:x', 'm:c:y'];
        const incremental = insertKeysToTree([], sampleKeys, ':');
        const full = keysToTree(sampleKeys, ':', null);
        expect(incremental.map(label)).toEqual(full.map(label));
        expect(incremental[1]?.children?.map(label)).toEqual(full[1]?.children?.map(label));
    });
});

describe('resolveScanCount 的分级', () => {
    it('浏览态（无搜索词）用小 count 尽快出首屏', async () => {
        const { resolveScanCount } = await import('../resource/composables/useKeyScan');
        expect(resolveScanCount('', 0, 'standalone')).toBe(250);
        expect(resolveScanCount('', 10000000, 'standalone')).toBe(250);
    });

    it('搜索态按库规模分级，超过阈值的大库封顶 2000', async () => {
        const { resolveScanCount } = await import('../resource/composables/useKeyScan');
        expect(resolveScanCount('user:*', 50000, 'standalone')).toBe(1000);
        expect(resolveScanCount('user:*', 100000, 'standalone')).toBe(1000);
        expect(resolveScanCount('user:*', 100001, 'standalone')).toBe(2000);
        expect(resolveScanCount('user:*', 10000000, 'standalone')).toBe(2000);
    });

    it('集群模式按 3 个 master 摊薄单次 count', async () => {
        const { resolveScanCount } = await import('../resource/composables/useKeyScan');
        expect(resolveScanCount('', 1000, 'cluster')).toBe(83);
        expect(resolveScanCount('user:*', 50000, 'cluster')).toBe(333);
        expect(resolveScanCount('user:*', 200000, 'cluster')).toBe(666);
    });
});

describe('scan 的稀疏首屏自动续扫', () => {
    /** 一批扫描响应：cursor 为 0 表示整库扫完 */
    const batch = (keys: string[], cursor: number, extra: Record<string, unknown> = {}) => ({
        keys,
        cursor: { '0': cursor },
        dbSize: 1000,
        summaries: [],
        ...extra,
    });

    /** 起一个已登记实例/库的扫描域（动态 import 避开 vi.mock 工厂的 TDZ） */
    async function setupScan(match = '') {
        const { useKeyScan } = await import('../resource/composables/useKeyScan');
        const domain = useKeyScan();
        domain.state.scanParam.id = 1;
        domain.state.scanParam.db = 0;
        domain.state.scanParam.mode = 'standalone';
        domain.state.scanParam.match = match;
        return domain;
    }

    afterEach(() => {
        scanRequestMock.mockReset();
    });

    it('首批就有数据只扫一次，并随批填充摘要', async () => {
        scanRequestMock.mockResolvedValueOnce(batch(['x'], 5, { dbSize: 100, summaries: [{ key: 'x', type: 'string', ttl: -1 }] }));
        const { state, scan } = await setupScan();
        await expect(scan(false)).resolves.toEqual(['x']);
        expect(scanRequestMock).toHaveBeenCalledTimes(1);
        expect(state.summaries['x']).toEqual({ key: 'x', type: 'string', ttl: -1 });
    });

    it('本批 0 key 且游标未归零时自动续扫，直到凑出数据', async () => {
        scanRequestMock
            .mockResolvedValueOnce(batch([], 17))
            .mockResolvedValueOnce(batch([], 34))
            .mockResolvedValueOnce(batch(['a', 'b'], 0));
        const { state, scan } = await setupScan('rare:*');
        await expect(scan(false)).resolves.toEqual(['a', 'b']);
        expect(scanRequestMock).toHaveBeenCalledTimes(3);
        expect(state.keys).toEqual(['a', 'b']);
    });

    it('连续空批有上限：首扫 + 最多 3 次续扫后停止', async () => {
        scanRequestMock.mockResolvedValue(batch([], 99));
        const { scan } = await setupScan('rare:*');
        await expect(scan(false)).resolves.toEqual([]);
        expect(scanRequestMock).toHaveBeenCalledTimes(4);
    });

    it('游标归零即停，不做无谓续扫', async () => {
        scanRequestMock.mockResolvedValueOnce(batch([], 0));
        const { scan } = await setupScan();
        await expect(scan(false)).resolves.toEqual([]);
        expect(scanRequestMock).toHaveBeenCalledTimes(1);
    });

    it('appendKey=true 把新批接到已有 key 之后', async () => {
        scanRequestMock.mockResolvedValueOnce(batch(['c'], 0));
        const { state, scan } = await setupScan();
        state.keys = ['a', 'b'];
        await expect(scan(true)).resolves.toEqual(['c']);
        expect(state.keys).toEqual(['a', 'b', 'c']);
    });
});

describe('视角描述符引用的文案必须两侧语言包都有', () => {
    const goKeys = new Set(
        readdirSync(HANDLER_DIR)
            .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
            .flatMap((name) => readFileSync(join(HANDLER_DIR, name), 'utf-8').match(/"redis\.[A-Za-z0-9_]+"/g) ?? [])
            .map((raw) => raw.replace(/"redis\./, '').replace(/"$/, ''))
    );

    it('后端确实声明了视角文案（防止扫描路径失效导致本用例空跑）', () => {
        expect(goKeys.size).toBeGreaterThan(50);
    });

    it('中文语言包齐全', () => {
        expect([...goKeys].filter((key) => !keys(zh).has(key))).toEqual([]);
    });

    it('英文语言包齐全', () => {
        expect([...goKeys].filter((key) => !keys(en).has(key))).toEqual([]);
    });
});

describe('成员/操作弹层的回填契约', () => {
    it('新增与操作弹层不能传空对象作为回填数据（否则 schema 默认值失效并残留上次输入）', () => {
        const source = readFileSync(join(import.meta.dirname, '../keyview/useKeyFormDialog.ts'), 'utf-8');
        expect(source).not.toMatch(/dialog\.data = \{\}/);
        // 编辑态仍按行数据回填，新增态与操作态传 null 交给宿主重建默认表单
        expect(source).toMatch(/prefillForm\(schema, row\) : null/);
    });
});

describe('前端引用的 redis 文案必须两侧语言包都有', () => {
    /** 递归收集模块源码（跳过用例自身，避免把断言里的 key 当引用） */
    const collect = (dir: string): string[] =>
        readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
            const full = join(dir, entry.name);
            if (entry.isDirectory()) {
                return entry.name === '__tests__' ? [] : collect(full);
            }
            return /\.(ts|vue)$/.test(entry.name) ? [readFileSync(full, 'utf-8')] : [];
        });

    const used = new Set(
        collect(join(import.meta.dirname, '..')).flatMap((src) => [...src.matchAll(/\$?t\(\s*'redis\.([A-Za-z0-9_]+)'/g)].map((match) => match[1]))
    );

    it('用例确实扫到了引用（防止路径失效导致空跑）', () => {
        expect(used.size).toBeGreaterThan(30);
    });

    it('中文语言包齐全', () => {
        expect([...used].filter((key) => !keys(zh).has(key))).toEqual([]);
    });

    it('英文语言包齐全', () => {
        expect([...used].filter((key) => !keys(en).has(key))).toEqual([]);
    });
});

/** 命令目录条目的测试构造器：只关心提示与确认用到的字段 */
function cmdSpec(name: string, extra: Partial<RedisCommandSpec> = {}): RedisCommandSpec {
    return { name, arity: -1, flags: [], firstKey: 0, lastKey: 0, step: 0, needConfirm: false, ...extra };
}

describe('命令控制台的输入提示', () => {
    it('当前输入位：命令名、参数位、以空白结尾算开新参数', () => {
        expect(editingToken('H')).toEqual({ index: 0, token: 'H' });
        expect(editingToken('HGETALL ')).toEqual({ index: 1, token: '' });
        expect(editingToken('HGETALL us')).toEqual({ index: 1, token: 'us' });
        // 引号内的空格不构成分隔，否则带空格的值会把参数位算错
        expect(editingToken('ZADD key 1 "a b"')).toEqual({ index: 3, token: 'a b' });
        // 引号未闭合时按空白继续切分（与 redis-cli 一致，未闭合的引号不算一个参数），
        // 参数位会因此偏后而落在非键位上，结果是这一拍不提示，不影响命令执行
        expect(editingToken('ZADD key 1 "a b')).toEqual({ index: 4, token: 'b' });
    });

    it('键参数位置由命令自描述的 first/last/step 决定', () => {
        const hgetall = cmdSpec('HGETALL', { arity: 2, firstKey: 1, lastKey: 1, step: 1 });
        expect(isKeyArgument(hgetall, 0)).toBe(false);
        expect(isKeyArgument(hgetall, 1)).toBe(true);
        expect(isKeyArgument(hgetall, 2)).toBe(false);

        // MSET 的键参数是隔一个出现：key value key value
        const mset = cmdSpec('MSET', { firstKey: 1, lastKey: -1, step: 2 });
        expect(isKeyArgument(mset, 1)).toBe(true);
        expect(isKeyArgument(mset, 2)).toBe(false);
        expect(isKeyArgument(mset, 3)).toBe(true);

        // PING 这类无键参数的命令不给 key 提示；目录里查不到的命令宽松提示
        expect(isKeyArgument(cmdSpec('PING'), 1)).toBe(false);
        expect(isKeyArgument(undefined, 1)).toBe(true);
    });

    it('命令与 key 建议：前缀命中优先、空输入不给提示', () => {
        const specs = [cmdSpec('GETRANGE'), cmdSpec('GET'), cmdSpec('OBJECT'), cmdSpec('HGET')];
        expect(suggestCommands(specs, 'ge').map((item) => item.name)).toEqual(['GET', 'GETRANGE', 'HGET']);
        expect(suggestCommands(specs, '')).toEqual([]);
        // 命令建议带尾空格，选中后可直接继续输参数
        expect(suggestCommands(specs, 'get')[0].value).toBe('get ');

        const keys = ['user:1001', 'user:1002', 'session:1001'];
        expect(suggestKeys(keys, 'user').map((item) => item.name)).toEqual(['user:1001', 'user:1002']);
        expect(suggestKeys(keys, '1001').map((item) => item.name)).toEqual(['user:1001', 'session:1001']);
        expect(suggestKeys(keys, '')).toEqual([]);
    });

    it('执行前确认以服务端标记为准，目录未就绪时落到兜底名单', () => {
        expect(commandNeedsConfirm(cmdSpec('FLUSHALL', { needConfirm: true }), 'flushall')).toBe(true);
        expect(commandNeedsConfirm(cmdSpec('SET', { flags: ['write'] }), 'set')).toBe(false);
        expect(commandNeedsConfirm(undefined, 'FLUSHDB')).toBe(true);
        expect(commandNeedsConfirm(undefined, 'GET')).toBe(false);
    });
});

describe('命令控制台的命令行切分', () => {
    it('按空白切分并忽略多余空格', () => {
        expect(splitCommand('  SET   a  b  ')).toEqual(['SET', 'a', 'b']);
    });

    it('引号内的空格属于同一个参数', () => {
        expect(splitCommand('SET k "a b"')).toEqual(['SET', 'k', 'a b']);
        expect(splitCommand("SET k 'a b'")).toEqual(['SET', 'k', 'a b']);
    });

    it('支持转义字符', () => {
        expect(splitCommand('SET k "a\\nb"')).toEqual(['SET', 'k', 'a\nb']);
    });

    it('空行不产出参数', () => {
        expect(splitCommand('   ')).toEqual([]);
    });

    it('多次调用互不影响（切分正则是模块级共享对象）', () => {
        expect(splitCommand('GET a')).toEqual(['GET', 'a']);
        expect(splitCommand('GET b')).toEqual(['GET', 'b']);
    });
});

describe('成员分页的「还有更多」判据', () => {
    const caps = (over: Partial<Record<'rankPaging' | 'cursorPaging', boolean>>) => ({
        create: false,
        update: false,
        delete: false,
        batchDelete: false,
        keyword: false,
        rankPaging: false,
        cursorPaging: false,
        ops: false,
        ...over,
    });

    it('按下标分页的视角看已读条数（zset 首屏无游标也要能继续翻页）', () => {
        expect(hasMoreMembers(caps({ rankPaging: true, cursorPaging: true }), '', 50, 1000)).toBe(true);
        expect(hasMoreMembers(caps({ rankPaging: true }), '', 1000, 1000)).toBe(false);
    });

    it('纯 scan 类视角看游标是否归零', () => {
        expect(hasMoreMembers(caps({ cursorPaging: true }), '17', 50, 1000)).toBe(true);
        expect(hasMoreMembers(caps({ cursorPaging: true }), '', 50, 1000)).toBe(false);
    });
});

describe('通用成员视图解释器', () => {
    const geoRow: RedisKeyMember = { index: 0, field: 'palermo', value: '', score: 3386, id: '', extra: { longitude: '13.36', latitude: '38.11' } };

    it('自有字段优先，取不到时回退 extra（派生列）', () => {
        const score: RedisViewColumn = { field: 'score', label: 'redis.colScore', width: 100, value: 'number', sortable: true };
        const lon: RedisViewColumn = { field: 'longitude', label: 'redis.colLongitude', width: 100, value: 'number', sortable: false };
        expect(columnValue(geoRow, score)).toBe('3386');
        expect(columnValue(geoRow, lon)).toBe('13.36');
    });

    it('未知字段名不会误读出内容', () => {
        const bad: RedisViewColumn = { field: 'constructor', label: 'x', width: 0, value: 'text', sortable: false };
        expect(columnValue(geoRow, bad)).toBe('');
    });

    it('编辑回填按 schema 的 prop 取值，数字控件拿到真正的数字', () => {
        const schema: RedisFormSchema = {
            version: 1,
            fields: [
                { prop: 'field', label: 'redis.colMember', type: 'input' },
                { prop: 'longitude', label: 'redis.colLongitude', type: 'number' },
                { prop: 'missing', label: 'redis.colRemark' },
            ],
        };
        expect(prefillForm(schema, geoRow)).toEqual({ field: 'palermo', longitude: 13.36 });
    });

    it('没有行数据时不回填（新增态）', () => {
        expect(prefillForm({ version: 1, fields: [{ prop: 'value', label: 'redis.colValue' }] })).toEqual({});
    });
});

describe('批量选择的「已全选」判据', () => {
    it('可见项逐项命中才算已全选', () => {
        expect(coversAll(['a', 'b'], ['a', 'b', 'c'])).toBe(true);
        expect(coversAll(['a', 'b'], ['a'])).toBe(false);
        expect(coversAll(['a'], ['x', 'a'])).toBe(true);
    });

    it('长度比较会误判：筛到更小的集合时不能算成已全选', () => {
        // 先勾了 20 个 string，再把筛选切到只剩 2 个 hash：20 >= 2 会假报「已全选」
        expect(
            coversAll(
                ['h1', 'h2'],
                Array.from({ length: 20 }, (_, i) => `s${i}`)
            )
        ).toBe(false);
    });

    it('没有可见项时不认为已全选（按钮应留在「全选」而不是「取消全选」）', () => {
        expect(coversAll([], ['a'])).toBe(false);
        expect(coversAll([], [])).toBe(false);
    });
});

describe('字段过期列的读数', () => {
    it('剩余秒数按天/时/分三档刻度显示', () => {
        expect(formatTtlSeconds(45)).toBe('00:45');
        expect(formatTtlSeconds(300)).toBe('05:00');
        expect(formatTtlSeconds(3725)).toBe('01:02:05');
        expect(formatTtlSeconds(90061)).toBe('1d 01:01:01');
    });

    it('-1 是没设过期，-2 是字段已不存在，取不到值是当前实例不支持', () => {
        expect(ttlCellText('-1', '永久')).toBe('永久');
        expect(ttlCellText('-2', '永久')).toBe('');
        expect(ttlCellText('', '永久')).toBe('');
        expect(ttlCellText('60', '永久')).toBe('01:00');
    });
});

describe('图标名必须能解析到已注册图标', () => {
    /**
     * 图标靠全局注册按名解析，名字写错不会报错只会渲染成空白按钮（尤其是图标位由后端描述符下发时）。
     * 这里把前端字面量与后端描述符声明的图标一起对一遍
     */
    const registered = new Set(Object.values(eleIcons).map((item) => (item as { name: string }).name));

    const resolves = (name: string) => {
        const camel = name.replace(/-(\w)/g, (_, char: string) => char.toUpperCase());
        return [name, camel, `${camel.charAt(0).toUpperCase()}${camel.slice(1)}`].some((candidate) => registered.has(candidate));
    };

    const collectVue = (dir: string): string[] =>
        readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
            const full = join(dir, entry.name);
            if (entry.isDirectory()) {
                return entry.name === '__tests__' || entry.name === 'node_modules' ? [] : collectVue(full);
            }
            return entry.name.endsWith('.vue') ? [readFileSync(full, 'utf-8')] : [];
        });

    const frontendIcons = collectVue(join(import.meta.dirname, '..')).flatMap((src) => [
        ...[...src.matchAll(/<SvgIcon[^>]*?\sname="([A-Za-z][A-Za-z0-9 -]*)"/g)].map((match) => match[1]),
        ...[...src.matchAll(/\sicon="([A-Za-z][A-Za-z0-9 -]*)"/g)].map((match) => match[1]),
    ]);

    const skinIcons = [...Object.values(viewSkins).map((skin) => skin.icon), ...Object.values(opSkins)];

    it('用例确实扫到了图标声明（防止正则失效导致空跑）', () => {
        expect(frontendIcons.length).toBeGreaterThan(15);
        expect(skinIcons.length).toBeGreaterThan(25);
    });

    it('组件与皮肤表声明的图标都存在', () => {
        expect([...new Set([...frontendIcons, ...skinIcons])].filter((name) => !resolves(name))).toEqual([]);
    });
});

describe('视角皮肤必须覆盖后端声明的每个视角与操作', () => {
    /**
     * 皮肤搬到前端后，「后端有 9 个视角、前端只画了 8 个」不再有编译期或接口层约束，
     * 表现是某个类型的徽章静默变成 '--'、操作图标变成问号。这里扫后端 handler 源码核对，缺一个就红。
     */
    const sources = readdirSync(HANDLER_DIR)
        .filter((name) => name.endsWith('.go') && !name.endsWith('_test.go'))
        .map((name) => readFileSync(join(HANDLER_DIR, name), 'utf-8'));

    // 每个处理器文件恰好一个视角描述符，view 由 View* 常量声明
    const viewOf = (src: string) => src.match(/^const View\w+ = "([^"]+)"/m)?.[1];
    const goViews = sources
        .map(viewOf)
        .filter((view): view is string => !!view)
        .sort();

    // op 名可能是字符串字面量，也可能是同文件里的常量（如 opHExpire），两种都要认出来
    const goOps = sources.flatMap((src) => {
        const view = viewOf(src);
        if (!view) {
            return [];
        }
        const consts = [...src.matchAll(/^const (\w+) = "([^"]+)"/gm)];
        const nameOf = (raw: string) => {
            if (raw.startsWith('"')) {
                return raw.slice(1, -1);
            }
            return consts.find((match) => match[1] === raw)?.[2] ?? raw;
        };
        return [...src.matchAll(/op\(("([^"]+)"|[A-Za-z]\w+), "redis\.[A-Za-z]+", (?:true|false),/g)].map((match) => `${view}:${nameOf(match[1])}`).sort();
    });

    it('用例确实扫到了后端视角与操作（防止正则失效导致空跑）', () => {
        expect(goViews.length).toBeGreaterThanOrEqual(9);
        expect(goOps.length).toBeGreaterThanOrEqual(21);
    });

    it('每个视角都登记了皮肤', () => {
        expect(goViews.filter((view) => !(view in viewSkins))).toEqual([]);
    });

    it('每个视角操作都登记了图标', () => {
        expect(goOps.filter((op) => !(op in opSkins))).toEqual([]);
    });

    it('徽章定长且全局唯一（撞车就失去区分意义）', () => {
        const badges = Object.entries(viewSkins).map(([view, skin]) => [view, skin.badge] as const);
        expect(badges.filter(([, badge]) => badge.length !== 2)).toEqual([]);
        expect(new Set(badges.map(([, badge]) => badge)).size).toBe(badges.length);
    });
});
