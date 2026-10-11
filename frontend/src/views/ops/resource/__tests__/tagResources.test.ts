import { describe, expect, it, vi } from 'vitest';

import { registerContributor } from '../tree/registry';
import { registerResource } from '../resourceRegistry';
import { ERROR_KIND } from '../tree/types';
import type { TreeNodeData } from '../tree/types';
import { createTypeFilter, deriveTypeOptions, loadTagResources } from '../tagResources';

/**
 * 标签层取数策略专测：类型分组节点层移除后，原本由「一个分组一个懒加载节点」天然保证的
 * 三条性质全部收进 loadTagResources，必须逐条钉住：
 * ① 按类型顺序合并（同类资源仍相邻）；② 单个类型失败不拖垮整个标签；③ 未注册类型静默跳过。
 */

const TYPE_A = 'ut-tag-type-a';
const TYPE_B = 'ut-tag-type-b';
const TYPE_EMPTY = 'ut-tag-type-empty';
const TYPE_BOOM = 'ut-tag-type-boom';

const rows = (prefix: string, count: number): TreeNodeData[] =>
    Array.from({ length: count }, (_, i) => ({ key: `${prefix}.${i}`, kind: 'leaf', label: `${prefix}-${i}` }));

registerContributor({ kind: 'ut-root-a', resourceType: TYPE_A, loadRoots: async (tagPath) => rows(tagPath, 2) });
registerContributor({ kind: 'ut-root-b', resourceType: TYPE_B, loadRoots: async (tagPath) => rows(`${tagPath}-b`, 1) });
// 声明了根 kind 但不实现 loadRoots（如某资产只在子层出现）
registerContributor({ kind: 'ut-root-empty', resourceType: TYPE_EMPTY });
registerContributor({
    kind: 'ut-root-boom',
    resourceType: TYPE_BOOM,
    loadRoots: async () => {
        throw new Error('boom');
    },
});

it('按类型顺序合并，且每个贡献者收到的入参就是标签 codePath', async () => {
    const result = await loadTagResources('t1/', [TYPE_A, TYPE_B]);
    expect(result.map((n) => n.key)).toEqual(['t1/.0', 't1/.1', 't1/-b.0']);
});

it('未注册 / 未实现 loadRoots 的类型当空处理，不产生错误行', async () => {
    expect(await loadTagResources('t1/', ['ut-unknown-type'])).toEqual([]);
    expect(await loadTagResources('t1/', [TYPE_EMPTY])).toEqual([]);
});

it('单个类型取数失败：其余类型照常返回，末尾补一条指向标签的重试行', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    const result = await loadTagResources('t1/', [TYPE_A, TYPE_BOOM, TYPE_B]);

    expect(result.map((n) => n.key)).toEqual(['t1/.0', 't1/.1', 't1/-b.0', 't1/__load_error']);
    const errorRow = result[result.length - 1];
    expect(errorRow.kind).toBe(ERROR_KIND);
    // retryTarget 必须是标签节点本身：重试点亮整个标签的取数（与 useLazyTree 的错误占位同一协议）
    expect(errorRow.params).toEqual({ retryTarget: 't1/' });
    expect(err).toHaveBeenCalled();

    err.mockRestore();
});

it('全部类型失败时只剩重试行（不返回空集，避免看着像「这个标签没有资源」）', async () => {
    const err = vi.spyOn(console, 'error').mockImplementation(() => undefined);

    const result = await loadTagResources('t1/', [TYPE_BOOM]);

    expect(result).toHaveLength(1);
    expect(result[0].kind).toBe(ERROR_KIND);

    err.mockRestore();
});

describe('deriveTypeOptions', () => {
    const MACHINE_TYPE = 9101;
    const ORPHAN_TYPE = 9102;

    registerResource(MACHINE_TYPE, { resourceType: MACHINE_TYPE, manager: { componentConf: { name: 'tag.machine', icon: { name: 'Server' } } } });
    // 注册了资源模块但没有根贡献者：这类东西在树上不占一层，不该出现在筛选选项里
    registerResource(ORPHAN_TYPE, { resourceType: ORPHAN_TYPE, manager: { componentConf: { name: 'tag.db' } } });
    registerContributor({ kind: 'ut-kind-machine', resourceType: MACHINE_TYPE });

    it('只列「有 manager 配置 + 有根贡献者」的类型，值即根节点 kind', () => {
        const options = deriveTypeOptions();
        const machine = options.find((option) => option.kind === 'ut-kind-machine');
        expect(machine?.label).toBe('tag.machine');
        expect(machine?.icon).toEqual({ name: 'Server' });
        expect(options.some((option) => option.kind === 'ut-orphan-kind')).toBe(false);
    });
});

describe('createTypeFilter', () => {
    const ROOT = 'ut-filter-root';
    const MACHINE_TYPE = 9201;
    const REDIS_TYPE = 9202;
    registerResource(MACHINE_TYPE, { resourceType: MACHINE_TYPE, manager: { componentConf: { name: 'tag.machine' } } });
    registerResource(REDIS_TYPE, { resourceType: REDIS_TYPE, manager: { componentConf: { name: 'tag.redis' } } });
    registerContributor({ kind: ROOT, resourceType: MACHINE_TYPE });

    it('标签层按骨架声明的类型清单判定：未展开的标签不得被整批隐藏（否则选完类型树就空了）', () => {
        const filter = createTypeFilter(ROOT, () => undefined);
        expect(filter({ kind: 'tag', params: { resourceTypes: [MACHINE_TYPE] } })).toBe(true);
        expect(filter({ kind: 'tag', params: { resourceTypes: [REDIS_TYPE] } })).toBe(false);
    });

    it('资源层自身或任一祖先命中根 kind 即通过（其下属行随父可见）', () => {
        const machine = { kind: ROOT, parentKey: 't1/' };
        // 凭证的父是机器：链 凭证 → 机器 → 标签
        const lookup = (key: string) => (key === 'm1' ? machine : key === 't1/' ? { kind: 'tag' } : undefined);
        const filter = createTypeFilter(ROOT, lookup);

        expect(filter(machine)).toBe(true);
        expect(filter({ kind: 'ut-cert', parentKey: 'm1' })).toBe(true);
        expect(filter({ kind: 'ut-redis', parentKey: 't1/' })).toBe(false);
    });

    it('祖先链断裂或成环时判不命中，而不是死循环', () => {
        const loop = { kind: 'ut-loop', parentKey: 'loop' };
        const filter = createTypeFilter(ROOT, () => loop);
        expect(filter({ kind: 'ut-stray', parentKey: 'loop' })).toBe(false);
        // 上溯不到根且父链查不到：同样不命中
        expect(filter({ kind: 'ut-stray', parentKey: 'missing' })).toBe(false);
    });
});
