import { describe, expect, it, vi } from 'vitest';
import { flushPromises, mount } from '@vue/test-utils';
import ElementPlus from 'element-plus';
import { createI18n } from 'vue-i18n';

/**
 * 提案审批面板的「处理结果」形态判定。
 *
 * 受控执行是异步的：回调当时只记「已受理」回执，终态完成后后端把真实执行/校验结果回写到
 * 工单 biz_handle_res。面板必须区分三种形态——终态结果（结构化渲染）、受理回执（等待提示，
 * 不把 queued 当结果直出）、历史纯文本（原样兜底）——否则审批人会把入队回执误读为执行结论。
 */

// MarkdownView/EnumTag 在面板里是 defineAsyncComponent：测试环境下动态 import 要走真实的
// 模块转换（非纯微任务，flushPromises 等不到），换成同步轻组件后断言只看「哪条渲染分支」
vi.mock('@/components/markdown', async () => {
    const { defineComponent, h } = await import('vue');
    return {
        MarkdownView: defineComponent({ props: { content: String }, setup: (p) => () => h('div', p.content) }),
    };
});
// __esModule 不可省：vi.mock 工厂的返回值不是真 ESM 命名空间，而 defineAsyncComponent 只按
// `__esModule || [Symbol.toStringTag]==='Module'` 判定是否解包 default。缺该标记时整个命名空间
// 对象会被当组件渲染，@vue/test-utils 读 .__isTeleport 触发 vitest「export 未定义」未捕获异常
// （用例照样全绿、进程退出码却是 1，等于门禁失真）。
vi.mock('@/components/enum-tag/EnumTag.vue', async () => {
    const { defineComponent, h } = await import('vue');
    return {
        __esModule: true,
        default: defineComponent({ props: { enums: Object, value: [String, Number] }, setup: (p) => () => h('span', String(p.value)) }),
    };
});
// 运行详情抽屉换成同步桩，且只在 visible 时渲染并把 runId 外抛：关联运行链接「点得开、
// 且开的是对的那条」只能这么测——抽真组件会拉进 store/router，抽错了又会退化成只测样式
vi.mock('@/views/agentops/RunDetailDrawer.vue', async () => {
    const { defineComponent, h } = await import('vue');
    return {
        __esModule: true,
        // props 用数组形式：runId 在抽屉未打开时为 null，声明成 Number 会触发运行时类型告警
        default: defineComponent({
            props: ['visible', 'runId'],
            setup: (p) => () => (p.visible ? h('div', { class: 'run-detail-stub' }, `RUN:${String(p.runId)}`) : null),
        }),
    };
});

const { default: AgentRemediationBiz } = await import('@/views/flow/flowbiz/agentops/AgentRemediationBiz.vue');
// 回执字面值只从组件同一镜像引用（不在测试里再抄一份，否则两边一起错还是绿）
const {
    ACK_EXECUTIONS,
    ACK_KEY_EXECUTION,
    ACK_KEY_MESSAGE,
    ACK_EXECUTION_QUEUED,
    ACK_EXECUTION_REQUEUED,
    ACK_MESSAGE_ALREADY_PROCESSED,
    ACK_MESSAGE_ALREADY_CLAIMED,
    LEGACY_ACK_MESSAGES,
} = await import('@/views/flow/flowbiz/agentops/proposalAck');

const queuedTip = '已受理：受控执行在后台异步进行';

const i18n = createI18n({
    legacy: false,
    locale: 'zh',
    messages: {
        zh: {
            flow: {
                proposalTitle: '提案标题', proposalAction: '提案动作', proposalParams: '锁定参数', proposalRisk: '风险等级',
                proposalReason: '提案理由', proposalOwner: '提案归属人', procFromRun: '来源运行', approvalNotice: '须知',
                handleResult: '处理结果', handleResultStatus: '执行状态', handleResRuns: '关联运行',
                handleResRunExec: '执行 #{id}', handleResRunVerify: '校验 #{id}', handleResErr: '失败原因',
                handleResExecResult: '执行输出', handleResVerifyConclusion: '校验结论', handleQueuedTip: queuedTip,
            },
            common: {},
            agentops: { proposalExecuted: '已执行', proposalFailed: '执行失败', riskLow: '低' },
        },
    },
});

