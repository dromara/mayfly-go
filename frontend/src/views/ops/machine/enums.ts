import { EnumValue } from '@/common/Enum';

/** 机器默认端口（SSH）：新建态表单初值与协议默认端口的同源起点 */
export const MACHINE_DEFAULT_PORT = 22;

export const MachineProtocolEnum = {
    Ssh: EnumValue.of(1, 'SSH').setExtra({ defaultPort: MACHINE_DEFAULT_PORT }),
    Rdp: EnumValue.of(2, 'RDP').setExtra({ defaultPort: 3389 }),
    Vnc: EnumValue.of(3, 'VNC').setExtra({ defaultPort: 5901 }),
};

/**
 * 取协议对应的默认端口（端口属协议知识，与协议枚举同源维护，不得在各视图重复硬编码）
 *
 * 入参为 AutoForm 联动回调给出的原始值；非协议枚举值返回 undefined，由调用方保持当前端口。
 */
export const getProtocolDefaultPort = (protocol: unknown): number | undefined => {
    const matched = Object.values(MachineProtocolEnum).find((item) => item.value === Number(protocol));
    return matched?.extra?.defaultPort;
};

/**
 * 公共脚本归属的虚拟机器 id（对应后端 application.Common_Script_Machine_Id）
 *
 * 公共脚本不挂在某台真实机器下，列表筛选与保存均用该占位 id 表达“全局可见”。
 */
export const COMMON_SCRIPT_MACHINE_ID = 9999999;

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
    No: EnumValue.of(-1, 'machine.noRecord').tagTypeDanger(),
    OnError: EnumValue.of(1, 'machine.onErrorRecord').tagTypeWarning(),
    Yes: EnumValue.of(2, 'machine.record').tagTypeSuccess(),
};

// 计划任务执行记录状态
export const CronJobExecStatusEnum = {
    Error: EnumValue.of(-1, 'machine.cronJobExecStatusEnumFail').tagTypeDanger(),
    Success: EnumValue.of(1, 'machine.cronJobExecStatusEnumSuccess').tagTypeSuccess(),
};
