import EnumValue from '@/common/Enum';

export enum TerminalStatus {
    Error = -1,
    NoConnected = 0,
    Connected = 1,
    Disconnected = 2,
}

export const TerminalStatusEnum = {
    Error: EnumValue.of(TerminalStatus.Error, '连接出错').setExtra({ iconColor: 'var(--el-color-error)' }),
    NoConnected: EnumValue.of(TerminalStatus.NoConnected, '未连接').setExtra({ iconColor: 'var(--el-color-primary)' }),
    Connected: EnumValue.of(TerminalStatus.Connected, '连接成功').setExtra({ iconColor: 'var(--el-color-success)' }),
    Disconnected: EnumValue.of(TerminalStatus.Disconnected, '连接失败').setExtra({ iconColor: 'var(--el-color-error)' }),
};