async function mountBiz(bizHandleRes: string, bizForm?: Record<string, unknown>) {
    const wrapper = mount(AgentRemediationBiz, {
        props: {
            procinst: {
                // runId 默认带上：来源运行链接与终态回写无关（待审批工单也要能点）
                bizForm: JSON.stringify({ proposalId: 15, title: 't', action: 'MachineCommandExec', params: '{}', risk: 'low', reason: 'r', ownerUsername: 'admin', runId: 86, ...(bizForm ?? {}) }),
                bizHandleRes,
            },
        },
        global: { plugins: [ElementPlus, i18n] },
    });
    await flushPromises();
    return wrapper;
}

describe('提案审批面板的处理结果形态', () => {
    it('终态回写：结构化渲染真实执行/校验结果，不出现回执字样', async () => {
        const res = JSON.stringify({ proposalId: 15, status: 'executed', execRunId: 93, verifyRunId: 94, execResult: 'REAL-TOP-OUTPUT', verifyConclusion: 'VERIFIED-CONCLUSION' });
        const text = (await mountBiz(res)).text();
        expect(text).toContain('REAL-TOP-OUTPUT');
        expect(text).toContain('VERIFIED-CONCLUSION');
        expect(text).toContain('执行 #93');
        expect(text).toContain('校验 #94');
        expect(text).not.toContain('queued');
    });

    it('失败终态只跑到执行阶段：不得拼出一个空的「校验 #」位', async () => {
        const res = JSON.stringify({ proposalId: 14, status: 'failed', execRunId: 95, err: 'tool execution failed: Access denied' });
        const text = (await mountBiz(res)).text();
        expect(text).toContain('执行 #95');
        expect(text, 'verifyRunId 为 0 时校验链接不应渲染，旧写法会输出无意义的「校验 #-」').not.toContain('校验 #');
    });

    it('失败终态：渲染失败原因，状态标签落在提案枚举的 failed 项', async () => {
        const res = JSON.stringify({ proposalId: 14, status: 'failed', execRunId: 95, err: 'tool execution failed: Access denied' });
        const text = (await mountBiz(res)).text();
        expect(text).toContain('Access denied');
        expect(text).toContain('failed');
    });

    it('受理回执：给等待提示，不把 queued JSON 当结果直出', async () => {
        const res = JSON.stringify({ proposalId: 15, status: 'approved', [ACK_KEY_EXECUTION]: ACK_EXECUTION_QUEUED });
        const text = (await mountBiz(res)).text();
        expect(text).toContain(queuedTip);
        expect(text).not.toContain(ACK_EXECUTION_QUEUED);
    });

    it('崩溃重投/幂等落败回执同样降级为等待提示，不裸直出 JSON', async () => {
        // 后端 FlowBizHandle 的非终态返回共四种形态，漏一种就会把无结论回执当「处理结果」展示
        for (const ack of [
            { proposalId: 1, status: 'approved', [ACK_KEY_EXECUTION]: ACK_EXECUTION_REQUEUED },
            { proposalId: 1, status: 'approved', [ACK_KEY_MESSAGE]: ACK_MESSAGE_ALREADY_PROCESSED },
            { proposalId: 1, status: 'approved', [ACK_KEY_MESSAGE]: ACK_MESSAGE_ALREADY_CLAIMED },
        ]) {
            const text = (await mountBiz(JSON.stringify(ack))).text();
            expect(text).toContain(queuedTip);
            expect(text).not.toContain(ACK_EXECUTION_REQUEUED);
            expect(text).not.toContain(ACK_MESSAGE_ALREADY_PROCESSED);
            expect(text).not.toContain(ACK_MESSAGE_ALREADY_CLAIMED);
        }
    });

    it('历史句子形态的回执（存量工单）仍能识别，不退回裸 JSON', async () => {
        // 回执值曾以英文句子形态落库，改判为标识符后存量工单必须继续识别（见 proposalAck.ts）
        for (const legacy of LEGACY_ACK_MESSAGES) {
            const text = (await mountBiz(JSON.stringify({ proposalId: 1, status: 'approved', [ACK_KEY_MESSAGE]: legacy }))).text();
            expect(text).toContain(queuedTip);
            expect(text).not.toContain(legacy);
        }
    });

    it('回执字面值与后端线协议钉对（改值必须同步后端 proposal_ack.go）', () => {
        // 后端 `application/proposal_ack_test.go` 钉的是同一组值：任一侧改动都会红，
        // 而红之前存量工单的面板已经会静默退化成裸 JSON——所以不能只测组件行为
        expect(ACK_KEY_EXECUTION).toBe('execution');
        expect(ACK_KEY_MESSAGE).toBe('message');
        expect([ACK_EXECUTION_QUEUED, ACK_EXECUTION_REQUEUED]).toEqual(['queued', 'requeued']);
        expect([ACK_MESSAGE_ALREADY_PROCESSED, ACK_MESSAGE_ALREADY_CLAIMED]).toEqual(['already_processed', 'already_claimed']);
        // 机器可读标识符不得含空格，否则形似待翻译文案（后端同一条断言）
        for (const v of [...ACK_EXECUTIONS, ACK_MESSAGE_ALREADY_PROCESSED, ACK_MESSAGE_ALREADY_CLAIMED]) {
            expect(v).not.toContain(' ');
        }
        // 历史清单只读不改新值：后端不会再产出句子形态
        expect(LEGACY_ACK_MESSAGES).toEqual(['already processed', 'already claimed']);
    });

    it('历史纯文本结果：原样兜底展示', async () => {
        const text = (await mountBiz('success')).text();
        expect(text).toContain('success');
    });

    it('无处理结果（未审批/未执行）：整个处理结果区不渲染', async () => {
        const text = (await mountBiz('')).text();
        expect(text).not.toContain('处理结果');
    });
});

