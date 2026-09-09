/**
 * 参数补全中断处理器（参数提交类 handler 模式）
 *
 * 决策动作：complete（payload 携带补全后的参数）/ cancel，
 * 默认决策解释（工厂提供）已覆盖：complete → resolved / cancel → cancelled。
 */

import ParamCompletionInterrupt from '../param-completion/index.vue';
import { createInterruptHandler } from '../helpers';

export const paramCompletionHandler = createInterruptHandler('interrupt_param_completion', ParamCompletionInterrupt);
