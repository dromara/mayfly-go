/**
 * 架构守护：低层模块不得静态依赖应用图
 *
 * `common/`（请求与 Api 封装）与 `hooks/` 处在依赖图底部，而 `@/router` 会连带拉起
 * syssocket → 系统消息组件 → 各业务 `api.ts`；那些模块在**顶层**就调用 `Api.newGet()`。
 * 于是「common/Api.ts → hooks → router → …→ 业务 api → common/Api.ts」构成环，
 * 从环上任一点进入都会读到尚未初始化的 `class Api`，偶发
 * `Cannot access 'Api' before initialization`，表现是懒加载面板整块空白且静态检查全绿。
 */
import { describe, expect, it } from 'vitest';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';

const SRC_DIR = join(import.meta.dirname, '..');

const listFiles = (dir: string, exts: string[]): string[] =>
    readdirSync(dir).flatMap((name) => {
        const full = join(dir, name);
        if (statSync(full).isDirectory()) {
            return name === '__tests__' ? [] : listFiles(full, exts);
        }
        return exts.some((ext) => name.endsWith(ext)) ? [full] : [];
    });

/** 只看静态 import：`await import('...')` 是刻意的延迟求值，不参与成环 */
const staticImports = (file: string): string[] =>
    [...readFileSync(file, 'utf-8').matchAll(/(?:^|\n)\s*import(?:\s+type)?\s+(?:[^'"]*?from\s*)?['"]([^'"]+)['"]/g)].map((match) => match[1]);

describe('低层模块的依赖方向', () => {
    const lowLayerFiles = [...listFiles(join(SRC_DIR, 'common'), ['.ts', '.vue']), ...listFiles(join(SRC_DIR, 'hooks'), ['.ts', '.vue'])];

    it('common/ 与 hooks/ 不得静态 import @/router（会把应用图与业务 api 拉进依赖环）', () => {
        const violations = lowLayerFiles
            .filter((file) => staticImports(file).some((spec) => spec === '@/router' || spec === '@/router/index'))
            .map((file) => file.replace(SRC_DIR + '/', ''));
        expect(violations, `需改为在函数内 await import('@/router')：${violations.join(', ')}`).toEqual([]);
    });

    it('防空跑：确实扫到了低层模块', () => {
        expect(lowLayerFiles.length).toBeGreaterThan(10);
    });
});
