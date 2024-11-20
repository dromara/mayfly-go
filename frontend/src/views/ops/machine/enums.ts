import { EnumValue } from '@/common/Enum';

export const MachineProtocolEnum = {
    Ssh: EnumValue.of(1, 'SSH'),
    Rdp: EnumValue.of(2, 'RDP'),
    Vnc: EnumValue.of(3, 'VNC'),
};

// 脚本执行结果类型
export const ScriptResultEnum = {
    Result: EnumValue.of(1, 'machine.scriptResultEnumResult').tagTypeSuccess(),
    NoResult: EnumValue.of(2, 'machine.scriptResultEnumNoResult').tagTypeDanger(),
    RealTime: EnumValue.of(3, 'machine.scriptResultEnumRealTime').tagTypeInfo(),
};

// 脚本类型
export const ScriptTypeEnum = {
    Private: EnumValue.of(1, 'machine.scriptTypeEnumPrivate'),
    Public: EnumValue.of(2, 'machine.scriptTypeEnumPublic'),
};

// 文件类型枚举
export const FileTypeEnum = {
    Directory: EnumValue.of(1, 'machine.directory'),
    File: EnumValue.of(2, 'machine.file'),
};

// 计划任务状态
export const CronJobStatusEnum = {
    Enable: EnumValue.of(1, 'common.enable').tagTypeSuccess(),
    Disable: EnumValue.of(-1, 'common.disable').tagTypeDanger(),
};

// 计划任务保存执行结果类型
export const CronJobSaveExecResTypeEnum = {
    No: EnumValue.of(-1, '不记录').tagTypeDanger(),
    OnError: EnumValue.of(1, '错误时记录').tagTypeWarning(),
    Yes: EnumValue.of(2, '记录').tagTypeSuccess(),
};

// 计划任务执行记录状态
export const CronJobExecStatusEnum = {
    Error: EnumValue.of(-1, 'machine.cronJobExecStatusEnumFail').tagTypeDanger(),
    Success: EnumValue.of(1, 'machine.cronJobExecStatusEnumSuccess').tagTypeSuccess(),
};
