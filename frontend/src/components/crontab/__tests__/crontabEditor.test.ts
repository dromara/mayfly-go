/**
 * cron 编辑器交互回归。
 *
 * 钉住三条不变式：
 * 1. 读操作（切页签、翻预览）不改动表达式，每个页签只反映自己字段的规则
 * 2. 「日/周」互斥由模型维持，面板产出的一定是后端可解析的表达式
 * 3. 面板改的是草稿，点「确定」才回写 v-model
 *
 * el-dialog 用桩件替代：真实实现会 teleport 到 body 且带过渡，测试里只保留插槽渲染，
 * 让面板 DOM 留在组件树内可稳定查询。
 */
import { describe, expect, it, vi } from 'vitest';
import { defineComponent } from 'vue';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import CrontabInput from '../CrontabInput.vue';
import { CRON_FIELD_KEYS, defaultSpec, parseCronExpression, withRule, type CronFieldKey } from '../cronSpec';
import { useCronEditor } from '../useCronEditor';
import zhCommon from '@/i18n/zh-cn/common';

vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue([]) } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (key: string) => key } } }));

const i18n = createI18n({ legacy: false, globalInjection: true, locale: 'zh-cn', messages: { 'zh-cn': zhCommon } });

const ElDialogStub = defineComponent({
    name: 'ElDialog',
    props: { modelValue: { type: Boolean, default: false } },
    template: '<div v-if="modelValue" class="dialog-stub"><slot /><div class="dialog-footer"><slot name="footer" /></div></div>',
});

describe('草稿状态', () => {
    it('载入后未被改动时表达式保持稳定', () => {
        const editor = useCronEditor();
        editor.load('0 0 9 ? * 1-5');
        const expression = editor.expression.value;
        expect(expression).toBe('0 0 9 ? * 1-5');
        // 读操作不产生任何回写，等价于「切页签看一眼」
        expect(editor.expression.value).toBe(expression);
    });

    it('原表达式非法时报错并回落默认规则，面板继续可配', () => {
        const editor = useCronEditor();
        editor.load('0 0 0 L * ?');
        expect(editor.error.value).toMatchObject({ key: 'components.crontab.errSegment', field: 'day' });
        expect(editor.expression.value).toBe('* * * * * ?');
        editor.setRule('hour', { kind: 'list', values: [3] });
        expect(editor.error.value).toBeNull();
        expect(editor.expression.value).toBe('* * 3 * * ?');
    });

    it('描述符表达式面板只读，重置后才可继续编辑', () => {
        const editor = useCronEditor();
        editor.load('@every 5m');
        expect(editor.descriptorOnly.value).toBe(true);
        expect(editor.error.value).toBeNull();
        editor.reset();
        expect(editor.descriptorOnly.value).toBe(false);
        expect(editor.expression.value).toBe('* * * * * ?');
    });

    it('日与周互斥由模型维持', () => {
        const byWeek = withRule(defaultSpec(), 'week', { kind: 'cycle', from: 1, to: 5 });
        expect(byWeek.day).toEqual({ kind: 'none' });
        const byDay = withRule(byWeek, 'day', { kind: 'list', values: [1] });
        expect(byDay.week).toEqual({ kind: 'none' });
        expect(parseCronExpression('* * * 1 * ?').error).toBeNull();
    });
});

