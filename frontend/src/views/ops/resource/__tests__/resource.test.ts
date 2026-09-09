import { afterEach, describe, expect, it, vi } from 'vitest';

import { defineResourceConfig, getResourceConfig, getResourceConfigs, getResourceTypes, registerResource } from '../resourceRegistry';

/** 构造最小可用配置（根节点 kind 已收敛到贡献者 resourceType 声明，配置不再携带） */
const makeConf = (type: number | string, order?: number) => ({ resourceType: type, order }) as any;

/**
 * 资源注册中心单测：
 * - 注册 / 查询 / 类型清单
 * - 重复注册须在开发环境显式告警（防止静默覆盖，同 dbm 注册表同名覆盖问题）
 * - order 排序语义：有 order 的升序在前，无 order 的保持注册顺序排在其后
 */
describe('资源注册中心', () => {
    afterEach(() => {
        vi.restoreAllMocks();
    });

    it('注册后可通过类型查询配置与类型清单', () => {
        const conf = makeConf('test-registry-a', 1);
        registerResource('test-registry-a', conf);

        expect(getResourceConfig('test-registry-a')).toBe(conf);
        expect(getResourceTypes()).toContain('test-registry-a');
        expect(getResourceConfigs()).toContain(conf);
    });

    it('重复注册同一资源类型时开发环境给出告警', () => {
        const warnSpy = vi.spyOn(console, 'warn').mockImplementation(() => {});
        const first = makeConf('test-registry-dup', 1);
        registerResource('test-registry-dup', first);
        expect(warnSpy).not.toHaveBeenCalled();

        const second = makeConf('test-registry-dup', 2);
        registerResource('test-registry-dup', second);
        expect(warnSpy).toHaveBeenCalledTimes(1);
        expect(String(warnSpy.mock.calls[0][0])).toContain('test-registry-dup');
        // 后注册者覆盖先注册者
        expect(getResourceConfig('test-registry-dup')).toBe(second);
    });

    it('defineResourceConfig 原样透传配置（类型由函数签名守卫，替代 as 断言）', () => {
        const conf = defineResourceConfig({ order: 99, resourceType: 'test-define', manager: { permCode: 'x', componentConf: { name: 'test' } } });
        registerResource('test-define', conf);
        expect(getResourceConfig('test-define')).toBe(conf);
    });

    it('getResourceConfigs 按 order 升序排列且无 order 的排在其后', () => {
        registerResource('test-order-b', makeConf('test-order-b', 2));
        registerResource('test-order-a', makeConf('test-order-a', 1));
        registerResource('test-order-none', makeConf('test-order-none'));

        const types = getResourceConfigs().map((c) => c.resourceType);
        expect(types.indexOf('test-order-a')).toBeLessThan(types.indexOf('test-order-b'));
        expect(types.indexOf('test-order-b')).toBeLessThan(types.indexOf('test-order-none'));
    });
});
