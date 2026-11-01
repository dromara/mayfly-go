/**
 * 数据库诊断面板的三个页签（活动会话 / 慢查询 / 锁等待）。
 *
 * 为什么要有这组用例：诊断面板只能从资源树的库节点菜单打开，而该菜单在无头浏览器里
 * 渲染为空（同一次运行里「表」节点的菜单正常），所以三页签的结构与取数分支必须靠
 * 组件级用例钉住——否则「慢查询报不可用时会不会退化成空表」这种分支没人守。
 *
 * 钉住四件事：
 * 1. 三个页签都在，默认停在活动会话；
 * 2. 切到慢查询才取数（不切不取，避免打开面板就打聚合表）；
 * 3. 慢查询「不可用」时把服务端原因显示出来，而不是给一张空表；
 * 4. 锁等待渲染「等待方 + 阻塞方」，不是只给一个计数。
 */
import { describe, expect, it, vi, beforeEach } from 'vitest';
import { mount, flushPromises } from '@vue/test-utils';
import { createI18n } from 'vue-i18n';
import ElementPlus from 'element-plus';
import DbSessionPanel from '../DbSessionPanel.vue';
import { dbApi } from '../../api';

const sessions = vi.fn();
const slowQueries = vi.fn();
const lockWaits = vi.fn();
const metrics = vi.fn();
const recovery = vi.fn();
const accounts = vi.fn();

vi.mock('../../api', () => ({
    dbApi: {
        dbSessions: { request: (...a: unknown[]) => sessions(...a) },
        dbSlowQueries: { request: (...a: unknown[]) => slowQueries(...a) },
        dbLockWaits: { request: (...a: unknown[]) => lockWaits(...a) },
        dbMetrics: { request: (...a: unknown[]) => metrics(...a) },
        dbRecoveryReadiness: { request: (...a: unknown[]) => recovery(...a) },
        dbAccounts: { request: (...a: unknown[]) => accounts(...a) },
        cancelDbSession: { request: vi.fn() },
    },
}));

vi.mock('@/hooks/useI18n', () => ({
    Msg: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
    useI18nConfirm: vi.fn(async () => false),
}));
vi.mock('@vueuse/core', () => ({ useIntervalFn: () => ({ pause: vi.fn(), resume: vi.fn() }) }));
vi.mock('@/common/utils/format', () => ({ formatTime: (v: number) => `${v}s` }));

const i18n = createI18n({
    legacy: false,
    globalInjection: true,
    locale: 'zh-cn',
    missingWarn: false,
    fallbackWarn: false,
    messages: { 'zh-cn': {} },
});

const authDirective = { mounted: () => undefined, updated: () => undefined };

function mountPanel() {
    return mount(DbSessionPanel, {
        props: { dbId: 1, dbName: 'mayfly_go' },
        global: {
            plugins: [i18n, ElementPlus],
            directives: { auth: authDirective },
            // ECharts 在 jsdom 里没有 canvas：用同名桩替身，只断言传进去的 option
            stubs: { ECharts: { name: 'ECharts', props: ['option'], template: '<div class="echarts-stub" />' } },
        },
    });
}

beforeEach(() => {
    sessions.mockReset().mockResolvedValue([]);
    slowQueries.mockReset().mockResolvedValue([]);
    lockWaits.mockReset().mockResolvedValue([]);
    metrics.mockReset().mockResolvedValue({ metric: 'sessions_active', points: [] });
    recovery.mockReset().mockResolvedValue({ prepared: false, mode: 'none', reason: 'binlog 未开启' });
    accounts.mockReset().mockResolvedValue({ accounts: [], risks: [] });
});

