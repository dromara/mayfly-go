/**
 * 中断卡片徽章的渲染契约（H4）
 *
 * 为什么单独测：三个中断组件此前用 `v-if="!readonly"` 当「待决策」、`v-else` 直接显示
 * 「已处理」。readonly 表示「当前在只读轨迹里」，与这条中断有没有被批准是两件正交的事——
 * 于是运维轨迹里一条**从未被批准**的 MachineCommandExec 审批显示成「已处理」，
 * 审计完全读不出这里曾经卡住。
 *
 * 真机拿不到该夹具：现存 75 条运维运行全是只读诊断轨迹、不含中断；库里唯一一条含
 * interrupt 的 tool_call item 其 extra 为 NULL（前端只从 extra.interrupt 恢复卡片）。
 * 造该数据要写三张表并污染真实会话，因此改为挂载真实组件断言渲染出的文案。
 */
import { beforeEach, describe, expect, it } from 'vitest';
import { mount } from '@vue/test-utils';
import { i18n } from '@/i18n';
import type { InterruptEvent } from '../../protocol/types';
import ApprovalInterrupt from '../ApprovalInterrupt.vue';
import ConfirmationInterrupt from '../ConfirmationInterrupt.vue';
import ParamCompletionInterrupt from '../param-completion/index.vue';
import GenericInterrupt from '../GenericInterrupt.vue';

const t = (key: string) => i18n.global.t(key);

/** RejectReasonPopover 依赖 element-plus 弹层，在 happy-dom 下无需真实渲染 */
const globalMount = {
    plugins: [i18n],
    stubs: { RejectReasonPopover: true },
};

const makeInterrupt = (status: string): InterruptEvent => ({
    actionId: 'act-1',
    type: 'approval',
    description: '执行命令可能涉及高危操作，请审批',
    toolName: 'MachineCommandExec',
    toolCallId: 'call-1',
    turnId: 'turn-1',
    status,
    metadata: {},
});

const mountInterrupt = (component: object, status: string, readonly: boolean) =>
    mount(component, {
        props: { interrupt: makeInterrupt(status), turnId: 'turn-1', readonly, onAction: () => undefined },
        global: globalMount,
    });

describe('审批中断卡片：决策状态来自 status，不由 readonly 推断', () => {
    beforeEach(() => {
        i18n.global.locale.value = 'zh-cn';
    });

    it('只读轨迹里的未决策审批显示「待审批」，绝不显示「已处理」', () => {
        const wrapper = mountInterrupt(ApprovalInterrupt, 'pending', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.interrupt.approval.pendingApproval'));
        expect(text).not.toContain('已解决');
    });

    it('只读轨迹里的已批准审批显示具体决议（「已批准」），不显示待审批', () => {
        const wrapper = mountInterrupt(ApprovalInterrupt, 'approved', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.chat.resumeStatus.approved'));
        expect(text).not.toContain(t('ai.interrupt.approval.pendingApproval'));
    });

    it('只读轨迹里的已拒绝审批给拒绝语义（不得与批准同色同文）', () => {
        const wrapper = mountInterrupt(ApprovalInterrupt, 'rejected', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.chat.resumeStatus.rejected'));
        expect(text).not.toContain(t('ai.chat.resumeStatus.approved'));
    });

    it('只读态不给写入口（审批按钮不得渲染）', () => {
        const wrapper = mountInterrupt(ApprovalInterrupt, 'pending', true);
        expect(wrapper.text()).not.toContain(t('ai.interrupt.approval.approve'));
    });

    it('可交互态的未决策审批既有提示也有审批按钮', () => {
        const wrapper = mountInterrupt(ApprovalInterrupt, 'pending', false);
        const text = wrapper.text();
        expect(text).toContain(t('ai.interrupt.approval.pendingApproval'));
        expect(text).toContain(t('ai.interrupt.approval.approve'));
    });
});

describe('确认与参数补全卡片同一口径', () => {
    it('确认类中断在只读态未决策时显示「待确认」而非「已处理」', () => {
        const wrapper = mountInterrupt(ConfirmationInterrupt, 'pending', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.interrupt.confirmation.pendingConfirmation'));
        expect(text).not.toContain('已解决');
    });

    it('参数补全在只读态未补全时显示「待补全」而非「已完成」', () => {
        const wrapper = mountInterrupt(ParamCompletionInterrupt, 'pending', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.interrupt.paramCompletion.pending'));
        expect(text).not.toContain('已补全');
    });

    it('参数补全已完善时走决议徽章（paramsCompleted 专属文案）', () => {
        const wrapper = mountInterrupt(ParamCompletionInterrupt, 'params_completed', true);
        expect(wrapper.text()).toContain(t('ai.chat.resumeStatus.paramsCompleted'));
    });
});


describe('通用中断卡片（同一缺陷的第四个命中点，审查报告未列出）', () => {
    it('只读态未决策显示「待处理」而非「已解决」', () => {
        const wrapper = mountInterrupt(GenericInterrupt, 'pending', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.interrupt.generic.pending'));
        expect(text).not.toContain('已解决');
    });

    it('只读态已批准走决议徽章，且不给写入口', () => {
        const wrapper = mountInterrupt(GenericInterrupt, 'approved', true);
        const text = wrapper.text();
        expect(text).toContain(t('ai.chat.resumeStatus.approved'));
        expect(text).not.toContain(t('ai.interrupt.generic.confirm'));
    });
});