describe('审批面板里的关联运行必须点得开', () => {
    const finalRes = JSON.stringify({
        proposalId: 15,
        status: 'executed',
        execRunId: 93,
        verifyRunId: 94,
        execResult: 'X',
        verifyConclusion: 'Y',
    });

    /** [链接文案, 应打开的运行 id] */
    const cases: Array<[string, number]> = [
        ['86', 86], // 来源运行（取自工单 bizForm 的提案快照）
        ['执行 #93', 93],
        ['校验 #94', 94],
    ];

    it.each(cases)('点「%s」打开的是运行 %s 的详情', async (label, runId) => {
        const wrapper = await mountBiz(finalRes);

        // 未点开之前必须没有抽屉内容，否则「点开了」这条断言恒真（抽屉常驻可见时测不出接线）
        expect(wrapper.find('.run-detail-stub').exists(), '未点击就不该已渲染运行详情').toBe(false);

        const link = wrapper.findAll('.el-link').find((l) => l.text().trim() === label);
        expect(link, `未找到文案为「${label}」的链接`).toBeDefined();
        await link!.trigger('click');
        expect(wrapper.find('.run-detail-stub').text(), `「${label}」应接到运行 ${runId}`).toBe(`RUN:${runId}`);
    });

    it('待审批工单（无终态回写）也必须点得开来源运行', async () => {
        // 「处理结果」整块在 finalRes 为空时不渲染：抽屉若挂在那一块里，审批人正好在最需要
        // 看来源轨迹的时候点不开（而错的方式不会报错，只是点了没反应）
        const wrapper = await mountBiz('');
        const source = wrapper.findAll('.el-link').find((l) => l.text().trim() === '86');
        expect(source, '待审批工单里没渲染来源运行链接').toBeDefined();
        await source!.trigger('click');
        expect(wrapper.find('.run-detail-stub').text()).toBe('RUN:86');
    });

    it('bizForm 没带 runId 时不得渲染出可点的空链接', async () => {
        const wrapper = await mountBiz(finalRes, { runId: undefined });
        expect(wrapper.findAll('.el-link').map((l) => l.text().trim())).toEqual(['执行 #93', '校验 #94']);
    });
});
