/**
 * 指标趋势渲染的回归测试。
 *
 * 响应经 `JSONBig({ storeAsString: true })`（src/hooks/useRequest.ts）解析后，长小数会回传成**字符串**，
 * 而 types.ts 里这些字段声明为 number —— 类型检查、lint、构建全都发现不了。
 * 一旦对它们调 `.toFixed()` 就抛 `x.toFixed is not a function`，趋势图整块静默空白（只剩坐标轴容器）。
 * 本测试用字符串数值喂进去，钉住「趋势图必须渲染出数字 series」。
 */
import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import { defineComponent, h } from 'vue';

// 收集传给 ECharts 的 option，用于断言趋势配置真的生成了
const captured: Array<{ height: string; option: Record<string, unknown> }> = [];

vi.mock('@/components/echarts/ECharts.vue', () => ({
    default: defineComponent({
        name: 'EChartsStub',
        props: { height: { type: [String, Number], default: '' }, option: { type: Object, default: () => ({}) } },
        // 在 render 里采集（setup 只跑一次，首次 option 还是空对象）
        setup(props) {
            return () => {
                captured.push({ height: String(props.height), option: props.option as Record<string, unknown> });
                return h('div', { class: 'echarts-stub' });
            };
        },
    }),
}));

// 指标点与真实回传形态一致：float 是字符串，累计字节也可能是字符串
const METRIC_ROWS = [
    { collectTime: '2026-10-06T08:00:00+08:00', cpuUsage: '18.199996948242188', memUsage: '74.2127414324183', diskUsage: '20.483708366554783', netRx: '1000000', netTx: '2000000' },
    { collectTime: '2026-10-06T08:05:00+08:00', cpuUsage: '33.5', memUsage: '75.5', diskUsage: '21.25', netRx: '1524288', netTx: '2548736' },
];

vi.mock('@/views/ops/machine/api', () => ({
    machineApi: {
        stats: {
            request: vi
                .fn()
                .mockResolvedValue({
                    memInfo: { total: '1691451392', available: '424046592' },
                    cpu: { idle: '99.5534', iowait: '0.1', user: '0.2', system: '0.1' },
                    load: { load1: '0.02', load5: '0.09', load10: '0.06' },
                    // parseNetInter 会对 netIntf 取 Object.keys，缺失会先于取数抛错
                    netIntf: { eth0: { rx: '1', tx: '2' } },
                    fsInfos: [],
                }),
        },
    },
    metricApi: { range: { request: vi.fn().mockResolvedValue(METRIC_ROWS) }, healthOverview: { request: vi.fn().mockResolvedValue([]) } },
}));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue({}) } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }));

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

describe('MachineStats 指标趋势', () => {
    it('指标字段以字符串回传时，趋势图仍生成数字 series（不静默空白）', async () => {
        captured.length = 0;
        const MachineStats = (await import('../MachineStats.vue')).default;
        mount(MachineStats, {
            props: { visible: true, machineId: 1, title: 't' },
            global: { plugins: [ElementPlus, i18n] },
            attachTo: document.body,
        });
        await flushPromises();

        const trend = captured.find((c) => c.height === '240' && Array.isArray((c.option as { series?: unknown[] }).series));
        expect(trend, '趋势图未生成 option（多半是取数/渲染路径抛错）').toBeTruthy();

        const series = (trend!.option as { series: Array<{ name: string; data: unknown[] }> }).series;
        const cpu = series.find((s) => s.name === 'CPU');
        expect(cpu, 'CPU 趋势 series 缺失').toBeTruthy();
        // 断言为「数字」而非字符串，且数值正确四舍五入
        expect(cpu!.data).toEqual([18.2, 33.5]);
        expect(cpu!.data.every((v) => typeof v === 'number' && Number.isFinite(v))).toBe(true);

        const net = captured.find((c) => c.height === '240' && JSON.stringify(c.option).includes('machine.receive'));
        expect(net, '网络速率趋势未生成').toBeTruthy();
        const netSeries = (net!.option as { series: Array<{ name: string; data: unknown[] }> }).series;
        // 5 分钟间隔差值 524288B -> 1.7 KB/s 量级，绝不该是 NaN
        expect(netSeries.every((s) => s.data.every((v) => typeof v === 'number' && !Number.isNaN(v)))).toBe(true);
    });
});
