/**
 * 磁盘分析占比条的口径测试。
 *
 * 字节数走 `JSONBig({ storeAsString: true })` 后可能是**字符串**，而字符串的 `>` 是字典序比较：
 * `"45050000" > "310150000"` 为 true（位数不同时字典序先比首位），
 * 于是「以最大目录为基准」的基准值取错，最大目录的占比条会算出 689% 这种荒谬值。
 * 本测试用位数不同的字符串尺寸喂入，钉住占比条必须按数值口径。
 */
import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';
import { formatByteSize } from '@/common/utils/format';

// 服务端按 size 倒序返回；这里刻意用「字符串 + 位数不同」复现真实回传形态
const NODES = [
    { path: '/root', size: '310150000' },
    { path: '/root/mayfly-go-linux-amd64', size: '45050000' },
    { path: '/root/bin', size: '31410000' },
];

vi.mock('@/views/ops/machine/api', () => ({
    diskApi: { analyze: { request: vi.fn().mockResolvedValue({ nodes: NODES }) } },
}));
vi.mock('@/common/request', () => ({ default: { request: vi.fn().mockResolvedValue({}) } }));
vi.mock('@/i18n', () => ({ i18n: { global: { t: (k: string) => k } } }));
vi.mock('@/components/svg-icon/index.vue', () => ({ default: { template: '<i />' } }));

const i18n = createI18n({ legacy: false, locale: 'en', messages: { en: {} } });

describe('DiskAnalyzeDialog 占比口径', () => {
    it('占比条以数值最大目录为基准（100%），且不存在超过 100% 的行', async () => {
        const DiskAnalyzeDialog = (await import('../DiskAnalyzeDialog.vue')).default;
        const wrapper = mount(DiskAnalyzeDialog, {
            props: { visible: true, machineId: 1, machineName: 'm1' },
            global: { plugins: [ElementPlus, i18n] },
            attachTo: document.body,
        });
        await flushPromises();
        await new Promise(r => setTimeout(r, 100));

        // 打开时不自动跑采集（避免点开就占一次 SSH），必须显式触发分析
        const goBtn = [...document.querySelectorAll('button')].find(b => (b.textContent || '').includes('machine.diskAnalyze'));
        expect(goBtn, '未找到分析按钮').toBeTruthy();
        goBtn!.click();
        await flushPromises();
        await new Promise(r => setTimeout(r, 100));

        // el-dialog 会 teleport 到 body，wrapper.findAll 看不到，需直接查 document
        const bars = [...document.querySelectorAll('.el-progress-bar__inner')].map(el => parseFloat((el.getAttribute('style') || '').match(/width:\s*([\d.]+)%/)?.[1] ?? 'NaN'));
        expect(bars.length).toBe(NODES.length);
        // 最大目录（/root）应为基准 100%
        expect(bars[0]).toBeCloseTo(100, 1);
        // 修复前此处会出现 689% 这类值
        for (const b of bars) {
            expect(b).toBeLessThanOrEqual(100.0001);
        }
        // 次大项 Math.round(45050000 / 310150000 * 100) = 15
        expect(bars[1]).toBe(15);
        expect(bars[2]).toBe(10);

        // 大小列对字符串字节数也能正常格式化（不能出 NaN）
        const body = document.body.textContent || '';
        expect(body).toContain(formatByteSize(NODES[0].size as unknown as number));
        expect(body).not.toContain('NaN');
        wrapper.unmount();
    });
});
