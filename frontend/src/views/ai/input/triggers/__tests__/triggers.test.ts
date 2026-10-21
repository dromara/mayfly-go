/**
 * 触发器子系统的两个纯函数契约（都是实测踩过的坑）
 *
 * 1. 触发词前置边界：提示词以中文写成为主，「…危险命令。@」这种「中文标点后紧跟触发符」
 *    必须触发；同时邮箱 test@qq 里的 @ 不能误触发。
 * 2. 技能过滤：内置技能显示名是中文（「数据库操作手册」），用户按技能码搜（/db →
 *    db-ops-guide）是主路径，只匹配显示名会显示「没有匹配项」。
 */
import { describe, expect, it } from 'vitest';
import { buildTriggerTokenRegex } from '../registry';
import { buildSkillItems } from '../skillTrigger';
import type { SkillItem } from '../../types';

/** 取光标前文本命中的触发词（模拟 useTriggerState 的检测口径） */
const matchToken = (before: string): string | null => {
    const m = before.match(buildTriggerTokenRegex());
    return m ? m[1] : null;
};

describe('触发词前置边界', () => {
    it('中文与中文标点后直接输入触发符也能触发', () => {
        expect(matchToken('巡检机器@')).toBe('@');
        expect(matchToken('禁止写操作。@')).toBe('@');
        expect(matchToken('先看手册，/db')).toBe('/db');
        expect(matchToken('目标：@ali')).toBe('@ali');
    });

    it('行首与空白后触发；字母/邮箱里的 @ 不误触发', () => {
        expect(matchToken('@')).toBe('@');
        expect(matchToken('  @ali')).toBe('@ali');
        expect(matchToken('test@qq')).toBeNull();
        expect(matchToken('word@')).toBeNull();
    });
});

describe('技能过滤', () => {
    const skills = [
        { id: 'db-ops-guide', name: '数据库操作手册', description: '库与慢查询' },
        { id: 'react-code-review', name: 'React 代码审查', description: '' },
    ] as SkillItem[];

    it('按技能码可搜到中文显示名的技能', () => {
        expect(buildSkillItems(skills, 'db').map((i) => i.id)).toEqual(['db-ops-guide']);
    });

    it('按显示名可搜到', () => {
        expect(buildSkillItems(skills, '审查').map((i) => i.id)).toEqual(['react-code-review']);
    });

    it('空查询列出全部，无匹配时为空', () => {
        expect(buildSkillItems(skills, '')).toHaveLength(2);
        expect(buildSkillItems(skills, '不存在的词')).toHaveLength(0);
    });
});
