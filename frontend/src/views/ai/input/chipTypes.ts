/**
 * 芯片类型注册（模块导入副作用，由 usePromptEditor 与 InlineChip 各自 import 触发）
 *
 * - skill：`/` 触发的技能引用芯片
 * - resource：`@` 触发的资源引用芯片（机器/数据库等），随消息下发使模型明确目标
 */
import { registerChipType } from './chipRegistry';

registerChipType({
    type: 'skill',
    icon: () => 'zap',
    // `$技能码` 与内置 Runbook 指令里既有的引用写法一致（模型据此 skill_read 加载全文）
    toText: (attrs) => `$${(attrs.data as Record<string, unknown>)?.id ?? attrs.label ?? ''}`,
    fromSegment: (seg) => {
        const code = String(seg.extra?.skillCode ?? '');
        if (!code) return null;
        return { type: 'skill', label: seg.text, data: { id: code } };
    },
    color: {
        bg: 'var(--el-color-primary-light-9)',
        text: 'var(--el-color-primary)',
        border: 'var(--el-color-primary-light-7)',
    },
    extractSegment: (attrs) => ({
        type: 'skill',
        text: String(attrs.label || ''),
        extra: {
            skillCode: (attrs.data as Record<string, unknown>)?.id ?? '',
        },
    }),
});

registerChipType({
    type: 'resource',
    // 图标按资源子类型细分（机器=server、数据库=database）：判定收在注册处，
    // 编辑器与消息回显都经 chipIcon() 取同一个结果
    icon: (ctx) => (ctx.data?.resourceType === 'db' ? 'database' : 'server'),
    // `@资源名` 只是可读占位：可直接取参的定位信息在段的 extra 里（派发时展开）
    toText: (attrs) => `@${attrs.label ?? ''}`,
    fromSegment: (seg) => {
        const extra = seg.extra || {};
        // 没有资源类型就无法确定定位口径（machine/db 的渲染器不同），降级为纯文本更安全
        if (!extra.resourceType) return null;
        return { type: 'resource', label: seg.text, data: { ...extra } };
    },
    color: {
        bg: 'var(--el-color-success-light-9)',
        text: 'var(--el-color-success)',
        border: 'var(--el-color-success-light-7)',
    },
    extractSegment: (attrs) => {
        const data = (attrs.data as Record<string, unknown>) || {};
        return {
            type: 'resource',
            text: String(attrs.label || ''),
            // extra 携带完整定位标识（id/code/ip/port/authCertName/db），后端渲染为引用文本，
            // 确保工具调用（machineId/machineIp/machinePort/authCertName/dbId/dbName 等）参数准确
            extra: {
                resourceType: data.resourceType ?? '',
                id: data.id ?? '',
                code: data.code ?? '',
                ip: data.ip ?? '',
                port: data.port ?? '',
                authCertName: data.authCertName ?? '',
                username: data.username ?? '',
                // 数据库引用选到物理库时的库名（db 工具 dbName 参数）
                db: data.db ?? '',
            },
        };
    },
});
