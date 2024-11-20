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
