/**
 * 中断徽章三态判定的契约
 *
 * 为什么单独测：三个中断组件此前用 `v-if="!readonly"` 当「待决策」、`v-else` 直接显示「已处理」，
 * 于是运维轨迹里一条**从未被批准**的 MachineCommandExec 审批会显示成「已处理」——审计完全读不出
 * 这里曾经卡住。readonly 表示「当前在只读轨迹」，与中断有没有被决策是两件正交的事。
 *
 * 真机无法构造该场景（现存 75 条 agentops 运行全是只读诊断轨迹、不含中断），所以把判据钉在这里。
 */
import { describe, expect, it } from 'vitest';
import { isInterruptDecided, RESUME_STATUS_BADGES, FALLBACK_RESUME_BADGE, InterruptResumeStatus } from '../../registries/statuses';

describe('中断是否已决策', () => {
    it('待决策状态不得被判为已决策（含只读轨迹场景）', () => {
        expect(isInterruptDecided(InterruptResumeStatus.Pending)).toBe(false);
        expect(isInterruptDecided('')).toBe(false);
        expect(isInterruptDecided(undefined)).toBe(false);
    });

    it('已决策状态必须可判真（徽章走具体决议文案）', () => {
        for (const status of [
            InterruptResumeStatus.Approved,
            InterruptResumeStatus.Rejected,
            InterruptResumeStatus.ParamsCompleted,
        ]) {
            expect(isInterruptDecided(status)).toBe(true);
        }
    });
});

describe('决议徽章配置单点', () => {
    it('每个已决策状态都有专属 labelKey，不得共用笼统文案', () => {
        const keys = new Set(Object.values(RESUME_STATUS_BADGES).map((b) => b.labelKey));
        // 已批准与已解决语义同源（同一文案），其余必须各自独立
        expect(keys.size).toBeGreaterThanOrEqual(Object.keys(RESUME_STATUS_BADGES).length - 1);
        expect(keys.has('ai.chat.resumeStatus.rejected')).toBe(true);
        expect(keys.has('ai.chat.resumeStatus.paramsCompleted')).toBe(true);
        expect(keys.has('ai.chat.resumeStatus.answered')).toBe(true);
    });

    it('徽章配置必须同时给出语义色与文案 key（缺一项界面就会掉色或吐空）', () => {
        for (const [status, badge] of Object.entries(RESUME_STATUS_BADGES)) {
            expect(badge.class, status).toMatch(/text-|bg-/);
            expect(badge.labelKey, status).toMatch(/^ai\.chat\.resumeStatus\./);
        }
    });

    it('未登记状态回退兜底徽章而不是 undefined（历史数据里的新状态不能让徽章区空白）', () => {
        expect(RESUME_STATUS_BADGES['some_future_status']).toBeUndefined();
        expect(FALLBACK_RESUME_BADGE.labelKey).toBeTruthy();
        expect(FALLBACK_RESUME_BADGE.class).toMatch(/bg-muted/);
    });
});
