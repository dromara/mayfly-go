/**
 * 触发器注册表（对齐 tokhub 的 TriggerConfig 聚合方式）
 *
 * 触发词检测正则与全量触发字符均由注册表派生，新增触发类型零改动。
 *
 * 注意：具体触发器（skillTrigger/resourceTrigger）在文件底部显式注册，
 * 而非在触发器实现文件顶层自注册——自注册会经 index.ts re-export 形成
 * registry ↔ 实现文件的循环依赖，触发器顶层 registerTrigger 调用时
 * triggerDefs 尚未初始化（TDZ ReferenceError 导致整个应用白屏）。
 */

import type { TriggerDef } from './types';
import { registerSkillTrigger } from './skillTrigger';
import { registerResourceTrigger } from './resourceTrigger';

const triggerDefs: TriggerDef[] = [];

/** 注册触发器（重复 kind 覆盖，支持热替换） */
export function registerTrigger(def: TriggerDef): void {
    const idx = triggerDefs.findIndex((d) => d.kind === def.kind);
    if (idx >= 0) {
        triggerDefs[idx] = def;
    } else {
        triggerDefs.push(def);
    }
}

export function getTriggerDefs(): TriggerDef[] {
    return triggerDefs;
}

export function findTriggerDef(char: string): TriggerDef | undefined {
    return triggerDefs.find((d) => d.chars.includes(char));
}

/** 全部触发字符（含全角变体，供 lastIndexOf 检测） */
export function getTriggerChars(): string[] {
    return triggerDefs.flatMap((d) => d.chars);
}

const escapeCharClass = (c: string) => c.replace(/[\\\]\-\^]/g, '\\$&');

/**
 * 触发词正则：光标前以触发字符开头且无空格的词（如 '/sk'、'＠机器'）。
 * 前置边界为行首/空白/CJK 字符（中文后直接输入 @ 也触发；
 * 纯字母前不触发，避免邮箱 test@qq 误触）。
 */
export function buildTriggerTokenRegex(): RegExp {
    const chars = getTriggerChars().map(escapeCharClass).join('');
    return new RegExp(`(?:^|\\s|[\\u4e00-\\u9fff])([${chars}][^\\s]*)$`);
}

// 内置触发器注册（triggerDefs 初始化之后执行，避免 TDZ）
registerSkillTrigger();
registerResourceTrigger();
