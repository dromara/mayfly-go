import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { RUN_ROLES, RunRoleEnum } from '../enums';

/**
 * 运行形态（runRole）的跨语言契约守卫。
 *
 * 后端 `entity.RunRole*` 是值域真源，前端 `RunRoleEnum` 是它的展示镜像。两者靠人记忆同步时，
 * 漂移的表现不是报错而是「新形态的运行标签退化成显示原始键串」——用户看着像数据异常，
 * 而所有门禁全绿。这里直接读后端源码比对，把同步变成会红的机制。
 */
const BACKEND_RUN_ROLE = resolve(__dirname, '../../../../../server/internal/agentops/domain/entity/run_role.go');

/** 从后端常量块里取出全部形态值（形如 `\tRunRoleOrchStep = "orch_step"`） */
function backendRunRoleValues(): string[] {
    const source = readFileSync(BACKEND_RUN_ROLE, 'utf8');
    return [...source.matchAll(/^\tRunRole\w+\s*=\s*"([^"]+)"/gm)].map((m) => m[1]);
}

describe('运行形态的前后端值集合一致', () => {
    it('前端登记的形态恰好覆盖后端声明的全集', () => {
        const backend = backendRunRoleValues();
        expect(backend.length, '后端未解析到任何 RunRole 常量（常量写法变了要同步本守卫）').toBeGreaterThan(0);
        expect([...RUN_ROLES].sort()).toEqual([...backend].sort());
    });

    it('每个形态都有 i18n 标签键，且键名遵循 agentops.role<驼峰> 约定', () => {
        // 漏配标签时 EnumTag/enumLabel 会把 i18n key 原文吐到界面上，比显示原始值更糟
        for (const [, ev] of Object.entries(RunRoleEnum)) {
            expect(ev.label).toMatch(/^agentops\.role[A-Z][A-Za-z]*$/);
        }
    });

    it('中英两份字典都补齐了全部形态标签', () => {
        for (const lang of ['zh-cn', 'en']) {
            const source = readFileSync(resolve(__dirname, `../../../i18n/${lang}/agentops.ts`), 'utf8');
            for (const [, ev] of Object.entries(RunRoleEnum)) {
                const key = ev.label.split('.')[1];
                expect(source, `${lang} 缺形态标签 ${key}`).toContain(`${key}:`);
            }
        }
    });

    it('后端历史前缀表里的角色都在已声明值域内（防死项）', () => {
        const source = readFileSync(BACKEND_RUN_ROLE, 'utf8');
        const declared = new Set(backendRunRoleValues());
        const referenced = [...source.matchAll(/Role:\s*(RunRole\w+)/g)].map((m) => m[1]);
        expect(referenced.length, '未解析到历史前缀表（写法变了要同步本守卫）').toBeGreaterThan(0);
        const names = new Set([...source.matchAll(/^\tRunRole(\w+)\s*=\s*"([^"]+)"/gm)].map((m) => `${m[1]}=${m[2]}`));
        for (const name of referenced) {
            expect([...names].some((kv) => kv.startsWith(`${name.slice('RunRole'.length)}=`)), `历史前缀表引用了未声明的 ${name}`).toBe(true);
        }
        expect(declared.size).toBe(RUN_ROLES.length);
    });
});
