/**
 * 中断处理器注册表
 * 对齐 tokhub interrupts/registry.ts 的注册表模式
 *
 * 设计原则（开闭原则）：
 * - 新增中断类型只需 registerInterruptHandler()，框架零修改
 * - 扩展点即消费方真实需要的最小集合：
 *   - buildFromHistoryItem：chatStore 历史加载时恢复 pending 中断
 *   - isPending：配合 buildFromHistoryItem 判断是否可恢复
 *   - interpretDecision：eventWriter 决策落地时委托解释（类型特定 action 语义）
 *   - getPendingLabel / getResumeBadge：ToolCallPart 徽章文案/样式覆盖
 *   - component：ToolCallPart 经 getInterruptComponent 渲染中断操作 UI
 *   （流式到达的 InterruptEvent 无需 per-type 转换，直接作为决策通道数据）
 */

import type { Component } from 'vue';
import type { InterruptEvent, TurnItem } from '../protocol/types';

// ==================== 中断状态 ====================

/** 中断状态（前端展示用） */
export interface InterruptState {
    /** 中断类型标识，使用 (string & {}) 保留字面量自动补全同时允许透传 */
    kind: string;
    /** 中断唯一 ID */
    interruptId: string;
    /** 所属 turn */
    turnId: string;
    /** 描述文本 */
    description: string;
    /** 工具名称 */
    toolName?: string;
    /** 工具调用 ID */
    toolCallId?: string;
    /** 状态（取值域由 registries/statuses.ts 的 InterruptResumeStatus 统一维护） */
    status: string;
    /** 扩展元数据 */
    metadata?: Record<string, unknown>;
}

/** 中断决策请求（用户操作后提交给后端的数据） */
export interface InterruptDecision {
    interruptId: string;
    interruptType: string;
    turnId: string;
    action: string;
    payload?: Record<string, unknown>;
}

/** 决策解释结果（对齐 tokhub interpretDecision 返回形状） */
export interface DecisionInterpretation {
    /** 标准化恢复状态（取值见 registries/statuses.ts InterruptResumeStatus） */
    status: string;
    reason?: string;
    answer?: string;
}

// ==================== 处理器接口 ====================

/**
 * 中断处理器接口（对齐 tokhub interrupts/types.ts InterruptHandler）
 * 每种中断类型实现此接口并注册到 registry，
 * 公共代码（eventWriter / ToolCallPart / chatStore）只经注册表调用，不引用具体类型
 */
export interface InterruptHandler {
    /** 中断类型标识 */
    readonly type: string;

    /**
     * 从历史 TurnItem 恢复 pending 中断
     * 用于页面加载后恢复未完成的中断；非本类型返回 null。
     * 中断信息存于 tool_call item 的 extra 列（对齐 tokhub），由 itemExtra 传入
     */
    buildFromHistoryItem(item: TurnItem, itemExtra?: Record<string, unknown>): InterruptState | null;

    /**
     * 判断中断是否处于待处理状态
     */
    isPending(state: InterruptState): boolean;

    /**
     * 解释用户决策，返回标准化恢复状态（对齐 tokhub interpretDecision）
     * 将类型特定的 action/payload（如审批 reject + reason）转换为通用状态，
     * 公共代码（eventWriter）不解析 action 语义，委托给此方法
     */
    interpretDecision(action: string, payload?: Record<string, unknown>): DecisionInterpretation;

    /**
     * 待决策徽章文案覆盖（可选，对齐 tokhub getPendingLabel）
     * 不同类型 pending 文案不同（"待审批"/"待提交"...），返回 null 使用通用文案
     */
    getPendingLabel?: (t: (key: string) => string) => string | null;

    /**
     * 决议徽章配置覆盖（可选，对齐 tokhub getResumeBadge）
     * 自定义类型可定义自己的决议徽章（文案 i18n key + 样式类），
     * 返回 null 回退 ToolCallPart 的通用配置
     */
    getResumeBadge?: (
        status: string,
        t: (key: string) => string,
    ) => { label: string; badgeClass: string } | null;

    /** 中断操作 UI 组件（单一注册表：业务逻辑与操作组件同源） */
    component: Component;
}

// ==================== 注册表 ====================

const handlerRegistry = new Map<string, InterruptHandler>();

/**
 * 注册中断处理器
 * @returns uninstaller 函数，调用后移除注册
 *
 * @example
 * const uninstall = registerInterruptHandler({
 *   type: 'approval',
 *   buildFromHistoryItem: (item) => ({ ... }),
 *   isPending: (state) => state.status === 'pending',
 *   component: ApprovalInterruptVue,
 * })
 */
export function registerInterruptHandler(handler: InterruptHandler): () => void {
    handlerRegistry.set(handler.type, handler);
    return () => handlerRegistry.delete(handler.type);
}

/**
 * 按类型查找中断处理器
 */
export function getInterruptHandler(type?: string): InterruptHandler | undefined {
    return type ? handlerRegistry.get(type) : undefined;
}

/**
 * 遍历所有已注册的处理器
 */
export function forEachInterruptHandler(fn: (handler: InterruptHandler) => void): void {
    handlerRegistry.forEach(fn);
}
