import { describe, it, expect } from 'vitest';
import { registerContributor } from '../tree/registry';
import type { TreeNode, LocateAccess } from '../tree/types';
import { parseCodePath, resolveCodePathLocate } from '../locateResource';
import { resGroupKey } from '../resourceKeys';

/**
 * codePath 定位解析单测（纯函数 + 依赖注入，无需挂载树/组件）：
 * 覆盖标签路径解析、单级资源、嵌套 tag + 凭证下钻、跨级跳过（数据库账号段）、以及 pending/missing 三态判定。
 * 用独立 kind 注册带 locateCode 的假贡献者，验证「解析只依赖 locateCode 契约、不依赖 key 规则」这一开闭原则要点。
 */

const KIND_MACHINE = 'ut-machine';
const KIND_MACHINE_CERT = 'ut-machine-cert';
const KIND_DB_INST = 'ut-db-inst';
const KIND_DB = 'ut-db';

// 各资源以 locateCode 声明自身对应的 codePath 资源段 code（key 规则故意各异，定位不应关心）
registerContributor({ kind: KIND_MACHINE, locateCode: (n) => n.params.code as string });
registerContributor({ kind: KIND_MACHINE_CERT, locateCode: (n) => (n.params.cert as { name: string }).name });
registerContributor({ kind: KIND_DB_INST, locateCode: (n) => n.params.code as string });
registerContributor({ kind: KIND_DB, locateCode: (n) => n.params.dbCode as string });

const node = (key: string, kind: string, params: Record<string, unknown> = {}, children?: TreeNode[]): TreeNode => ({
    key,
    kind,
    label: key,
    params,
    hasChildren: !!children?.length,
    loaded: !!children,
    children,
});

/** 假树访问器：递归铺平所有子孙节点（对齐真实水合层的扁平 nodeIndex），expandNode 仅翻转 loaded 标记 */
function makeAccess(nodes: TreeNode[], rootLoaded: boolean): LocateAccess {
    const map = new Map<string, TreeNode>();
    const walk = (list: TreeNode[]) => {
        for (const n of list) {
            map.set(n.key, n);
            if (n.children) walk(n.children);
        }
    };
    walk(nodes);
    return {
        getNode: (k) => map.get(k),
        expandNode: async (k) => {
            const n = map.get(k);
            if (n) n.loaded = true;
        },
        rootLoaded,
    };
}

describe('parseCodePath', () => {
    it('拆分标签路径与有序资源段', () => {
        const { tagPath, resources } = parseCodePath('a/b/1|m1/5|root/');
        expect(tagPath).toBe('a/b/');
        expect(resources).toEqual([
            { type: 1, code: 'm1' },
            { type: 5, code: 'root' },
        ]);
    });

    it('纯标签路径无资源段', () => {
        const { tagPath, resources } = parseCodePath('a/b/');
        expect(tagPath).toBe('a/b/');
        expect(resources).toEqual([]);
    });
});

describe('resolveCodePathLocate', () => {
    it('单级资源：解析到分组下的实例节点', async () => {
        const group = node(resGroupKey('t1/', 1), 'group', {}, [node('mc1', KIND_MACHINE, { code: 'mc1' })]);
        const access = makeAccess([group], true);
        expect(await resolveCodePathLocate('t1/1|mc1/', access)).toEqual({ status: 'resolved', key: 'mc1' });
    });

    it('嵌套 tag + 凭证：逐层下钻到机器授权凭证节点', async () => {
        const cert = node('m1.root', KIND_MACHINE_CERT, { cert: { name: 'root' } });
        const machine = node('m1', KIND_MACHINE, { code: 'm1' }, [cert]);
        const group = node(resGroupKey('a/b/', 1), 'group', {}, [machine]);
        const access = makeAccess([group], true);
        expect(await resolveCodePathLocate('a/b/1|m1/5|root/', access)).toEqual({ status: 'resolved', key: 'm1.root' });
    });

    it('跨级跳过：数据库 codePath 含账号段(5|)但树为 实例-库名，仍解析到库名节点', async () => {
        const db = node('inst1.db1', KIND_DB, { dbCode: 'db1' });
        const inst = node('inst1', KIND_DB_INST, { code: 'inst1' }, [db]);
        const group = node(resGroupKey('d/', 2), 'group', {}, [inst]);
        const access = makeAccess([group], true);
        expect(await resolveCodePathLocate('d/2|inst1/5|acc1/22|db1/', access)).toEqual({ status: 'resolved', key: 'inst1.db1' });
    });

    it('纯标签路径：标签节点 key 即其 codePath', async () => {
        const access = makeAccess([node('a/b/', 'tag')], true);
        expect(await resolveCodePathLocate('a/b/', access)).toEqual({ status: 'resolved', key: 'a/b/' });
    });

    it('分组缺失且根已加载 -> missing（无权限/已删除，放弃挂起）', async () => {
        const access = makeAccess([], true);
        expect(await resolveCodePathLocate('x/1|mc9/', access)).toEqual({ status: 'missing' });
    });

    it('分组缺失但根未加载 -> pending（待水合后重试）', async () => {
        const access = makeAccess([], false);
        expect(await resolveCodePathLocate('x/1|mc9/', access)).toEqual({ status: 'pending' });
    });

    it('分组命中但首段资源 code 无人匹配 -> missing（资源已删除）', async () => {
        const group = node(resGroupKey('t1/', 1), 'group', {}, [node('other', KIND_MACHINE, { code: 'other' })]);
        const access = makeAccess([group], true);
        expect(await resolveCodePathLocate('t1/1|mc1/', access)).toEqual({ status: 'missing' });
    });
});
