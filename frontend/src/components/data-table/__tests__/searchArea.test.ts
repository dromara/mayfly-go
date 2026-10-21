import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

/**
 * 搜索区共享层（DataTable + SearchForm）的两条观感/可达性守卫。
 *
 * 存在理由：这两处都在全站列表页的共用组件里，改错了会同时波及所有模块，而纯类型检查与
 * 单测都看不出来：
 * ① 折叠态（多数列表页的默认态）下整个搜索表单被高度裁剪层收起，表单里那个唯一的「重置」
 *    根本点不到；工具条只给了输入与搜索圆钮 —— 用户填了条件就没有一键清空的入口。
 * ② label 冒号由伪元素统一追加是既有决策（模板不硬编码冒号，中英文一致），但它带的左边距
 *    会把冒号推离文字、读成「名称 :」，在字段名后头制造一段无意义的空隙。
 */
const read = (name: string) => readFileSync(resolve(__dirname, '..', name), 'utf8');

const dataTable = read('DataTable.vue');
const searchForm = read('SearchForm/index.vue');

/** 取 CSS 里某个选择器的规则块；匹配不到就抛错，避免「没匹配到 → 断言恒真」 */
function cssBlock(selector: string): string {
    const escaped = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const block = searchForm.match(new RegExp(`${escaped}\\s*\\{[\\s\\S]*?\\}`));
    if (!block) {
        throw new Error(`未找到 ${selector} 规则块（改名或删除必须同步本守卫）`);
    }
    return block[0];
}

describe('搜索区的重置可达性', () => {
    /** 取工具条里那个重置按钮的开标签（属性顺序无关：按语义拼断言，不让重排属性变成误报） */
    function toolbarResetTag(): string {
        const tags = dataTable.match(/<el-button[\s\S]*?\/>/g) ?? [];
        const hit = tags.find((tag) => /icon="RefreshLeft"/.test(tag));
        if (!hit) {
            throw new Error('工具条没有 icon="RefreshLeft" 的重置按钮：折叠态下唯一的重置仍藏在被高度裁剪的表单里，点不到');
        }
        return hit;
    }

    it('工具条的重置入口在折叠态可见（展开表单时隐去，避免与表单内重置同屏双份）', () => {
        expect(toolbarResetTag()).toMatch(/!isShowSearch/);
    });

    it('工具条重置必须复用表单同一个 reset 方法（另写一套清空逻辑就是第二真源，会与表单态分叉）', () => {
        expect(toolbarResetTag()).toMatch(/@click="reset"/);
    });

    it('重置入口必须有可读名称（title/aria-label），不靠图标猜', () => {
        expect(toolbarResetTag()).toMatch(/\$t\('common\.reset'\)/);
    });
});

describe('搜索表单 label 冒号的排版', () => {
    it('冒号仍由伪元素统一追加（模板不硬编码冒号），但不再带左边距', () => {
        const block = cssBlock('.el-form-item__label::after');
        expect(block).toContain("content: ':'");
        expect(block, '冒号前的额外间距会把它推离文字，读成「名称 :」').not.toMatch(/margin-left|padding-left/);
    });

    it('模板里没有第二处硬编码冒号（否则两个真源，语言切换时不一致）', () => {
        expect(searchForm).not.toMatch(/<span>\s*\{\{[^}]*\}\}\s*[:：]/);
    });
});
