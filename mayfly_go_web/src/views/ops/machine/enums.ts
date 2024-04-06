import { EnumValue } from '@/common/Enum';

export const MachineProtocolEnum = {
    Ssh: EnumValue.of(1, 'SSH'),
    Rdp: EnumValue.of(2, 'RDP'),
};

// 脚本执行结果类型
export const ScriptResultEnum = {
    Result: EnumValue.of(1, '有结果'),
    NoResult: EnumValue.of(2, '无结果'),
    RealTime: EnumValue.of(3, '实时交互'),
};

// 脚本类型
export const ScriptTypeEnum = {
    Private: EnumValue.of(1, '私有'),
    Public: EnumValue.of(2, '公共'),
};

// 文件类型枚举
export const FileTypeEnum = {
    Directory: EnumValue.of(1, '目录'),
    File: EnumValue.of(2, '文件'),
};

// 授权凭证认证方式枚举
export const AuthMethodEnum = {
    Password: EnumValue.of(1, '密码').tagTypeSuccess(),
    PrivateKey: EnumValue.of(2, '秘钥'),
};

// 计划任务状态
export const CronJobStatusEnum = {
    Enable: EnumValue.of(1, '启用').tagTypeSuccess(),
    Disable: EnumValue.of(-1, '禁用').tagTypeDanger(),
};

// 计划任务保存执行结果类型
export const CronJobSaveExecResTypeEnum = {
    No: EnumValue.of(-1, '不记录').tagTypeDanger(),
    OnError: EnumValue.of(1, '错误时记录').tagTypeWarning(),
    Yes: EnumValue.of(2, '记录').tagTypeSuccess(),
};

// 计划任务执行记录状态
export const CronJobExecStatusEnum = {
    Error: EnumValue.of(-1, '错误').tagTypeDanger(),
    Success: EnumValue.of(1, '成功').tagTypeSuccess(),
};
