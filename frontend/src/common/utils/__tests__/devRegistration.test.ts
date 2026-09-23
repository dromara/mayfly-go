import { describe, expect, it, vi } from 'vitest';

import { reportDuplicateRegistration } from '../devRegistration';

/** 告警经 queueMicrotask 聚合刷出，断言前需让出微任务队列 */
const settle = () => new Promise((resolve) => setTimeout(resolve, 0));

describe('common/utils/devRegistration 重复注册聚合上报', () => {
    it('同一注册表的一批重复只产生一条告警，且明细保留全部 key', async () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

        reportDuplicateRegistration('unit-tree-command', 'cmd.a');
        reportDuplicateRegistration('unit-tree-command', 'cmd.b');
        reportDuplicateRegistration('unit-tree-command', 'cmd.c');
        await settle();

        expect(warn).toHaveBeenCalledTimes(1);
        const message = String(warn.mock.calls[0][0]);
        expect(message).toContain('unit-tree-command');
        expect(message).toContain('3 项');
        ['cmd.a', 'cmd.b', 'cmd.c'].forEach((key) => expect(message).toContain(key));
    });

    it('不同注册表各自独立成条，不互相合并', async () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

        reportDuplicateRegistration('unit-resource', '1');
        reportDuplicateRegistration('unit-contributor', 'db-table');
        await settle();

        expect(warn).toHaveBeenCalledTimes(2);
    });

    it('达到批量阈值时在告警里点明符合 HMR 重放特征，便于区分真实冲突', async () => {
        const warn = vi.spyOn(console, 'warn').mockImplementation(() => {});

        for (let i = 0; i < 5; i++) {
            reportDuplicateRegistration('unit-batch', `kind.${i}`);
        }
        await settle();

        expect(warn).toHaveBeenCalledTimes(1);
        expect(String(warn.mock.calls[0][0])).toContain('HMR');
    });
});
