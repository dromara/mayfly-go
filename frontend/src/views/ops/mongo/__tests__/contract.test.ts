/**
 * Mongo 模块的跨语言契约与命令目录缓存测试。
 *
 * 三类守护：
 *  1. 中英文语言包键集一致，且模块里引用到的每个 key 两边都有词——缺词时界面会直接显示裸 key，
 *     这类错误只在用户屏幕上暴露，测试阶段拦住代价最小；
 *  2. 后端命令目录用 i18n key 描述命令，前端负责出词。后端新增命令而前端没补文案时，
 *     控制台下拉项会显示 `mongo.cmdXxxDesc`，这里按 Go 源码逐键核对；
 *  3. 命令目录缓存的模块级语义（命中缓存不再发请求、并发合流、失败不写缓存可重试）。
 */
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';

import en from '@/i18n/en/mongo';
import zh from '@/i18n/zh-cn/mongo';

const commandsRequestMock = vi.fn();
// catalog 依赖模块的 api 封装，mock 掉 IO 才能单独测缓存语义
vi.mock('../api', () => ({
    mongoApi: {
        commands: { request: (...args: unknown[]) => commandsRequestMock(...args) },
    },
}));

const MODULE_DIR = join(import.meta.dirname, '..');
const BACKEND_CATALOG = join(import.meta.dirname, '../../../../../../server/internal/mongo/application/mongodoc/cmdclass.go');

const localeKeys = (locale: { mongo: Record<string, string> }) => new Set(Object.keys(locale.mongo));

