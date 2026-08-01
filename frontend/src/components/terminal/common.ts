import EnumValue from '@/common/Enum';

export enum TerminalStatus {
    Error = -1,
    NoConnected = 0,
    Connected = 1,
    Disconnected = 2,
}

export const TerminalStatusEnum = {
    Error: EnumValue.of(TerminalStatus.Error, 'components.terminal.connError').setExtra({ iconColor: 'var(--el-color-error)' }),
    NoConnected: EnumValue.of(TerminalStatus.NoConnected, 'components.terminal.notConn').setExtra({ iconColor: 'var(--el-color-primary)' }),
    Connected: EnumValue.of(TerminalStatus.Connected, 'components.terminal.connSuccess').setExtra({ iconColor: 'var(--el-color-success)' }),
    Disconnected: EnumValue.of(TerminalStatus.Disconnected, 'components.terminal.connFail').setExtra({ iconColor: 'var(--el-color-error)' }),
};

/**
 * 终端元数据，由各调用方自行填充（如机器信息）。
 * TerminalDialog 仅读取 id/protocol/selectAuthCert.name，其余字段通过索引签名透传给插槽。
 */
export interface TerminalMeta {
    id?: number;
    protocol?: number;
    selectAuthCert?: { name?: string; [key: string]: unknown };
    [key: string]: unknown;
}
