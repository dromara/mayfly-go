import { beforeEach, describe, expect, it } from 'vitest';
import { allResourceOpTabs, activeResourceOpTabKey, type ResourceOpTab } from '@/views/ops/resource/resourceOp';
import { findDbWorkspaceTabKey } from '../helpers';

/**
 * 库级操作复用哪个工作区 tab 的判据守卫
 *
 * tab key 形如 `实例code.库code.库名.schema`（四段以 `.` 分隔）。库节点无 schema 段 → key 以 `.` 结尾；
 * schema 分支 tab → key 带该段。库级命令按库级 key（`...db.`）归一到该库已有的 schema 分支工作区，
 * 否则会凭空多出一个 tab（mysql schema 段为空，库级 key 本身即已存在 → 精确命中，只往已有 tab 加子 tab）。
 * 用 `.` 作段边界（与 instCode/dbCode 同规则），使「库名本身含 `/`」不再与 schema 段混淆。
 */

const tab = (key: string): ResourceOpTab => ({ key, name: key, component: {} as ResourceOpTab['component'] });

const openTabs = (...keys: string[]) => {
    allResourceOpTabs.clear();
    keys.forEach((key) => allResourceOpTabs.set(key, tab(key)));
};

beforeEach(() => {
    allResourceOpTabs.clear();
    activeResourceOpTabKey.value = '';
});

describe('findDbWorkspaceTabKey', () => {
    it('pg：该库只有 schema 分支工作区时复用该分支，而不是再开一个库级 tab', () => {
        openTabs('inst.dbCode.mydb.public');
        expect(findDbWorkspaceTabKey('inst.dbCode.mydb.')).toBe('inst.dbCode.mydb.public');
    });

    it('多个 schema 分支时优先当前激活的那个，避免把用户正在看的 tab 切走', () => {
        openTabs('inst.dbCode.mydb.public', 'inst.dbCode.mydb.report');
        activeResourceOpTabKey.value = 'inst.dbCode.mydb.report';
        expect(findDbWorkspaceTabKey('inst.dbCode.mydb.')).toBe('inst.dbCode.mydb.report');
    });

    it('mysql：库级 key（schema 段为空、以 . 结尾）本身已存在即精确命中，不被其他库的分支干扰', () => {
        openTabs('inst.dbCode.other.', 'inst.dbCode.mydb.');
        activeResourceOpTabKey.value = 'inst.dbCode.other.';
        expect(findDbWorkspaceTabKey('inst.dbCode.mydb.')).toBe('inst.dbCode.mydb.');
    });

    it('该库尚无任何工作区时返回 undefined，由调用方按库级 key 新建', () => {
        openTabs('inst.dbCode.other.', 'inst2.dbCode.mydb.public');
        expect(findDbWorkspaceTabKey('inst.dbCode.mydb.')).toBeUndefined();
    });

    it('前缀相近的库名不误命中（mydb 不得拿走 mydb2/public 的工作区）', () => {
        openTabs('inst.dbCode.mydb2.public');
        expect(findDbWorkspaceTabKey('inst.dbCode.mydb.')).toBeUndefined();
    });

    it('库名本身含 / 时，另一库 foo 的库级命令不得误命中 foo/bar 库的工作区', () => {
        // 若沿用把 schema 折进 `/` 的旧规则，`foo` 的前缀 `foo/` 会误吞名为 `foo/bar` 的库级 tab
        openTabs('inst.dbCode.foo/bar.');
        expect(findDbWorkspaceTabKey('inst.dbCode.foo.')).toBeUndefined();
    });
});
