/**
 * 确认中断处理器（handlers/human-input/humanInputHandler.ts 模式）
 *
 * 决策动作：confirm（携带 selected 选项）/ cancel，
 * 默认决策解释（工厂提供）已覆盖：confirm → resolved / cancel → cancelled。
 */

import ConfirmationInterrupt from '../ConfirmationInterrupt.vue';
import { createInterruptHandler } from '../helpers';

export const confirmationHandler = createInterruptHandler('interrupt_confirmation', ConfirmationInterrupt);
