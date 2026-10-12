import { mount } from '@vue/test-utils';
import { ElButton, ElDropdown, ElDropdownItem, ElDropdownMenu, ElIcon, ElPopconfirm, ElTooltip } from 'element-plus';
import { describe, expect, it, vi } from 'vitest';

// 权限过滤不能依赖登录态：约定权限码以 ':denied' 结尾即视为无权限
vi.mock('@/components/auth/auth', () => ({
    hasPerm: (code: string) => !code.endsWith(':denied'),
}));

// 确认框是全局单例，测试里换成可控桩
const { confirmMock } = vi.hoisted(() => ({ confirmMock: vi.fn(async () => true) }));
vi.mock('@/hooks/useI18n', () => ({ useI18nConfirm: confirmMock }));

import RowActions from '../RowActions.vue';
import { i18n } from '@/i18n';
import type { ActionItem, ActionOptions } from '../index';

// RowActionEntry 走的是全局 t（不依赖插件注入的组件上下文），断言也取同一口径
const t = (key: string) => i18n.global.t(key);

const row = { id: 7, running: false };

/** 生成 n 格操作项；denied 下标的格无权限；hiddenAt 下标的格仅本行不显示 */
const items = (n: number, opts: { denied?: number[]; hiddenAt?: number[]; confirmAt?: number[] } = {}): ActionItem[] =>
    Array.from({ length: n }, (_, i) => ({
        key: `k${i}`,
        label: `L${i}`,
        perm: opts.denied?.includes(i) ? 'a:denied' : 'a:ok',
        show: opts.hiddenAt?.includes(i) ? () => false : undefined,
        confirm: opts.confirmAt?.includes(i) ? { title: 'common.deleteConfirm' } : undefined,
        onClick: vi.fn(),
    }));

const mountActions = (list: ActionItem[], options: ActionOptions = {}) =>
    mount(RowActions, {
        props: { items: list, data: row, options },
        global: {
            mocks: { $t: (key: string) => key },
            components: { ElButton, ElTooltip, ElDropdown, ElDropdownMenu, ElDropdownItem, ElIcon, ElPopconfirm },
        },
    });

/**
 * 行内操作契约：内联/折叠的切分、权限与行级显隐对渲染的影响、确认门禁与回调入参。
 * 列宽侧的口径由 TableColumn 的用例覆盖，这里只管「渲染出什么」。
 */
describe('data-table/RowActions', () => {
    it('超出内联上限时折进「更多」，直显格数不超过上限', () => {
        const w = mountActions(items(6));
        // 3 直显 + 1 折叠触发器
        expect(w.findAllComponents(ElButton)).toHaveLength(4);
        expect(w.findAllComponents(ElDropdown)).toHaveLength(1);
    });

    it('折完只剩一条就不折：宁可四个直显，也不做「⋯ 里塞一个动作」', () => {
        const w = mountActions(items(4));
        expect(w.findAllComponents(ElDropdown)).toHaveLength(0);
        expect(w.findAllComponents(ElButton)).toHaveLength(4);
    });

    it('无权限的按钮既不渲染也不占位（受限账号看到更窄的操作列）', () => {
        // 5 格里 2 格无权限 → 可见 3 格 → 不触发折叠
        const w = mountActions(items(5, { denied: [3, 4] }));
        expect(w.findAllComponents(ElButton)).toHaveLength(3);
        expect(w.findAllComponents(ElDropdown)).toHaveLength(0);
    });

    it('先按行过滤再折叠：主动作不会被行状态挤进菜单', () => {
        // 6 格里第 4 格本行不显示 → 可见 5 格 → 3 直显 + 2 折叠
        expect(mountActions(items(6, { hiddenAt: [3] })).findAllComponents(ElButton)).toHaveLength(4);
        // 可见 4 格（含主动作）→ 全部直显，主动作不必点「更多」
        const w = mountActions(items(6, { hiddenAt: [2, 4, 5] }));
        expect(w.findAllComponents(ElDropdown)).toHaveLength(0);
        expect(w.findAllComponents(ElButton)).toHaveLength(3);
    });

    it('图标按钮不渲染可见文案，但 aria-label 取声明的 label', () => {
        const w = mountActions([{ key: 'del', label: 'common.delete', icon: 'delete', iconOnly: true, onClick: () => undefined }]);
        const btn = w.find('button');
        expect(btn.text()).toBe('');
        expect(btn.attributes('aria-label')).toBe(t('common.delete'));
        // 纯图标格自动带自解释提示（没写 tooltip 时用文案）
        expect(w.findAllComponents(ElTooltip)).toHaveLength(1);
    });

    it('列级图标开关：整列收图标，没 icon 的格依旧显文字', () => {
        const list: ActionItem[] = [
            { key: 'edit', label: 'common.edit', icon: 'edit', onClick: () => undefined },
            { key: 'detail', label: 'common.detail', onClick: () => undefined },
        ];
        const btns = mountActions(list, { iconOnly: true }).findAll('button');
        expect(btns[0].text()).toBe('');
        expect(btns[1].text()).toBe(t('common.detail'));
    });

    it('带文字的格不强制包 tooltip，避免与列的溢出提示叠成两层', () => {
        expect(mountActions([{ key: 'edit', label: 'common.edit', onClick: () => undefined }]).findAllComponents(ElTooltip)).toHaveLength(0);
    });

    it('就地气泡确认：点按钮不直接执行，由气泡的 confirm 事件驱动', async () => {
        const onClick = vi.fn();
        const w = mountActions([{ key: 'cancel', label: 'A', confirm: { title: 'flow.cancelProcessConfirm', pop: true }, onClick }]);
        expect(w.findAllComponents(ElPopconfirm)).toHaveLength(1);
        await w.find('button').trigger('click');
        expect(onClick).not.toHaveBeenCalled();
    });

    it('行级 loading 渲染到按钮上（本行进行中才转圈）', () => {
        const entry = (loading: (row: unknown) => boolean) => mountActions([{ key: 'play', label: 'A', loading, onClick: () => undefined }]);
        expect(entry(() => true).find('button').classes()).toContain('is-loading');
        expect(entry(() => false).find('button').classes()).not.toContain('is-loading');
    });

    it('点击直显按钮把整行数据与原生事件回传给 onClick', async () => {
        const list = items(2);
        await mountActions(list).findAllComponents(ElButton)[0].trigger('click');
        expect(list[0].onClick).toHaveBeenCalledTimes(1);

        const [row0, event0] = vi.mocked(list[0].onClick).mock.calls[0];
        // props 经响应式代理，按内容比对而非同一引用
        expect(row0).toEqual(row);
        // event 用于「按点击位置开新窗口」这类动作（如机器列表的终端）
        expect(event0).toBeInstanceOf(MouseEvent);
        expect(list[1].onClick).not.toHaveBeenCalled();
    });

    it('声明了 confirm 的动作要先过确认，取消则不执行', async () => {
        const list = items(2, { confirmAt: [0] });
        const btn = mountActions(list).findAllComponents(ElButton)[0];

        confirmMock.mockResolvedValueOnce(false);
        await btn.trigger('click');
        expect(list[0].onClick).not.toHaveBeenCalled();

        confirmMock.mockResolvedValueOnce(true);
        await btn.trigger('click');
        expect(list[0].onClick).toHaveBeenCalledTimes(1);
    });
});