describe('数据库诊断面板', () => {
    it('三个页签齐全，默认只取会话', async () => {
        const wrapper = mountPanel();
        await flushPromises();

        const labels = wrapper.findAll('.el-tabs__item').map((n) => n.text());
        expect(labels).toEqual([
            'db.sessionTabSessions',
            'db.sessionTabSlow',
            'db.sessionTabLocks',
            'db.sessionTabMetrics',
            'db.sessionTabRecovery',
            'db.sessionTabAccounts',
        ]);
        expect(sessions).toHaveBeenCalledTimes(1);
        // 不切页签不取聚合表：打开面板就打 performance_schema 的代价太高
        expect(slowQueries).not.toHaveBeenCalled();
        expect(lockWaits).not.toHaveBeenCalled();
        // 趋势页签同理：不切不取（采样表按实例遍历，代价不为零）
        expect(metrics).not.toHaveBeenCalled();
        expect(recovery).not.toHaveBeenCalled();
        expect(accounts).not.toHaveBeenCalled();
    });

    it('切到慢查询才取数，并渲染语句与耗时', async () => {
        slowQueries.mockResolvedValue([
            { digest: 'd1', sql: 'SELECT * FROM t_ai_skill', schema: 'mayfly_go', calls: 12, avgMs: 8000, maxMs: 20000, totalMs: 96000, rowsAvg: 3 },
        ]);
        const wrapper = mountPanel();
        await flushPromises();

        await wrapper.findAll('.el-tabs__item')[1].trigger('click');
        await flushPromises();

        expect(slowQueries).toHaveBeenCalledTimes(1);
        expect(wrapper.text()).toContain('SELECT * FROM t_ai_skill');
        expect(wrapper.text()).toContain('96s');
        // 毫秒级耗时必须显示成 ms：按秒取整会把 8000ms 之外的短耗时压成 "-"
        expect(wrapper.text()).toContain('8.0s');
        expect(wrapper.text()).toContain('20.0s');
    });

    it('慢查询不可用时显示服务端原因，而不是一张空表', async () => {
        slowQueries.mockRejectedValue(new Error('该实例无法提供慢查询统计：performance_schema 未开启'));
        const wrapper = mountPanel();
        await flushPromises();

        await wrapper.findAll('.el-tabs__item')[1].trigger('click');
        await flushPromises();

        const alert = wrapper.find('.el-alert');
        expect(alert.exists()).toBe(true);
        expect(alert.text()).toContain('performance_schema 未开启');
    });

    it('锁等待渲染等待方与阻塞方', async () => {
        lockWaits.mockResolvedValue([
            {
                waitSessionId: '6241',
                waitUser: 'root',
                waitQuery: 'UPDATE t SET a=1',
                waitMs: 9000,
                blockSessionId: '6240',
                blockUser: 'root',
                blockQuery: 'SELECT * FROM t FOR UPDATE',
                lockType: 'mayfly_go.t PRIMARY',
            },
        ]);
        const wrapper = mountPanel();
        await flushPromises();

        await wrapper.findAll('.el-tabs__item')[2].trigger('click');
        await flushPromises();

        expect(lockWaits).toHaveBeenCalledTimes(1);
        const text = wrapper.text();
        expect(text).toContain('6241');
        expect(text).toContain('6240');
        expect(text).toContain('mayfly_go.t PRIMARY');
        // 阻塞源是处置对象：按钮必须落在这一行
        expect(wrapper.find('.el-button').exists()).toBe(true);
    });

    it('趋势页签按选中的指标与时间窗取数，并把点交给图表', async () => {
        metrics.mockResolvedValue({
            metric: 'sessions_active',
            points: [
                { ts: 1729700000000, value: 3 },
                { ts: 1729700300000, value: 5 },
            ],
        });
        const wrapper = mountPanel();
        await flushPromises();

        await wrapper.findAll('.el-tabs__item')[3].trigger('click');
        await flushPromises();

        expect(metrics).toHaveBeenCalledTimes(1);
        expect(metrics).toHaveBeenCalledWith({ id: 1, metric: 'sessions_active', minutes: 60 });
        // 有数据点时渲染图表组件，并把 [[ts,value]] 交给它
        const chart = wrapper.findComponent({ name: 'ECharts' });
        expect(chart.exists()).toBe(true);
        const option = chart.props('option') as { series: { data: number[][] }[] };
        expect(option.series[0].data).toEqual([
            [1729700000000, 3],
            [1729700300000, 5],
        ]);
    });

    it('趋势为空时给出空态而不是空白画布', async () => {
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[3].trigger('click');
        await flushPromises();
        expect(wrapper.findComponent({ name: 'ECharts' }).exists()).toBe(false);
        expect(wrapper.text()).toContain('db.metricEmpty');
    });

    it('恢复能力页签：不可用时把原因显示出来，而不是只显示「不可用」', async () => {
        recovery.mockResolvedValue({
            prepared: false,
            mode: 'none',
            reason: 'binlog 未开启（log_bin=OFF），平台只能恢复到逻辑备份时刻',
        });
        const wrapper = mountPanel();
        await flushPromises();

        await wrapper.findAll('.el-tabs__item')[4].trigger('click');
        await flushPromises();

        expect(recovery).toHaveBeenCalledWith({ id: 1, db: 'mayfly_go' });
        const text = wrapper.text();
        expect(text).toContain('db.recoveryNotPrepared');
        // 原因必须原样呈现：这是运维要不要改部署形态的唯一依据
        expect(text).toContain('log_bin=OFF');
        expect(text).toContain('db.recoveryMode_none');
    });

    it('恢复能力就绪时展示位点与保留期', async () => {
        recovery.mockResolvedValue({
            prepared: true,
            mode: 'binlog',
            oldestPoint: 'mysql-bin.000003',
            latestPoint: 'mysql-bin.000011:1234',
            retention: '24 小时',
        });
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[4].trigger('click');
        await flushPromises();

        const text = wrapper.text();
        expect(text).toContain('db.recoveryPrepared');
        expect(text).toContain('mysql-bin.000003');
        expect(text).toContain('mysql-bin.000011:1234');
        expect(text).toContain('24 小时');
    });

    it('账号页签：列出账号并区分「必须处置」与「建议处置」', async () => {
        accounts.mockResolvedValue({
            accounts: [
                { name: 'root', host: '%', canLogin: true, locked: false, passwordExpired: false, superuser: true, canGrant: true, neverExpire: false },
                { name: 'app', host: '10.0.0.%', canLogin: true, locked: false, passwordExpired: true, superuser: false, canGrant: false, neverExpire: false },
                { name: 'grp', canLogin: false, locked: false, passwordExpired: false, superuser: false, canGrant: false, neverExpire: true },
            ],
            risks: [
                { level: 'high', code: 'superuser', account: 'root', host: '%' },
                { level: 'high', code: 'password_expired', account: 'app' },
                { level: 'warn', code: 'any_host', account: 'root', host: '%' },
            ],
        });
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[5].trigger('click');
        await flushPromises();

        expect(accounts).toHaveBeenCalledWith({ id: 1, db: 'mayfly_go' });
        const text = wrapper.text();
        expect(text).toContain('db.accountTotal');
        // 2 个必须处置 / 1 个建议处置
        expect(text).toContain('db.accountRiskHighCount');
        expect(text).toContain('db.accountRiskWarnCount');
        // 账号行与标记都要出现
        expect(text).toContain('root');
        expect(text).toContain('db.accountPwdExpired');
        expect(text).toContain('db.accountNoLogin');
    });

    it('账号读不到时显示原因，而不是空表', async () => {
        accounts.mockRejectedValue(new Error('该实例无法盘点账号：SELECT command denied to user'));
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[5].trigger('click');
        await flushPromises();

        expect(wrapper.text()).toContain('SELECT command denied');
        expect(wrapper.find('.el-alert').exists()).toBe(true);
    });

    it('账号按风险排序：高危在前，无风险最后（列表默认顺序会让高危埋在中间）', async () => {
        accounts.mockResolvedValue({
            accounts: [
                { name: 'plain_a', host: '10.0.0.1', canLogin: true, locked: false, passwordExpired: false, superuser: false, canGrant: false, neverExpire: false },
                { name: 'root_like', host: '%', canLogin: true, locked: false, passwordExpired: false, superuser: true, canGrant: true, neverExpire: false },
                { name: 'expired_one', host: '10.0.0.2', canLogin: true, locked: false, passwordExpired: true, superuser: false, canGrant: false, neverExpire: false },
            ],
            risks: [
                { level: 'high', code: 'superuser', account: 'root_like' },
                { level: 'high', code: 'password_expired', account: 'expired_one' },
            ],
        });
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[5].trigger('click');
        await flushPromises();

        // jsdom 下 :visible 不可靠（offsetParent 恒 null），按页签索引取 pane
        const accountPane = wrapper.findAll('.el-tab-pane')[5];
        const names = accountPane.findAll('.el-table__body tr').map((tr) => tr.findAll('td')[0].text());
        expect(names).toEqual(['expired_one', 'root_like', 'plain_a']);
    });

    it('慢查询行可复制 SQL（诊断后的下一步动作）', async () => {
        slowQueries.mockResolvedValue([
            { digest: 'd1', sql: 'SELECT * FROM t_slow', schema: 'mayfly_go', calls: 1, avgMs: 10, maxMs: 10, totalMs: 10, rowsAvg: 1 },
        ]);
        const wrapper = mountPanel();
        await flushPromises();
        await wrapper.findAll('.el-tabs__item')[1].trigger('click');
        await flushPromises();

        const slowPane = wrapper.findAll('.el-tab-pane')[1];
        const copyLink = slowPane.findAll('.el-link').find((link) => link.text() === 'common.copy');
        expect(copyLink).toBeTruthy();
    });
});



