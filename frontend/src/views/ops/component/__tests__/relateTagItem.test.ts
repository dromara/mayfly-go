/**
 * 资源分组必填守卫。
 *
 * 资源分组（tagCodePaths）决定资源归属与团队隔离，后端所有资源入参都以 binding:"required" 声明；
 * 前端一旦漏声明，界面上既没有星号、也不会拦截空分组——用户以为可选，直到提交才被后端 400 顶回来。
 * 本用例锁两件事：
 *  1. 工厂产出的字段确实带必填规则（且文案是下拉式控件的「请选择」，不是默认的「请输入」）；
 *  2. 所有使用 TagTreeSelect 的资源编辑表单都引用该工厂，不再各写一份（防止新增资源类型时漏标）。
 */
import { readFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { describe, expect, it } from 'vitest';
import { relateTagItem } from '../relateTagItem';

/** src/views/ops：资源编辑表单所在目录 */
const OPS_DIR = join(import.meta.dirname, '..', '..');

/** 递归收集 ops 下的组件源码（跳过测试目录） */
const collectVueFiles = (dir: string): string[] =>
    readdirSync(dir, { withFileTypes: true }).flatMap((entry) => {
        if (entry.name === '__tests__') {
            return [];
        }
        const full = join(dir, entry.name);
        if (entry.isDirectory()) {
            return collectVueFiles(full);
        }
        return entry.name.endsWith('.vue') ? [full] : [];
    });

describe('资源分组字段声明', () => {
    it('必填规则：空分组判失败、有值放行', () => {
        const item = relateTagItem();
        expect(item.prop).toBe('tagCodePaths');
        expect(item.label).toBe('tag.relateTag');

        const rules = Array.isArray(item.rules) ? item.rules : [];
        expect(rules).toHaveLength(1);
        expect(rules[0].required).toBe(true);
        // 关联标签是下拉式多选树，提示必须是「请选择…」而不是「请输入…」
        expect(rules[0].message).toBe('请选择资源分组');
    });

    it('每个使用 TagTreeSelect 的资源编辑表单都引用 relateTagItem', () => {
        const forms = collectVueFiles(OPS_DIR).filter((file) => readFileSync(file, 'utf8').includes('TagTreeSelect'));

        // 当前 8 类资源：机器 / 数据库 / Redis / MongoDB / ES / Milvus / Kafka / 容器
        expect(forms.length).toBe(8);

        for (const file of forms) {
            const source = readFileSync(file, 'utf8');
            expect(source, `${file} 的资源分组未走 relateTagItem（必填声明唯一出处）`).toContain('relateTagItem<');
        }
    });
});
