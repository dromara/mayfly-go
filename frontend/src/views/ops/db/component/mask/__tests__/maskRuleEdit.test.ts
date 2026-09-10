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

// 占位文案由 AutoForm 内部 t() 渲染，测试用的 common 必须与真实命名空间同形，否则“无缺 key 告警”断言不成立
const common = { status: '状态', enable: '启用', disable: '禁用', remark: '备注', cancel: '取消', confirm: '确定', pleaseInput: '请输入{label}', pleaseSelect: '请选择{label}' };
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

/**
 * i18n 缺 key 告警：AutoForm 会对 item.label / option.label 再做一次 $t()，
 * 声明处预先 t() 会把已译文当 key 传入（运行期只告警不报错，切换语言后文案还会固定在挂载时的语种）
 */
const i18nMissWarns = (spy: ReturnType<typeof vi.spyOn>) =>
    spy.mock.calls.map((c) => c.join(' ')).filter((s) => s.includes('locale messages'));

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
    return { switches, nameValue: nameInput?.value, warns: switchWarns(warnSpy), i18nWarns: i18nMissWarns(warnSpy), algoOptions: algoOptionTexts() };
};

/** 算法下拉的实际渲染文本（已选项以 input[value] 呈现，未展开时取可见的 el-select 触发器） */
const algoOptionTexts = (): string[] => {
    return Array.from(document.body.querySelectorAll('.el-select__selected-item, .el-select__placeholder'))
        .map((el) => (el.textContent ?? '').trim())
        .filter(Boolean);
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

    it('中英双语挂载均无 i18n 缺 key 告警，且算法下拉展示译文而非原始 key', async () => {
        for (const locale of ['zh-cn', 'en'] as const) {
            i18n.global.locale.value = locale;
            const { algoOptions, i18nWarns } = await openEdit(ruleRow, false);
            expect(i18nWarns, `[${locale}] 存在未被解析的 i18n key`).toEqual([]);
            expect(algoOptions.join('|'), `[${locale}] 算法下拉未展示译文`).toContain(locale === 'zh-cn' ? '部分遮盖' : 'Partial');
            expect(algoOptions.join('|')).not.toContain('db.maskAlgo');
        }
        i18n.global.locale.value = 'zh-cn';
    });

    it('宿主以 visible=true 直接挂载（异步组件/HMR）同样不报警告', async () => {
        for (const data of [ruleRow, { ...ruleRow, status: 0 }, null]) {
            const { switches, warns } = await openEdit(data, true);
            expect(switches).toBe(1);
            expect(warns).toEqual([]);
        }
    });
});
