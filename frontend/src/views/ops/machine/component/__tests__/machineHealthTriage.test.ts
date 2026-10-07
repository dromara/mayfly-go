/**
 * 健康总览「分诊台」语义测试。
 *
 * 这个视图存在的唯一理由是「异常优先」，因此必须钉住三件事：
 * 1. 排序：离线 → 越线(按规则优先级) → 未判定 → 正常；
 * 2. **未判定不得显示成正常**（取不到当前值 ≠ 健康，这是 fail-closed 的最后一环）；
 * 3. 越线指标要有颜色区分，且「仅看异常」能收敛掉正常行。
 */
import { afterEach, describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

// i18n 用空字典：t() 原样返回 key，断言直接针对键名，改文案不会误伤判据
const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

const ROWS = [
    {
        machineId: 4,
        name: 'm-normal',
        ip: '10.0.0.4',
        port: 22,
        code: 'c4',
        status: 1,
        cpuUsage: 5,
        memUsage: 10,
        diskUsage: 12,
        collectTime: '2026-10-06T10:00:00+08:00',
        priority: -1,
        triaged: true,
        hits: null,
    },
    {
        machineId: 3,
        name: 'm-unknown',
        ip: '10.0.0.3',
        port: 22,
        code: 'c3',
        status: 1,
        cpuUsage: 0,
        memUsage: 0,
        diskUsage: 0,
        collectTime: null,
        priority: -1,
        triaged: false, // 有规则覆盖但取不到当前值
        hits: null,
    },
    {
        machineId: 2,
        name: 'm-breached',
        ip: '10.0.0.2',
        port: 22,
        code: 'c2',
        status: 1,
        cpuUsage: 8,
        memUsage: 60,
        diskUsage: 96.4,
        collectTime: '2026-10-06T10:00:00+08:00',
        priority: 0,
        triaged: true,
        hits: [{ ruleId: 11, ruleName: '磁盘紧急', metric: 'disk_usage', compare: 'gt', threshold: 85, current: 96.4, priority: 0 }],
    },
    {
        machineId: 1,
        name: 'm-offline',
        ip: '10.0.0.1',
        port: 22,
        code: 'c1',
        status: 0,
        cpuUsage: 0,
        memUsage: 0,
        diskUsage: 0,
        collectTime: '2026-10-06T09:00:00+08:00',
        priority: -1,
        triaged: true,
        hits: null,
    },
];

vi.mock('@/views/ops/machine/api', () => ({
    metricApi: { healthOverview: { request: vi.fn().mockResolvedValue(ROWS) }, range: { request: vi.fn().mockResolvedValue([]) } },
}));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue({}) } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }));
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { template: '<i />' } }));

async function mountDialog() {
    const MachineHealthDialog = (await import('../MachineHealthDialog.vue')).default;
    const wrapper = mount(MachineHealthDialog, {
        // 先以关闭态挂载再翻开：总览数据是在 visible 由 false→true 时拉取的，
        // 直接以 visible=true 挂载没有跳变，会测不到取数链路
        props: { visible: false },
        global: { plugins: [ElementPlus, i18n] },
        attachTo: document.body,
    });
    await wrapper.setProps({ visible: true });
    await flushPromises();
    await new Promise((r) => setTimeout(r, 150));
    return wrapper;
}

// el-dialog 会 teleport 到 body，断言统一查 document；用例间必须清空，否则上一个弹层的表格行会污染计数
const bodyRows = () => [...document.querySelectorAll('.el-table__body tr')];
const rowText = (idx: number) => (bodyRows()[idx]?.textContent || '').replace(/\s+/g, ' ');

afterEach(() => {
    document.body.innerHTML = '';
});

describe('MachineHealthDialog 分诊语义', () => {
    it('按异常优先排序：离线 → 越线 → 未判定 → 正常', async () => {
        await mountDialog();
        expect(bodyRows().length).toBe(4);
        expect(rowText(0)).toContain('m-offline');
        expect(rowText(1)).toContain('m-breached');
        expect(rowText(2)).toContain('m-unknown');
        expect(rowText(3)).toContain('m-normal');
    });

    // 本条是整个设计的安全底线：取不到值时不能给出「正常」的假结论
    it('未判定的机器显示「未判定」，绝不显示成正常', async () => {
        await mountDialog();
        const unknownRow = rowText(2);
        expect(unknownRow).toContain('machine.healthNotTriaged');
        expect(unknownRow).not.toContain('machine.healthNormal');
        // 只允许那台真正常的机器拿到「正常」标签
        const normals = bodyRows().filter((tr) => (tr.textContent || '').includes('machine.healthNormal'));
        expect(normals.length).toBe(1);
    });

    it('摘要按类别给计数，全正常时才显示「全部正常」', async () => {
        await mountDialog();
        const summary = document.body.textContent || '';
        // 1 离线 / 1 越线 / 1 未判定
        expect(summary).toContain('machine.healthSummaryOffline');
        expect(summary).toContain('machine.healthSummaryAbnormal');
        expect(summary).toContain('machine.healthSummaryUnknown');
        expect(summary).not.toContain('machine.healthAllOk');
    });

    it('越线指标进度条标红，正常机器不标红', async () => {
        await mountDialog();
        const breached = bodyRows()[1];
        const normal = bodyRows()[3];
        expect(breached.querySelector('.el-progress.is-exception')).toBeTruthy();
        expect(normal.querySelector('.el-progress.is-exception')).toBeFalsy();
    });

    it('「仅看异常」过滤掉正常行，异常与未判定保留', async () => {
        await mountDialog();
        const checkbox = [...document.querySelectorAll('.el-checkbox')].find((c) => (c.textContent || '').includes('machine.healthOnlyAbnormal'));
        expect(checkbox).toBeTruthy();
        (checkbox as HTMLElement).click();
        await flushPromises();
        await new Promise((r) => setTimeout(r, 100));
        const left = bodyRows().map((tr) => tr.textContent || '');
        expect(left.length).toBe(3);
        expect(left.join('|')).not.toContain('m-normal');
        expect(left.join('|')).toContain('m-unknown');
    });

    it('点击「查看趋势」把该行机器抛给父级下钻', async () => {
        const wrapper = await mountDialog();
        const btn = [...bodyRows()[1].querySelectorAll('button')].find((b) => (b.textContent || '').includes('machine.healthViewTrend'));
        expect(btn).toBeTruthy();
        (btn as HTMLElement).click();
        await flushPromises();
        const emitted = wrapper.emitted('view-trend');
        expect(emitted).toBeTruthy();
        const payload = emitted![0][0] as { machineId: number; name: string };
        expect(payload.machineId).toBe(2);
        expect(payload.name).toBe('m-breached');
    });
});
