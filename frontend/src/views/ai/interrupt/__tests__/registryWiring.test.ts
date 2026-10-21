import { describe, expect, it } from 'vitest';
import { getInterruptComponent } from '../index';
import { getInterruptHandler } from '../../registries/interruptRegistry';
import GenericInterrupt from '../GenericInterrupt.vue';

/**
 * 内置中断处理器的**接线守卫**。
 *
 * 缺陷类：三种内置中断的注册是 `interrupt/index.ts` 的模块副作用（三行 registerInterruptHandler）。
 * 漏掉任意一行不会报错、不会编译失败——该类型会静默退化成 GenericInterrupt：
 * 审批/补参的专属卡片、决议徽章文案、getPendingLabel 覆盖全部消失，用户只看到一张通用卡片。
 * 此前没有任何用例断言过「内置类型确实注册上了」（注册表只有一个 Map，没有自检）。
 *
 * 前提：import { getInterruptComponent } from '../index' 就是为了触发注册副作用——
 * 这正是本守卫要锁住的依赖关系，不要改为从 registries 直接导入
 */
const BUILTIN_TYPES = ['interrupt_approval', 'interrupt_confirmation', 'interrupt_param_completion'];

describe('内置中断处理器接线', () => {
    it.each(BUILTIN_TYPES)('%s 必须注册且映射到专属组件', (type) => {
        const handler = getInterruptHandler(type);
        expect(handler, `${type} 未注册：会静默退化为通用卡片`).toBeDefined();
        expect(handler?.component, `${type} 没有绑定专属组件`).toBeDefined();
        // 专属组件与兜底组件必须是两个不同的组件（否则等于退化成通用卡片）
        expect(getInterruptComponent(type)).not.toBe(GenericInterrupt);
    });

    it('未知类型必须降级到通用卡片（不抛错、不空渲染）', () => {
        expect(getInterruptComponent('__unknown_interrupt__')).toBe(GenericInterrupt);
        // 缺省类型同样走兜底（历史数据可能没有 type）
        expect(getInterruptComponent(undefined)).toBe(GenericInterrupt);
    });
});
