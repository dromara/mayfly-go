/**
 * 轮次收尾通知（notice）的写入时机契约。
 *
 * 与后端 decideTurnFinalization 单测配对：后端保证「用户取消/拒绝」不下发 error 事件，
 * 前端保证「红色错误提示只由 error 事件产生」，且 cancelled 终态不再叠加任何提示。
 * 两段合起来才构成「用户点取消不会闪一行红色故障提示」的完整证据，缺一段都可能被将来
 * 单侧改动破坏（例如在 onTurnCompleted 里给 cancelled 也补一条提示）。
 */
import { defineComponent, ref } from 'vue';
import { mount } from '@vue/test-utils';
import { createPinia, setActivePinia } from 'pinia';
import { createI18n } from 'vue-i18n';
import { beforeEach, describe, expect, it } from 'vitest';
import { useChatCallbacks } from '../stream/useChatCallbacks';
import { useChatStore } from '../stores/chatStore';
import { TurnStatus } from '../registries/statuses';
import type { NoticePart } from '../protocol/types';

const CONV_ID = 4200;

/** 在 setup 上下文里取出回调（useI18n 需要组件注入，useChatStore 需要 active pinia） */
function buildCallbacks() {
    const pinia = createPinia();
    setActivePinia(pinia);
    const i18n = createI18n({ legacy: false, locale: 'zh', messages: { zh: {} } });
    let callbacks: ReturnType<typeof useChatCallbacks> | undefined;
    mount(defineComponent({
        setup() {
            callbacks = useChatCallbacks(ref(CONV_ID));
            return () => null;
        },
    }), { global: { plugins: [pinia, i18n] } });
    return callbacks!;
}

/** 该会话里所有已写入的 notice part */
function notices(): NoticePart[] {
    const slice = useChatStore().getOrCreateSlice(CONV_ID);
    return slice.messages.flatMap((msg) => (msg.parts ?? []).filter((part) => part.type === 'notice'));
}

describe('轮次收尾通知写入时机', () => {
    beforeEach(() => {
        setActivePinia(createPinia());
    });

    it('用户取消参数补全/拒绝审批（cancelled）不写任何提示', () => {
        const callbacks = buildCallbacks();
        callbacks.onTurnStart('t-cancel', CONV_ID);
        callbacks.onTurnCompleted('t-cancel', TurnStatus.Cancelled);

        // 收尾文案由服务端落的 i18n 气泡承担，前端再叠一条会把「我拒了这条命令」显示成「我中断了整轮生成」
        expect(notices()).toEqual([]);
    });

    it('显式停止（stopped）只写一条停止提示', () => {
        const callbacks = buildCallbacks();
        callbacks.onTurnStart('t-stop', CONV_ID);
        callbacks.onTurnCompleted('t-stop', TurnStatus.Stopped);

        const list = notices();
        expect(list).toHaveLength(1);
        expect(list[0].kind).toBe('stopped');
    });

    it('红色错误提示只由 error 事件产生，且不受轮次终态影响', () => {
        const callbacks = buildCallbacks();
        callbacks.onTurnStart('t-err', CONV_ID);
        callbacks.onError('dial tcp connect: connection refused', 'agent');

        const list = notices();
        expect(list).toHaveLength(1);
        expect(list[0].kind).toBe('error');
        expect(list[0].text).toContain('connection refused');
    });

    it('成功收尾不写提示', () => {
        const callbacks = buildCallbacks();
        callbacks.onTurnStart('t-ok', CONV_ID);
        callbacks.onTurnCompleted('t-ok', TurnStatus.Success);
        expect(notices()).toEqual([]);
    });
});