describe('面板交互', () => {
    /** 挂载并打开面板，返回按字段页签定位控件的助手（el-tabs 会保留已访问页签的 DOM，故必须限定到目标 pane） */
    async function openPanel(modelValue: string) {
        const wrapper = mount(CrontabInput, {
            props: { modelValue },
            global: { plugins: [ElementPlus, i18n], stubs: { 'el-dialog': ElDialogStub } },
        });
        const byText = (text: string) => wrapper.findAll('button').find((button) => button.text() === text);
        // 触发按钮只带图标，按容器定位
        await wrapper.find('.el-input-group__prepend button').trigger('click');
        await flushPromises();
        const pane = (key: CronFieldKey) => wrapper.find(`#pane-${key}`);
        return {
            wrapper,
            expression: () => wrapper.find('.cron-expression').text(),
            segments: () => wrapper.findAll('.cron-cell-value').map((cell) => cell.text()),
            tab: async (key: CronFieldKey) => {
                await wrapper.find(`#tab-${key}`).trigger('click');
                await flushPromises();
            },
            /** 该字段页签里选中的规则类型文案 */
            ruleKindOf: (key: CronFieldKey) => pane(key).find('.el-radio-group label.is-active span').text(),
            /** 该字段页签切换到指定规则类型 */
            pickKind: async (key: CronFieldKey, index: number) => {
                await pane(key).findAll('.el-radio-group input')[index].setValue(true);
                await flushPromises();
            },
            chipsOf: (key: CronFieldKey) => pane(key).findAll('.cron-chip'),
            /** 点击该字段页签里的「全选/清空」链接 */
            clickLink: async (key: CronFieldKey, index: number) => {
                await pane(key).findAll('.el-link')[index].trigger('click');
                await flushPromises();
            },
            clickChip: async (key: CronFieldKey, index: number) => {
                await pane(key).findAll('.cron-chip')[index].trigger('click');
                await flushPromises();
            },
            setNumber: async (key: CronFieldKey, index: number, value: string) => {
                await pane(key).findAll('.el-input-number input')[index].setValue(value);
                await flushPromises();
            },
            clickButton: async (text: string) => {
                await byText(text)?.trigger('click');
                await flushPromises();
            },
        };
    }

    it('打开面板即按输入框取值回显各段与运行时间', async () => {
        const panel = await openPanel('0 0 9 ? * 1-5');
        expect(panel.expression()).toBe('0 0 9 ? * 1-5');
        // 周限定周一到周五，日即为不指定
        expect(panel.segments()).toEqual(['0', '0', '9', '?', '*', '1-5']);
        // 预览给出具体时刻，而不是旧的「计算结果中」占位
        const times = panel.wrapper.findAll('.cron-preview-list li');
        expect(times).toHaveLength(5);
        expect(times[0].text()).toMatch(/^\d{4}-\d{2}-\d{2} 09:00:00$/);
    });

    it('切换页签不改表达式，各页签只反映自己的规则', async () => {
        const panel = await openPanel('0 0 9 ? * 1-5');
        for (const key of CRON_FIELD_KEYS) {
            await panel.tab(key);
            expect(panel.expression()).toBe('0 0 9 ? * 1-5');
        }
        expect(panel.ruleKindOf('hour')).toBe('指定');
        expect(panel.ruleKindOf('min')).toBe('指定');
        expect(panel.ruleKindOf('day')).toBe('不指定');
        expect(panel.ruleKindOf('week')).toBe('区间');
        expect(panel.ruleKindOf('month')).toBe('全部');
    });

    it('限定日期时星期自动改为不指定', async () => {
        const panel = await openPanel('0 0 9 ? * 1-5');
        await panel.tab('day');
        expect(panel.ruleKindOf('day')).toBe('不指定');
        await panel.pickKind('day', 4);
        expect(panel.chipsOf('day')).toHaveLength(31);
        expect(panel.expression()).toBe('0 0 9 1 * ?');
        await panel.clickChip('day', 5);
        expect(panel.expression()).toBe('0 0 9 1,6 * ?');
        // 日被限定后，周页签跟着反映为不指定，且切过去不会再把表达式改掉
        await panel.tab('week');
        expect(panel.ruleKindOf('week')).toBe('不指定');
        expect(panel.expression()).toBe('0 0 9 1,6 * ?');
    });

    it('全选回写 `*` 而不是全量列表', async () => {
        const panel = await openPanel('0 0 0 * * ?');
        await panel.tab('hour');
        await panel.pickKind('hour', 3);
        expect(panel.expression()).toBe('0 0 0 * * ?');
        await panel.clickLink('hour', 0);
        expect(panel.ruleKindOf('hour')).toBe('全部');
        expect(panel.expression()).toBe('0 0 * * * ?');
    });

    it('步长写回完整规则', async () => {
        const panel = await openPanel('0 0 0 * * ?');
        await panel.tab('min');
        await panel.pickKind('min', 2);
        expect(panel.expression()).toBe('0 0/1 0 * * ?');
        await panel.setNumber('min', 1, '15');
        expect(panel.expression()).toBe('0 0/15 0 * * ?');
    });

    it('区间不弹回刚输入的值，也不产生回绕写法', async () => {
        const panel = await openPanel('0 0 0 * * ?');
        await panel.tab('hour');
        await panel.pickKind('hour', 1);
        expect(panel.expression()).toBe('0 0 0-0 * * ?');
        // 起始抬高到超过截止时，截止跟着抬高
        await panel.setNumber('hour', 0, '9');
        expect(panel.expression()).toBe('0 0 9-9 * * ?');
        await panel.setNumber('hour', 1, '18');
        expect(panel.expression()).toBe('0 0 9-18 * * ?');
        // 截止压低到低于起始时，起始跟着压低
        await panel.setNumber('hour', 1, '5');
        expect(panel.expression()).toBe('0 0 5-5 * * ?');
    });

    it('草稿只在点确定后回写，取消不改输入框', async () => {
        const panel = await openPanel('0 0 9 ? * 1-5');
        await panel.tab('hour');
        // 周一到周五的 9 点改为 3 点
        await panel.clickChip('hour', 3);
        await panel.clickChip('hour', 9);
        expect(panel.expression()).toBe('0 0 3 ? * 1-5');
        expect(panel.wrapper.emitted('update:modelValue')).toBeUndefined();
        await panel.clickButton('取消');
        expect(panel.wrapper.emitted('update:modelValue')).toBeUndefined();
    });

    it('确定把草稿写回 v-model', async () => {
        const panel = await openPanel('0 0 9 ? * 1-5');
        await panel.tab('hour');
        await panel.clickChip('hour', 3);
        await panel.clickChip('hour', 9);
        await panel.clickButton('确定');
        expect(panel.wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['0 0 3 ? * 1-5']);
    });

    it('预设一次点齐常用配置', async () => {
        const panel = await openPanel('* * * * * ?');
        await panel.clickButton('工作日九点');
        expect(panel.expression()).toBe('0 0 9 ? * 1-5');
    });

    it('非法表达式在面板内给出可读原因而不是静默兜底', async () => {
        const panel = await openPanel('0 0 0 32 * ?');
        const alert = panel.wrapper.find('.el-alert');
        expect(alert.exists()).toBe(true);
        expect(alert.text()).toContain('32');
        expect(alert.text()).toContain('日');
    });
});

describe('输入框提示行', () => {
    const mountInput = (modelValue: string) =>
        mount(CrontabInput, { props: { modelValue }, global: { plugins: [ElementPlus, i18n], stubs: { 'el-dialog': ElDialogStub } } });

    it('合法表达式给出一行下次运行时间', () => {
        expect(mountInput('0 0 0 * * ?').find('.cron-hint').text()).toMatch(/^下次运行：\d{4}-\d{2}-\d{2} 00:00:00$/);
    });

    it('敲到一半不报错，段数齐了才提示不合法', () => {
        expect(mountInput('0 0 0 L').find('.cron-hint').exists()).toBe(false);
        expect(mountInput('0 0 0 L * ?').find('.cron-hint').text()).toBe('表达式无法解析');
    });
});
