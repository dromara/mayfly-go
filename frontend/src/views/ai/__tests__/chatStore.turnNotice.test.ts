import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { i18n } from '@/i18n';
import { useChatStore } from '../stores/chatStore';
import { TURN_NOTICE_TEXT_KEY } from '../registries/noticeKinds';
import type { TurnGroupVO } from '../protocol/types';

/**
 * 「本轮被用户停止」的持久化与还原契约（c30）。
 *
 * 缺陷形态（真机实测）：停止提示原先只由服务端推一个事件、前端写进内存，零持久化。刷新或重进会话后
 * 那一轮只剩一段在句子中途断开的回答（截断处还留着未闭合的 `**` 加粗），与模型正常答完完全同形——
 * 回看的人会当成模型自己写崩了，而「被人停了」这个事实对后续判断（结论可不可信、要不要重跑）是关键信息。
 */
const t = (key: string) => i18n.global.t(key);

const stoppedTurnGroup = (): TurnGroupVO => ({
    turnId: 't-stop',
    items: [
        {
            id: 1,
            turnId: 't-stop',
            itemType: 'message',
            itemId: 'u-1',
            status: 'success',
            createTime: '2026-10-21T09:00:00',
            item: { type: 'message', id: 'u-1', role: 'user', content: [{ type: 'input_text', text: '列一下全部工具' }] },
        },
        {
            id: 2,
            turnId: 't-stop',
            itemType: 'message',
            itemId: 'a-1',
            status: 'success',
            createTime: '2026-10-21T09:00:05',
            // 截断的正文：末尾是未闭合的加粗标记，与「正常答完」在文本上无法区分
            item: { type: 'message', id: 'a-1', role: 'assistant', content: [{ type: 'output_text', text: 'propose_remediation | 强制安全' }] },
        },
        {
            id: 3,
            turnId: 't-stop',
            itemType: 'turn_notice',
            itemId: 'n-1',
            status: 'success',
            createTime: '2026-10-21T09:00:06',
            // 后端只存种类，不存句子（句子会把写入时的语言固化进历史）
            item: { type: 'turn_notice', id: 'n-1', notice_kind: 'stopped' },
        },
    ],
});

beforeEach(() => setActivePinia(createPinia()));

describe('停止提示的持久化还原', () => {
    it('历史里的 turn_notice item 必须还原成 notice part（缺它就等于缺陷原样存在）', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [stoppedTurnGroup()]);

        const assistant = store
            .getSlice(1)!
            .messages.find((m) => m.role === 'assistant');
        expect(assistant, '截断的 assistant 消息必须仍在').toBeDefined();

        const notice = assistant!.parts!.find((p) => p.type === 'notice');
        expect(notice, '刷新后必须仍能看出这一轮是被停止的，而不是模型答完了').toBeDefined();
        expect(notice!.kind).toBe('stopped');
    });

    it('还原出的文案必须与实时路径取自同一张键表（两边各写一遍就会在改文案时分叉）', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [stoppedTurnGroup()]);
        const notice = store.getSlice(1)!.messages.find((m) => m.role === 'assistant')!.parts!.find((p) => p.type === 'notice')!;

        expect(notice.text).toBe(t(TURN_NOTICE_TEXT_KEY.stopped));
    });

    it('notice part 按 (turn, kind) 幂等：实时写过的停止提示，历史重建或重进会话不得叠成两条', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [stoppedTurnGroup()]);
        const convId = 1;
        const turnId = 't-stop';

        // 模拟实时路径再来一次（同 turn 同种类）
        store.appendTurnNoticePart(convId, turnId, { type: 'notice', id: 'live-1', kind: 'stopped', text: t(TURN_NOTICE_TEXT_KEY.stopped) });
        store.appendTurnNoticePart(convId, turnId, { type: 'notice', id: 'live-2', kind: 'stopped', text: t(TURN_NOTICE_TEXT_KEY.stopped) });

        const assistant = store.getSlice(convId)!.messages.find((m) => m.role === 'assistant')!;
        const notices = assistant.parts!.filter((p) => p.type === 'notice');
        expect(notices, 'part.id 每次新生成，去重只能按种类；按 id 去重会堆出两条「已停止生成」').toHaveLength(1);
    });

    it('未登记的新种类不得被静默丢掉（静丢就等于回到本缺陷的形态）', () => {
        const store = useChatStore();
        const group = stoppedTurnGroup();
        group.items[2].item = { type: 'turn_notice', id: 'n-1', notice_kind: 'paused_by_gateway' } as never;
        store.loadTurnGroups(1, [group]);

        const notice = store.getSlice(1)!.messages.find((m) => m.role === 'assistant')!.parts!.find((p) => p.type === 'notice')!;
        expect(notice, '未来后端新增种类时前端宁可回显原始值，也不能把它吞掉').toBeDefined();
        expect(notice.text).toContain('paused_by_gateway');
    });
});
