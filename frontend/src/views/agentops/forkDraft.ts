import type { AgentRun } from './types';

/** 单字段进草稿的字符预算：指令/结论各截一段，避免把整段工具明细灌进输入框淹没追问本身 */
const FIELD_BUDGET = 2000;

/** 截断并在超长时标注原始长度（用户能判断「还有多少没贴进来」） */
function clip(text: string): string {
    const value = (text ?? '').trim();
    if (!value) return '—';
    if (value.length <= FIELD_BUDGET) return value;
    return `${value.slice(0, FIELD_BUDGET)}\n…（共 ${value.length} 字符，已截断）`;
}

/**
 * 组装「带上下文追问」的草稿正文（占位符由 i18n 模板提供，本函数只负责取值与截断）。
 *
 * 为什么是「整理摘要」而不是把那次运行的会话原样搬过去：追问发生在**新的用户会话**里，
 * 模型看不到那次运行的会话历史，需要的是「谁触发、对什么资源、被要求做什么、结论是什么」这份
 * 可复述的上下文；整段会话既超长又含工具返回的敏感明细，贴进输入框只会淹没用户真正想问的那句。
 *
 * 状态/档位/触发方式传的是已本地化的标签（调用方经 enumLabel 转换），因此这里不做任何文案拼接
 */
export function buildRunForkParams(run: AgentRun, labels: { status: string; profile: string; trigger: string }): Record<string, string> {
    return {
        runNo: run.runNo,
        title: run.title || '—',
        status: labels.status || '—',
        profile: labels.profile || '—',
        trigger: labels.trigger || '—',
        owner: run.ownerUsername || '—',
        // input 是运行自持的「本次实际被要求做什么」（含目标资源上下文），
        // 比任务的 targetResources 更准确：关联任务与内联触发都收敛到这一个字段
        input: clip(run.input ?? ''),
        instruction: clip(run.instruction ?? ''),
        summary: clip(run.summary ?? ''),
        err: clip(run.err ?? ''),
    };
}
