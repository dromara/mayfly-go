import { readFileSync, readdirSync, statSync } from 'node:fs';
import path from 'node:path';
import { describe, expect, expectTypeOf, it } from 'vitest';
import type { TreeNode } from '@/views/ops/resource/tree';
import type { DbNodeParams, DbTableNodeParams } from '../../types';
import { DbKind, DbTableKind, dbNodeParams, dbTableNodeParams, toTableCallbackData, type DbOpTabApi } from '../helpers';

/**
 * DB 资源树跨层契约守卫
 *
 * 资源树（commands / contributors / widgets）与标签页容器 DbDataOp 之间隔着组件实例边界：
 * 前者经 getDbOpTabCompInst 拿到实例后按 `DbOpTabApi` 调用后者。这类跨组件契约一旦把签名写宽
 * （参数用 any），`satisfies DbOpTabApi` 就退化成「只查键名不查签名」——实现端改参数顺序或类型
 * 都不报错，要等用户点了右键菜单才在运行时炸。退化本身不改变任何现有功能表现，故用断言锁住。
 *
 * 守卫分三层，缺一层就有绕过路径：
 * 1. 源码守卫：接口体里不得出现 any（vitest 直接跑就能拦，不依赖类型检查开关）；
 * 2. 编译期守卫：any 藏在类型别名后面时源码扫描看不出，由 AnyMembers 推导为 never 兜住；
 * 3. 地基守卫：DbDataOp 端的 `satisfies` 一旦被摘掉，上面两层全部形同虚设。
 */

const DB_ROOT = path.resolve(__dirname, '../..');
const HELPERS_FILE = path.resolve(__dirname, '../helpers.ts');
const DB_DATA_OP_FILE = path.resolve(__dirname, '../DbDataOp.vue');

/** 去掉块注释与整行注释，避免注释里举例的 any / as Record<string, any> 被当成真代码 */
function stripComments(src: string): string {
    return src.replace(/\/\*[\s\S]*?\*\//g, '').replace(/^\s*\/\/.*$/gm, '');
}

/** 递归收集 db 模块源码文件（跳过测试目录） */
function collect(dir: string): string[] {
    return readdirSync(dir).flatMap((name) => {
        const full = path.join(dir, name);
        if (statSync(full).isDirectory()) {
            return name === '__tests__' ? [] : collect(full);
        }
        return /\.(ts|vue)$/.test(name) ? [full] : [];
    });
}

/** 构造树节点 fixture；不传 params 即模拟水合前的退化态 */
const node = (params?: Record<string, unknown>): TreeNode => ({
    key: 'inst1.db1',
    kind: DbKind,
    label: 'db1',
    hasChildren: true,
    params: params ?? {},
});

// ---------------------------------- 类型级工具 ----------------------------------

/** 判定 T 是否为 any：只有 any 会让 `1 & T` 塌成 any，从而 0 可赋值给它 */
type IsAny<T> = 0 extends 1 & T ? true : false;

/** 解包 Promise，使 `Promise<any>` 这类返回值同样被判定为失真 */
type Unwrap<T> = T extends Promise<infer U> ? U : T;

/** 函数签名中任一入参或返回值（解包后）为 any，即视为签名失真 */
type HasAnyInSignature<F> = F extends (...args: infer A) => infer R
    ? true extends { [K in keyof A]: IsAny<A[K]> }[number]
        ? true
        : IsAny<Unwrap<R>>
    : IsAny<F>;

/** 收集接口中签名失真的成员名；契约完好时应为 never */
type AnyMembers<T> = { [K in keyof T]-?: HasAnyInSignature<T[K]> extends true ? K : never }[keyof T];

describe('DbOpTabApi 跨组件契约不得退化为 any', () => {
    /** 取出 `export interface DbOpTabApi { ... }` 的接口体（成员均缩进，故首个行首右括号即结束） */
    const apiBody = (): string => {
        const src = readFileSync(HELPERS_FILE, 'utf8');
        const start = src.indexOf('export interface DbOpTabApi {');
        expect(start, 'helpers.ts 中已找不到 DbOpTabApi 声明，本用例需随之调整').toBeGreaterThan(-1);
        const rest = src.slice(start);
        return rest.slice(0, rest.indexOf('\n}'));
    };

    it('接口体内不出现 any（源码守卫）', () => {
        const code = stripComments(apiBody());
        // 逐个成员列出命中项，便于直接定位是哪一条签名被写宽
        const loosened = code
            .split('\n')
            .filter((line) => /\bany\b/.test(line))
            .map((line) => line.trim());
        expect(loosened, 'DbOpTabApi 的以下成员签名含 any，satisfies 校验会失效').toEqual([]);
    });

    it('没有任何成员的签名含 any（编译期守卫，覆盖藏在别名后的 any）', () => {
        expectTypeOf<AnyMembers<DbOpTabApi>>().toEqualTypeOf<never>();
    });

    it('DbDataOp 的 defineExpose 仍受 satisfies DbOpTabApi 约束（地基守卫）', () => {
        const src = readFileSync(DB_DATA_OP_FILE, 'utf8');
        expect(src, '摘掉 satisfies 后接口与实现即可各自漂移，上面两层守卫全部失效').toMatch(
            /defineExpose\(\{[\s\S]*?\}\s*satisfies\s+DbOpTabApi\s*\)/
        );
    });
});

describe('节点 params 单点收窄', () => {
    it('原样返回节点 params（不复制，调用方拿到的就是水合后的同一对象）', () => {
        const n = node({ id: 7, db: 'db1' });
        expect(dbNodeParams(n)).toBe(n.params);
    });

    it('params 缺失时兜底为空对象，而非抛 undefined 解构', () => {
        // TreeNodeData 的 params 是可选的，访问器放宽到最小结构后这里无需 cast 即可覆盖
        expect(dbNodeParams({})).toEqual({});
        expect(dbTableNodeParams({ params: undefined })).toEqual({});
    });

    it('泛型参数可为叶子节点声明更强不变式（tableName 由可选收窄为必有）', () => {
        const params = dbTableNodeParams<DbTableNodeParams & { tableName: string }>({
            params: { id: 7, db: 'db1', tableName: 't_user' },
        });
        expectTypeOf(params.tableName).toEqualTypeOf<string>();
        expect(params.tableName).toBe('t_user');
    });

    it('表粒度 params 继承库粒度 params（两处定义不得各自漂移）', () => {
        expectTypeOf<DbTableNodeParams>().toMatchTypeOf<DbNodeParams>();
        const params = dbNodeParams(node({ id: 7, db: 'db1' }));
        expect(params.id).toBe(7);
    });

    it('toTableCallbackData 把树节点映射为表级操作载荷', () => {
        const n: TreeNode = { ...node({ id: 7, db: 'db1', tableName: 't_user' }), kind: DbTableKind };
        expect(toTableCallbackData(n)).toEqual({ params: n.params, key: n.key, label: n.label });
    });

    it('db 模块不用 `as Record<string, any>` 绕行（其他资源模块的既有写法）', () => {
        const violators = collect(DB_ROOT)
            .map((file) => ({ file: path.relative(DB_ROOT, file), code: stripComments(readFileSync(file, 'utf8')) }))
            .filter(({ code }) => /as\s+Record<string,\s*any>/.test(code))
            .map(({ file }) => file);
        expect(violators, '节点 params 一律经 helpers 的收窄访问器读取').toEqual([]);
    });
});
