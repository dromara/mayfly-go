import { describe, expect, it } from 'vitest';
import { buildRunForkParams } from '../forkDraft';
import type { AgentRun } from '../types';

/** 一条字段齐备的运行（详情接口返回形态：含 instruction/input/err 等大字段） */
function run(overrides: Partial<AgentRun> = {}): AgentRun {
    return {
        id: 1,
        runNo: 'rn-1',
        title: 'DB 慢查询巡检',
        triggerType: 'cron',
        triggerRef: 'task-key',
        profileId: 'diagnostician',
        status: 'succeeded',
        ownerAccountId: 7,
        ownerUsername: 'ops',
        conversationId: 88,
        turnId: 't1',
        summary: '未发现慢查询',
        promptTokens: 1,
        completionTokens: 2,
        totalTokens: 3,
        costMs: 100,
        err: '',
        instruction: '巡检所有库',
        input: '目标资源：{"dbId":1}',
        ...overrides,
    };
}

const labels = { status: '成功', profile: '只读诊断', trigger: '定时' };

describe('buildRunForkParams（带上下文追问的草稿取值）', () => {
    it('映射运行事实，本地化标签由调用方传入并原样使用', () => {
        const params = buildRunForkParams(run(), labels);

        expect(params.runNo).toBe('rn-1');
        expect(params.title).toBe('DB 慢查询巡检');
        expect(params.owner).toBe('ops');
        // 组件不做文案拼接：状态/档位/触发方式用的是传入的标签
        expect(params.status).toBe('成功');
        expect(params.profile).toBe('只读诊断');
        expect(params.trigger).toBe('定时');
        expect(params.instruction).toBe('巡检所有库');
        expect(params.input).toBe('目标资源：{"dbId":1}');
    });

    it('缺失字段回落占位符，不留空行（草稿是要给人读并直接发送的）', () => {
        const params = buildRunForkParams(run({ instruction: undefined, input: undefined, ownerUsername: '' }), labels);

        expect(params.instruction).toBe('—');
        expect(params.input).toBe('—');
        expect(params.owner).toBe('—');
    });

    it('超长字段截断并标注原始长度（用户能判断还有多少没贴进草稿）', () => {
        const long = 'x'.repeat(2500);
        const params = buildRunForkParams(run({ summary: long }), labels);

        expect(params.summary.length).toBeLessThan(long.length);
        expect(params.summary).toContain('共 2500 字符，已截断');
    });

    it('失败原因随草稿带出（追问最常见的起点就是「这次为什么失败」）', () => {
        const params = buildRunForkParams(run({ status: 'failed', err: 'tool timeout' }), labels);

        expect(params.err).toBe('tool timeout');
    });
});
