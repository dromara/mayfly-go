/**
 * 脱敏规则编辑抽屉回归
 *
 * 1) maskParamsPlaceholder 文案含字面量 JSON 花括号，vue-i18n 会把 {} 当插值占位符，
 *    未用 {'{'} / {'}'} 转义时渲染期抛 "Message compilation error: Invalid token in placeholder"
 * 2) 抽屉内 status el-switch（activeValue 1 / inactiveValue 0）在各打开路径挂载时都必须拿到 0/1
 *    （宿主以 visible=true 直接挂载时不回填的历史缺陷由 useAutoFormHost 修复，此处覆盖真实调用链）
 */
import { describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import MaskRuleEdit from '../MaskRuleEdit.vue';
import zhDb from '@/i18n/zh-cn/db';
import enDb from '@/i18n/en/db';

vi.mock('@/components/monaco/MonacoEditor.vue', () => ({
    default: defineComponent({ name: 'MonacoEditorStub', props: ['modelValue', 'options'], template: '<div class="monaco-stub" />' }),
}));
vi.mock('@/components/svg-icon/index.vue', () => ({
    default: defineComponent({ name: 'SvgIconStub', template: '<i class="svg-icon-stub" />' }),
}));
// 切断 api 模块的重组件依赖链（system-message 等），本测试不触发网络
vi.mock('@/views/ops/db/api', () => ({
    dbApi: {},
    dbMaskApi: { maskRules: {}, saveMaskRule: { request: vi.fn() }, updateMaskRule: { request: vi.fn() }, deleteMaskRule: { request: vi.fn() } },
}));
// 全局 i18n 单例（@/common/rule 的 Rules 依赖它），组件文案由下方本地实例提供
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue([]) } }));

const common = { status: '状态', enable: '启用', disable: '禁用', remark: '备注', cancel: '取消', confirm: '确定' };
const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    fallbackLocale: 'en',
    messages: { 'zh-cn': { db: zhDb.db, common }, en: { db: enDb.db, common } },
});

/** 后端 t_db_mask_rule 真实行结构（status int8 -> number） */
const ruleRow = {
    id: 1780000003,
    name: '手机号',
    matchType: 1,
    pattern: '(?i)phone$|mobile$',
    algorithm: 'partial',
    params: '',
    status: 1,
    weight: 100,
    remark: '',
};

const switchWarns = (spy: ReturnType<typeof vi.spyOn>) =>
    spy.mock.calls.map((c) => c.join(' ')).filter((s) => s.includes('active-value or inactive-value'));

/** 挂载编辑抽屉（openAtMount=false 时模拟真实点击：先关闭挂载再置 visible） */
const openEdit = async (data: Record<string, unknown> | null, openAtMount: boolean) => {
    const warnSpy = vi.spyOn(console, 'warn');
    document.body.innerHTML = '';
    const wrapper = mount(MaskRuleEdit, {
        attachTo: document.body,
        props: { visible: openAtMount, title: '编辑规则', data },
        global: { plugins: [ElementPlus, i18n] },
    });
    if (!openAtMount) {
        await wrapper.setProps({ visible: true });
    }
    await flushPromises();
    // 抽屉展开 + teleport 渲染需额外一轮宏任务
    await new Promise((resolve) => setTimeout(resolve, 50));
    const switches = document.body.querySelectorAll('.el-switch').length;
    const nameInput = document.body.querySelector('.el-drawer input[type="text"]') as HTMLInputElement | null;
    return { switches, nameValue: nameInput?.value, warns: switchWarns(warnSpy) };
};

describe('MaskRuleEdit 脱敏规则编辑', () => {
    it('params 提示文案的字面量 JSON 花括号可被 vue-i18n 编译（中英双语）', async () => {
        for (const locale of ['zh-cn', 'en'] as const) {
            i18n.global.locale.value = locale;
            expect(i18n.global.t('db.maskParamsPlaceholder')).toContain('{"keepFirst":3,"keepLast":4}');
        }
    });

    it('点击编辑（先挂载后打开）回填生效且无 model-value 警告', async () => {
        const { switches, nameValue, warns } = await openEdit(ruleRow, false);
        expect(switches).toBe(1);
        expect(nameValue).toBe('手机号');
        expect(warns).toEqual([]);
    });

    it('宿主以 visible=true 直接挂载（异步组件/HMR）同样不报警告', async () => {
        for (const data of [ruleRow, { ...ruleRow, status: 0 }, null]) {
            const { switches, warns } = await openEdit(data, true);
            expect(switches).toBe(1);
            expect(warns).toEqual([]);
        }
    });
});
