/**
 * 芯片类型注册（模块导入副作用，index.ts 中触发）
 *
 * - skill：`/` 触发的技能引用芯片
 * - resource：`@` 触发的资源引用芯片（机器/数据库等），随消息下发使模型明确目标
 */
import { registerChipType } from './chipRegistry';

registerChipType({
    type: 'skill',
    icon: 'zap',
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
    icon: 'server',
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
