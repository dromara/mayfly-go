import { type InjectionKey, inject, provide } from 'vue';

/**
 * 消息渲染模式：同一套消息渲染组件服务两种消费面。
 *
 * - `chat`：AI 助手里可继续对话的会话（有输入框、可编辑自己发的消息、可处理待决中断）
 * - `transcript`：运维运行轨迹的只读视图（执行历史详情/整页），任何写操作类控件都不该出现
 *
 * 为什么走 provide/inject 而不是逐层 props：判定落在 MessageBubble（编辑按钮）与
 * ToolCallPart（中断动作）两个深处，中间隔着 AssistantMessageBubble / ProcessGroup 两层纯转发；
 * 用 props 穿透会把契约变成「新增展示层组件时必须记得透传 mode」的隐性约定，漏一层即静默失效
 * （只读视图里冒出可点的审批按钮，且服务端拒绝后前端只显示一条错误）。
 *
 * 单一注入点：由 MessageList 按自身 mode 提供，任何消息列表的消费方只需传一个 prop。
 */
export type MessageRenderMode = 'chat' | 'transcript';

/**
 * 未声明时的缺省模式：只读。
 *
 * 必须是单点常量：provide 端（MessageList）与 inject 端（子组件）各自写一个缺省值，
 * 就会出现两边取值相反、子组件的 fail-closed 被上层抵消的情况（实际踩过）
 */
export const DEFAULT_RENDER_MODE: MessageRenderMode = 'transcript';

/**
 * 是否只读轨迹态（单一判定函数）：MessageList 判自己的 prop、子组件判 inject 到的值都走它，
 * 新增第三种模式时只需改这一处
 */
export function isTranscriptMode(mode: MessageRenderMode): boolean {
    return mode === 'transcript';
}

const renderModeKey: InjectionKey<MessageRenderMode> = Symbol('messageRenderMode');

/** 由消息列表容器调用，向下声明本棵子树的渲染模式 */
export function provideMessageRenderMode(mode: MessageRenderMode) {
    provide(renderModeKey, mode);
}

/**
 * 当前渲染模式。
 *
 * 缺省按 DEFAULT_RENDER_MODE（只读）处理：漏接 provide 的后果必须是「少了一些按钮」，而不是
 * 「只读轨迹里冒出可点的审批/编辑控件」——后者要等服务端拒接才看得见，界面只表现为一条错误。
 * 聊天侧由 MessageList 显式 provide chat，不依赖这里的缺省
 */
export function useMessageRenderMode(): MessageRenderMode {
    return inject(renderModeKey, DEFAULT_RENDER_MODE);
}

/**
 * 是否只读轨迹态：所有「写操作类控件」的统一判定入口。
 *
 * 集中成一个函数而不是各处比字符串，是为了让「哪些控件算写操作」可被单点审计
 * （新增可交互入口时只需 grep 这个函数决定要不要让位）
 */
export function useReadonlyTranscript(): boolean {
    return isTranscriptMode(useMessageRenderMode());
}
