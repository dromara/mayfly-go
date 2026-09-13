import { beforeEach, describe, expect, expectTypeOf, it, vi } from 'vitest';
import { dbApi } from '@/views/ops/db/api';
import type { DbInstInfo } from '../../types';
import { cacheDbInst, clearAllDbInstCache, getCachedDbInst, getOrNewDbInst, type CachedDbInst } from '../dbCache';

/**
 * 实例缓存契约（CachedDbInst）行为守卫
 *
 * 缓存层原先以 `Map<number, unknown>` 存放实例，读写字段全靠 `(dbInst as { tagPath: string })`
 * 这类逐字段断言——字段名写错、类型写错编译器都不报错。改成泛型契约后，断言只剩缓存边界上
 * 的一处视图转换，故这里锁住「契约化没有改变缓存语义」：命中/回填/版本异步补齐/清空重建。
 */

/** 缓存层只用到兼容版本接口，整体 mock 掉 api 桶，避免把 request/加密/系统消息等副作用链拖进单测 */
vi.mock('@/views/ops/db/api', () => ({
    dbApi: {
        getCompatibleDbVersion: { request: vi.fn() },
    },
}));

/**
 * 模拟 DbInst：契约之外的字段（dbs）缓存层看不到，但经泛型取回后调用方仍能直接访问——
 * 这正是「调用点无需断言」的收益，也是本文件要用具体类而非契约类型做夹具的原因。
 */
class FakeInst implements CachedDbInst {
    id = 0;
    tagPath = '';
    databases: string[] = [];
    version = '';
    /** 契约外字段：DbInst.dbs 的对应物 */
    dbs = new Map<string, string>();
}

/**
 * 建实例夹具。显式标注返回 FakeInst：`Object.assign` 会推出交叉类型，
 * 令 getOrNewDbInst 的 T 随之变成交叉类型，类型断言就测不到「取回即原类型」这一点了。
 */
const newInst = (id: number, extra?: Partial<FakeInst>): FakeInst => Object.assign(new FakeInst(), { id }, extra);

/** 与 DbInst.getOrNewInst 的工厂同构：tagPath 由实例信息落地，否则「命中时保留原值」无从验证 */
const instFactory = (info: DbInstInfo): FakeInst => newInst(info.id, { tagPath: info.tagPath ?? '' });

const versionRequest = vi.mocked(dbApi.getCompatibleDbVersion.request);

/** 建一个实例信息（DbInstInfo 只强制 id） */
const instInfo = (id: number, tagPath?: string): DbInstInfo => ({ id, tagPath });

beforeEach(() => {
    clearAllDbInstCache();
    vi.clearAllMocks();
    versionRequest.mockResolvedValue('8.0.32');
});

describe('getOrNewDbInst', () => {
    it('首次调用走工厂建实例，二次调用命中缓存不再建（返回同一对象）', () => {
        const factory = vi.fn(instFactory);
        const first = getOrNewDbInst(instInfo(1), factory);
        const second = getOrNewDbInst(instInfo(1), factory);
        expect(factory).toHaveBeenCalledTimes(1);
        expect(second).toBe(first);
        // 建实例后已入缓存，getCachedDbInst 能直接取到
        expect(getCachedDbInst(1)).toBe(first);
    });

    it('缓存命中时按最新 tagPath 回填（同一库可关联多个标签，展示需取当次上下文）', () => {
        getOrNewDbInst(instInfo(2, 'tagA|'), instFactory);
        expect(getOrNewDbInst(instInfo(2, 'tagB|'), instFactory).tagPath).toBe('tagB|');
    });

    it('缓存命中但未带 tagPath 时保留原值（不被 undefined 覆盖）', () => {
        getOrNewDbInst(instInfo(3, 'tagA|'), instFactory);
        expect(getOrNewDbInst(instInfo(3), instFactory).tagPath).toBe('tagA|');
    });

    it('新实例带 databases 时以首库查兼容版本并异步回填 version', async () => {
        const inst = getOrNewDbInst(instInfo(4), (info) => newInst(info.id, { databases: ['db1', 'db2'] }));
        expect(versionRequest).toHaveBeenCalledWith({ id: 4, db: 'db1' });
        // 回填发生在 then 里，需等微任务落地
        await vi.waitFor(() => expect(inst.version).toBe('8.0.32'));
    });

    it('新实例无 databases 时不发起版本查询（避免以空库名请求）', async () => {
        const inst = getOrNewDbInst(instInfo(5), instFactory);
        expect(versionRequest).not.toHaveBeenCalled();
        await Promise.resolve();
        expect(inst.version).toBe('');
    });

    it('清空缓存后同一 id 会重新走工厂（实例内的表/列缓存随之丢弃）', () => {
        const factory = vi.fn(instFactory);
        const first = getOrNewDbInst(instInfo(6), factory);
        clearAllDbInstCache();
        expect(getCachedDbInst(6)).toBeUndefined();
        expect(getOrNewDbInst(instInfo(6), factory)).not.toBe(first);
        expect(factory).toHaveBeenCalledTimes(2);
    });
});

describe('缓存读写与泛型契约', () => {
    it('cacheDbInst 登记的实例可被 getCachedDbInst 取回', () => {
        const inst = newInst(7);
        cacheDbInst(7, inst);
        expect(getCachedDbInst(7)).toBe(inst);
    });

    it('未缓存的 id 返回 undefined，而非抛错或返回空对象', () => {
        expect(getCachedDbInst(404)).toBeUndefined();
    });

    it('泛型实参决定取回类型：契约外字段可直接访问，无需调用点断言', () => {
        cacheDbInst(8, newInst(8, { dbs: new Map([['db1', 'db1']]) }));
        const inst = getCachedDbInst<FakeInst>(8);
        expectTypeOf(inst).toEqualTypeOf<FakeInst | undefined>();
        expect(inst?.dbs.get('db1')).toBe('db1');
        // 缺省泛型即契约类型：只能看到契约内的字段，越界访问会被编译拦住
        expectTypeOf(getCachedDbInst(8)).toEqualTypeOf<CachedDbInst | undefined>();
    });

    it('工厂返回类型即 getOrNewDbInst 的返回类型（T 由 createFn 推断）', () => {
        const inst = getOrNewDbInst(instInfo(9), instFactory);
        expectTypeOf(inst).toEqualTypeOf<FakeInst>();
        expect(inst.dbs).toBeInstanceOf(Map);
    });
});
