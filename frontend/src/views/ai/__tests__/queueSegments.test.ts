/**
 * 队列与提交数据的段序列保真
 *
 * 为什么单独测：芯片的定位信息只在 segments 的 extra 里。「忙时入队 → 空闲 flush 重发」
 * 这条链路上任何一环丢段，用户看到的仍是带引用的消息（文本形态带 @标签），但模型收到的是
 * 纯文本、且全程没有任何报错——属于静默语义丢失。静态类型允许传空数组（`segments: []` 完全合法），
 * 门禁抓不住，实测就是这么漏过一次（编辑重发把 segments 置空）。
 */
import { beforeEach, describe, expect, it } from 'vitest';
import { createPinia, setActivePinia } from 'pinia';
import { useChatStore } from '../stores/chatStore';
import type { useChatStore as UseChatStore } from '../stores/chatStore';
import { buildSubmitData } from '../input/submitData';
import type { ChipSegment } from '../input/chipRegistry';

type ChatStore = ReturnType<typeof UseChatStore>;

const machineSegment: ChipSegment = {
    type: 'resource',
    text: 'aliyun(root)',
    extra: { resourceType: 'machine', id: '1', ip: '59.110.115.25', port: '22', authCertName: 'tVUcRFKwOC', username: 'root' },
};

const skillSegment: ChipSegment = { type: 'skill', text: 'db-ops-guide', extra: { id: 'db-ops-guide' } };

/** 段自身的 extra 必须逐字段活着：少一个 ip/authCertName，工具参数展开就会去猜或中断 */
const expectMachineExtra = (seg: ChipSegment | undefined) => {
    expect(seg?.extra).toMatchObject({
        resourceType: 'machine',
        id: '1',
        ip: '59.110.115.25',
        port: '22',
        authCertName: 'tVUcRFKwOC',
        username: 'root',
    });
};

describe('buildSubmitData 的文本/段成对契约', () => {
    it('段序列原样携带，文本 trim', () => {
        const data = buildSubmitData('  审查下该资源  ', [machineSegment]);
        expect(data.text).toBe('审查下该资源');
        expect(data.segments).toHaveLength(1);
        expectMachineExtra(data.segments[0]);
    });

    it('未传段时给空数组而不是 undefined（下游不必区分两种「没有」）', () => {
        expect(buildSubmitData('x', undefined).segments).toEqual([]);
    });

    it('空附件数组归一为不带该字段（避免「空数组」与「无附件」两条假分支）', () => {
        expect(buildSubmitData('x', [], []).attachments).toBeUndefined();
    });

    it('纯空白文本 trim 后为空串，供调用方判空早退', () => {
        expect(buildSubmitData('   \n ', [machineSegment]).text).toBe('');
    });
});

describe('待发送队列全程不丢段', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    const enqueueTwo = (store: ChatStore) => {
        store.enqueueMessage(7, buildSubmitData('第一条', [machineSegment]));
        store.enqueueMessage(7, buildSubmitData('第二条', [machineSegment, skillSegment]));
    };

    it('入队 → 出队保持顺序，且段内容逐字段完整', () => {
        const store = useChatStore();
        enqueueTwo(store);

        const first = store.dequeueMessage(7);
        expect(first?.data.text).toBe('第一条');
        expectMachineExtra(first?.data.segments[0]);

        const second = store.dequeueMessage(7);
        expect(second?.data.segments).toHaveLength(2);
        expect(second?.data.segments[1].type).toBe('skill');

        expect(store.dequeueMessage(7)).toBeUndefined();
    });

    it('拖拽重排后段跟着各自的项，不串行错位', () => {
        const store = useChatStore();
        enqueueTwo(store);
        store.reorderQueuedMessage(7, 1, 0);

        const queue = store.getSlice(7)!.queuedMessages;
        expect(queue[0].data.text).toBe('第二条');
        expect(queue[0].data.segments).toHaveLength(2);
        expect(queue[1].data.text).toBe('第一条');
        expectMachineExtra(queue[1].data.segments[0]);
    });

    it('编辑回填失败时原样退回队列，再取仍带完整定位信息', () => {
        const store = useChatStore();
        enqueueTwo(store);
        const item = store.getSlice(7)!.queuedMessages[0];
        store.removeQueuedMessage(7, item.id);
        expect(store.hasQueuedMessages(7)).toBe(true);

        // setValue 未就绪（返回 false）时调用方把原 data 退回队列：不能只退回文本
        store.enqueueMessage(7, item.data);
        expectMachineExtra(store.dequeueMessage(7)?.data.segments[0]);
    });

    it('队列只影响本会话切片（不串会话）', () => {
        const store = useChatStore();
        enqueueTwo(store);
        expect(store.hasQueuedMessages(8)).toBe(false);
        expect(store.getSlice(8)?.queuedMessages ?? []).toHaveLength(0);
    });
});


describe('flush 闸门：预读不消耗队列项', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('peek 不删项，只有确认被接受后才摘掉', () => {
        const store = useChatStore();
        store.enqueueMessage(7, buildSubmitData('第一条', [machineSegment]));
        store.enqueueMessage(7, buildSubmitData('第二条', [machineSegment]));

        const peeked = store.peekQueuedMessage(7);
        expect(peeked?.data.text).toBe('第一条');
        // 预读不得消耗队列：后端可能仍在收尾而拒收这一轮，此时消息必须还在
        expect(store.getSlice(7)!.queuedMessages).toHaveLength(2);

        store.removeQueuedMessage(7, peeked!.id);
        expect(store.getSlice(7)!.queuedMessages.map((q) => q.data.text)).toEqual(['第二条']);
    });

    it('空队列 peek 返回 undefined（不抛错，供 watch 早退）', () => {
        const store = useChatStore();
        expect(store.peekQueuedMessage(99)).toBeUndefined();
    });
});
