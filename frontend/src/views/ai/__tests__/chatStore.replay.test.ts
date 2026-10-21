import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useChatStore } from '../stores/chatStore';
import type { TurnGroupVO } from '../protocol/types';

/** 模拟服务端返回的一轮历史（用户消息 + reasoning + assistant 正文 + tool_call） */
function mockGroup(turnId: string): TurnGroupVO {
    return {
        turnId,
        items: [
            {
                id: 1,
                turnId,
                itemType: 'message',
                itemId: `u-${turnId}`,
                status: 'success',
                createTime: '2026-09-02T10:30:00',
                item: { type: 'message', id: `u-${turnId}`, role: 'user', content: [{ type: 'input_text', text: '你好' }] },
            },
            {
                id: 2,
                turnId,
                itemType: 'reasoning',
                itemId: `r-${turnId}`,
                status: 'success',
                createTime: '2026-09-02T10:31:00',
                item: { type: 'reasoning', id: `r-${turnId}`, text: '思考内容' },
            },
            {
                id: 3,
                turnId,
                itemType: 'message',
                itemId: `a-${turnId}`,
                status: 'success',
                createTime: '2026-09-02T10:32:00',
                item: { type: 'message', id: `a-${turnId}`, role: 'assistant', content: [{ type: 'output_text', text: 'AI 回复正文' }] },
            },
            {
                id: 4,
                turnId,
                itemType: 'tool_call',
                itemId: `t-${turnId}`,
                status: 'success',
                createTime: '2026-09-02T10:33:00',
                item: {
                    type: 'tool_call',
                    id: `t-${turnId}`,
                    tool_call_id: `tc-${turnId}`,
                    tool_name: 'MachineCommandExec',
                    arguments: '{}',
                    status: 'success',
                    output: 'ok',
                },
            },
        ],
    };
}

describe('chatStore 刷新后历史重放场景', () => {
    beforeEach(() => setActivePinia(createPinia()));

    it('loadTurnGroups 重放后不应残留 streaming 标志', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [mockGroup('t1')]);
        const slice = store.getSlice(1)!;
        const streamingMsgs = slice.messages.filter((m) => m.role === 'assistant' && m.streaming);
        expect(streamingMsgs, '历史 assistant 消息不应是 streaming 状态').toHaveLength(0);
    });

    it('重放后 resetConvStreaming 不应删除历史 assistant 消息', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [mockGroup('t1')]);
        const before = store.getSlice(1)!.messages.length;
        const hadStreaming = store.resetConvStreaming(1);
        const after = store.getSlice(1)!.messages.length;
        expect(hadStreaming, '历史重放产物不算流式中间态').toBe(false);
        expect(after, 'attach 前置重置不得删除历史消息').toBe(before);
    });

    it('重放产物的 assistant 消息应含完整 parts', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [mockGroup('t1')]);
        const msgs = store.getSlice(1)!.messages;
        const assistant = msgs.find((m) => m.role === 'assistant');
        expect(assistant).toBeDefined();
        expect(assistant!.parts!.some((p) => p.type === 'text' && p.text === 'AI 回复正文')).toBe(true);
        expect(assistant!.parts!.some((p) => p.type === 'reasoning')).toBe(true);
        expect(assistant!.parts!.some((p) => p.type === 'tool_call')).toBe(true);
        expect(assistant!.content).toContain('AI 回复正文');
    });

    it('消息时间应取 item 落库创建时间而非本地当前时间', () => {
        const store = useChatStore();
        store.loadTurnGroups(1, [mockGroup('t1')]);
        const msgs = store.getSlice(1)!.messages;
        const user = msgs.find((m) => m.role === 'user');
        const assistant = msgs.find((m) => m.role === 'assistant');
        // createTime '2026-09-02T10:30:00'（本地时区解析）→ HH:mm
        expect(user!.time).toBe('10:30');
        // assistant 消息时间 = 首个 assistant item（reasoning 10:31）的创建时间
        expect(assistant!.time).toBe('10:31');
    });
});

describe('历史用户消息的文本形态（引用标记不得在重放时丢失）', () => {
    /** 一条带资源芯片的用户消息：文本-芯片-文本 */
    const replayChipMessage = (convId: number, itemId: string) => {
        const store = useChatStore();
        store.loadTurnGroups(convId, [
            {
                turnId: 't-chip',
                items: [
                    {
                        id: 1,
                        turnId: 't-chip',
                        itemType: 'message',
                        itemId,
                        status: 'success',
                        createTime: '2026-09-02T10:30:00',
                        item: {
                            type: 'message',
                            id: itemId,
                            role: 'user',
                            content: [
                                { type: 'input_text', text: '巡检 ' },
                                { type: 'resource', text: '机器A', extra: { resourceType: 'machine', id: 11 } },
                                { type: 'input_text', text: ' 的磁盘' },
                            ],
                        },
                    },
                ],
            },
        ] as unknown as TurnGroupVO[]);
        return store.getSlice(convId)?.messages.find((m) => m.id === itemId);
    };

    it('芯片段按注册表文本形态进入 content', () => {
        // 旧实现按 output_text|input_text 白名单过滤，结果会是「巡检  的磁盘」——
        // 复制拿到缺引用的文本，「编辑→发送」发出去的消息一个 @资源 都没有
        expect(replayChipMessage(77, 'u-chip')?.content).toBe('巡检 @机器A 的磁盘');
    });

    it('同一份文本形态在重放路径上稳定可重复（不随切片重建漂移）', () => {
        const first = replayChipMessage(88, 'u-a')?.content;
        const second = replayChipMessage(88, 'u-b')?.content;
        expect(second).toBe(first);
    });
});