/** 收集模块内所有 `mongo.xxx` 形式的 key 引用（含 $t / t / i18n.global.t 与字面量） */
function referencedKeys(dir: string, collected = new Set<string>()): Set<string> {
    for (const entry of readdirSync(dir, { withFileTypes: true })) {
        const path = join(dir, entry.name);
        if (entry.isDirectory()) {
            // 测试自身会拼出 key 前缀，不参与扫描，避免自引用
            if (entry.name === '__tests__') {
                continue;
            }
            referencedKeys(path, collected);
            continue;
        }
        if (!/\.(vue|ts)$/.test(entry.name)) {
            continue;
        }
        const source = readFileSync(path, 'utf-8');
        for (const match of source.matchAll(/['"`]mongo\.([A-Za-z0-9_]+)['"`]/g)) {
            collected.add(match[1]);
        }
    }
    return collected;
}

describe('语言包键集', () => {
    it('中英文键集完全一致', () => {
        const zhKeys = localeKeys(zh);
        const enKeys = localeKeys(en);

        const onlyZh = [...zhKeys].filter((key) => !enKeys.has(key));
        const onlyEn = [...enKeys].filter((key) => !zhKeys.has(key));
        expect(onlyZh, `missing in en/mongo: ${onlyZh.join(', ')}`).toEqual([]);
        expect(onlyEn, `missing in zh-cn/mongo: ${onlyEn.join(', ')}`).toEqual([]);
        // 防空跑：本模块键很多，扫出个位数说明扫描方式失效了
        expect(zhKeys.size).toBeGreaterThan(40);
    });

    it('模块引用到的每个 key 两侧都有词', () => {
        const referenced = referencedKeys(MODULE_DIR);
        const zhKeys = localeKeys(zh);
        const enKeys = localeKeys(en);

        const missingZh = [...referenced].filter((key) => !zhKeys.has(key));
        const missingEn = [...referenced].filter((key) => !enKeys.has(key));
        expect(missingZh, `referenced but absent in zh-cn/mongo: ${missingZh.join(', ')}`).toEqual([]);
        expect(missingEn, `referenced but absent in en/mongo: ${missingEn.join(', ')}`).toEqual([]);
    });

    it('提示语里的 { } 只会是插值参数，不得出现 JSON 示例花括号', () => {
        // vue-i18n 把 `{"createdAt": -1}` 当插值占位解析并抛「Invalid token in placeholder」，
        // 该错误发生在读取它的 computed 里，被 Vue 静默吞掉后的现象是「点了没反应」而不是报错
        const badPlaceholder = (value: string) => [...value.matchAll(/\{([^{}]*)\}/g)].some(([, inner]) => !/^[A-Za-z0-9_]+$/.test(inner));

        for (const [locale, msgs] of [
            ['zh-cn', zh],
            ['en', en],
        ] as const) {
            const bad = Object.entries(msgs.mongo).filter(([, value]) => badPlaceholder(String(value)));
            expect(
                bad.map(([key]) => key),
                `${locale}/mongo 含非法插值占位的键`
            ).toEqual([]);
        }
    });

    it('语言包里没有取不到的死键', () => {
        // 后端 descKey 由服务端下发，不会出现在前端源码里，需要一并算作「被引用」
        const referenced = referencedKeys(MODULE_DIR);
        for (const key of backendDescKeys()) {
            referenced.add(key);
        }
        // 实例名等文案由 tag/menu 语言包负责，这里只校验 mongo 包内部
        const dead = [...localeKeys(zh)].filter((key) => !referenced.has(key));
        expect(dead, `unused keys in mongo locale: ${dead.join(', ')}`).toEqual([]);
    });
});

/** 从后端命令目录源码里提取全部 descKey（去掉 `mongo.` 前缀后的键名） */
function backendDescKeys(): string[] {
    const source = readFileSync(BACKEND_CATALOG, 'utf-8');
    const keys: string[] = [];
    for (const match of source.matchAll(/"(mongo\.[A-Za-z0-9_]+)"/g)) {
        keys.push(match[1].slice('mongo.'.length));
    }
    return keys;
}

describe('后端命令目录与前端语言包的契约', () => {
    it('目录里每条命令的描述都在两侧有词', () => {
        const descKeys = backendDescKeys();
        // 防空跑：目录条目数十条，扫不出来说明后端源码结构变了，测试须同步
        expect(descKeys.length).toBeGreaterThan(15);

        const zhKeys = localeKeys(zh);
        const enKeys = localeKeys(en);
        const missingZh = descKeys.filter((key) => !zhKeys.has(key));
        const missingEn = descKeys.filter((key) => !enKeys.has(key));
        expect(missingZh, `backend descKey missing in zh-cn/mongo: ${missingZh.join(', ')}`).toEqual([]);
        expect(missingEn, `backend descKey missing in en/mongo: ${missingEn.join(', ')}`).toEqual([]);
    });
});

describe('命令目录的共享缓存', () => {
    const specs = [{ name: 'ping', level: 'read', permission: '', needConfirm: false }];

    afterEach(() => {
        vi.clearAllMocks();
        // 模块级缓存是跨用例单例，reset 后动态 import 才能拿到干净实例
        vi.resetModules();
    });

    it('第二次读取命中缓存，不再发请求', async () => {
        commandsRequestMock.mockResolvedValue(specs);
        const mod = await import('../command/catalog');
        await mod.loadCommandCatalog(1);
        await mod.loadCommandCatalog(1);
        expect(commandsRequestMock).toHaveBeenCalledTimes(1);
    });

    it('在途请求共享同一个 Promise，并发调用只发一次', async () => {
        let release!: (value: unknown) => void;
        commandsRequestMock.mockReturnValue(new Promise((resolve) => (release = resolve)));

        const mod = await import('../command/catalog');
        const first = mod.loadCommandCatalog(2);
        const second = mod.loadCommandCatalog(2);
        release(specs);
        await Promise.all([first, second]);
        expect(commandsRequestMock).toHaveBeenCalledTimes(1);
    });

    it('失败不写缓存，下次仍可重试', async () => {
        commandsRequestMock.mockRejectedValueOnce(new Error('boom')).mockResolvedValueOnce(specs);
        const mod = await import('../command/catalog');

        await expect(mod.loadCommandCatalog(3)).rejects.toThrow('boom');
        await expect(mod.loadCommandCatalog(3)).resolves.toBe(specs);
        expect(commandsRequestMock).toHaveBeenCalledTimes(2);
    });

    it('不同实例各自缓存', async () => {
        commandsRequestMock.mockResolvedValue(specs);
        const mod = await import('../command/catalog');
        await mod.loadCommandCatalog(4);
        await mod.loadCommandCatalog(5);
        expect(commandsRequestMock).toHaveBeenCalledTimes(2);
    });

    it('clearCommandCatalog 指定实例只清该实例', async () => {
        commandsRequestMock.mockResolvedValue(specs);
        const mod = await import('../command/catalog');
        await mod.loadCommandCatalog(6);
        await mod.loadCommandCatalog(7);
        commandsRequestMock.mockClear();

        mod.clearCommandCatalog(6);
        await mod.loadCommandCatalog(6);
        await mod.loadCommandCatalog(7);
        expect(commandsRequestMock).toHaveBeenCalledTimes(1);
    });
});
