/**
 * 审批中断处理器（对齐 tokhub handlers/approval/approvalHandler.ts）
 *
 * 决策动作：approve / reject（reject 携带 reason），
 * 默认决策解释（工厂提供）已覆盖：approve → approved / reject → rejected。
 * 类型特定语义（如自定义 payload 结构）时经 createInterruptHandler overrides 覆盖。
 */

import ApprovalInterrupt from '../ApprovalInterrupt.vue';
import { createInterruptHandler } from '../helpers';

export const approvalHandler = createInterruptHandler('interrupt_approval', ApprovalInterrupt);
