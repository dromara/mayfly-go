/**
 * Machine 模块类型定义
 * 对应后端: machine/domain/entity/* + machine/api/form/* + machine/api/vo/*
 */
import type { AuthCertInfo, AuthCerts, PageParam, RelateTags } from '@/types/common';

// ==================== Entity ====================

/** 机器实体 (对应 entity.Machine) */
export interface Machine {
    id: number;
    code: string;
    name: string;
    protocol: number;
    ip: string;
    port: number;
    status: number;
    remark: string;
    sshTunnelMachineId: number;
    enableRecorder: number;
    extra?: Record<string, unknown>;
    createTime: string;
    creator: string;
}

/** 机器脚本实体 (对应 entity.MachineScript) */
export interface MachineScript {
    id: number;
    name: string;
    machineId: number;
    type: number;
    category: string;
    description: string;
    params: string;
    script: string;
    createTime: string;
    creator: string;
}

/** 机器定时任务实体 (对应 entity.MachineCronJob) */
export interface MachineCronJob extends RelateTags {
    id: number;
    name: string;
    key: string;
    cron: string;
    script: string;
    status: number;
    remark: string;
    lastExecTime?: string;
    saveExecResType: number;
    running?: boolean;
    createTime: string;
    creator: string;
}

/** 机器定时任务执行记录 (对应 entity.MachineCronJobExec) */
export interface MachineCronJobExec {
    id: number;
    cronJobId: number;
    machineCode: string;
    status: number;
    res: string;
    execTime: string;
}

/** 机器文件配置实体 (对应 entity.MachineFile) */
export interface MachineFile {
    id: number;
    name: string;
    machineId: number;
    type: number;
    path: string;
    createTime: string;
    creator: string;
}

/** 机器终端操作记录 (对应 entity.MachineTermOp) */
export interface MachineTermOp {
    id: number;
    machineId: number;
    username: string;
    fileKey: string;
    execCmds: string;
    createTime: string;
    creatorId: number;
    creator: string;
    endTime?: string;
}

/** 机器命令配置实体 (对应 entity.MachineCmdConf) */
export interface MachineCmdConf {
    id: number;
    name: string;
    cmds: string[];
    status: number;
    stratege: string;
    remark: string;
    createTime: string;
    creator: string;
}

// ==================== Form ====================

/** 机器列表查询参数 */
export interface MachineListParam extends PageParam {
    protocol?: number;
    tagPath?: string;
    code?: string;
}

/** 机器表单 (对应 form.MachineForm) */
export interface MachineForm {
    id?: number | null;
    code?: string;
    tagPath?: string;
    protocol: number;
    name?: string | null;
    ip?: string | null;
    port: number;
    tagCodePaths: string[];
    authCerts: MachineAuthCert[];
    remark?: string;
    sshTunnelMachineId?: number | null;
    enableRecorder?: number;
    extra: Record<string, string>;
}

/** 机器授权凭证表单 */
export interface MachineAuthCert {
    name?: string;
    username: string;
    ciphertext?: string;
    ciphertextType: number;
    type: number;
    extra?: Record<string, string>;
}

/** 机器运行命令表单 (对应 form.MachineRunForm) */
export interface MachineRunForm {
    machineId: number;
    cmd: string;
}

/** 机器脚本表单 (对应 form.MachineScriptForm) */
export interface MachineScriptForm {
    id?: number | null;
    name: string;
    machineId: number;
    type?: number | null;
    category?: string;
    description: string;
    params?: string;
    script: string;
}

/** 机器定时任务表单 (对应 form.MachineCronJobForm) */
export interface MachineCronJobForm {
    id?: number | null;
    name: string;
    cron: string;
    script: string;
    status: number;
    saveExecResType: number;
    remark?: string;
    codePaths?: string[];
}

/** 机器命令配置表单 (对应 form.MachineCmdConfForm) */
export interface MachineCmdConfForm {
    id?: number;
    name: string;
    cmds: string[];
    status?: number;
    stratege?: string;
    remark?: string;
    codePaths?: string[];
}

// ==================== VO ====================

/** 机器视图对象 (对应 vo.MachineVO) */
export interface MachineVO extends AuthCerts {
    id: number;
    code: string;
    name: string;
    protocol: number;
    ip: string;
    port: number;
    status?: number;
    sshTunnelMachineId: number;
    createTime?: string;
    creator?: string;
    creatorId?: number;
    updateTime?: string;
    modifier?: string;
    modifierId?: number;
    remark?: string;
    enableRecorder: number;
    stat?: MachineListStat;
    selectAuthCert?: MachineAuthCert;
    extra?: Record<string, unknown>;
}

/** 简单机器视图 (对应 vo.SimpleMachineVO) */
export interface SimpleMachineVO {
    id: number;
    code: string;
    name: string;
    ip: string;
    port: number;
    remark?: string;
}

/** 机器脚本视图 (对应 vo.MachineScriptVO) */
export interface MachineScriptVO {
    id?: number;
    name?: string;
    script?: string;
    type?: number;
    category: string;
    description?: string;
    params?: string;
    machineId?: number;
}

/** 机器定时任务视图 (对应 vo.MachineCronJobVO) */
export interface MachineCronJobVO extends RelateTags {
    id: number;
    key: string;
    name: string;
    cron: string;
    script: string;
    status: number;
    saveExecResType: number;
    remark: string;
    running: boolean;
}

/** 机器文件配置视图 (对应 vo.MachineFileVO) */
export interface MachineFileVO {
    id: number;
    name: string;
    path: string;
    type: number;
    machineId: number;
}

/** 机器文件信息 (对应 vo.MachineFileInfo) */
export interface MachineFileInfo {
    name: string;
    path: string;
    size: number;
    type: string;
    mode: string;
    modTime: string;
    uid: number;
    gid: number;
    // UI-only extended fields
    isFolder?: boolean;
    icon?: string;
    nameEdit?: boolean;
    dirSize?: string;
    loadingDirSize?: boolean;
    stat?: string;
    loadingStat?: boolean;
}

/** 机器用户信息 */
export interface MachineUserInfo {
    uid: number;
    uname: string;
}

/** 机器用户组信息 */
export interface MachineGroupInfo {
    gid: number;
    gname: string;
}

/** 机器命令配置视图 (对应 vo.MachineCmdConfVO) */
export interface MachineCmdConfVO extends RelateTags {
    id: number;
    name: string;
    cmds: string[];
    status: number;
    stratege: string;
    remark: string;
    createTime: string;
    creator: string;
}

/** 机器进程信息 */
export interface MachineProcess {
    pid: number;
    name: string;
    cmdline: string;
    username: string;
    status: string;
    cpuPercent: number;
    memPercent: number;
    memRss: number;
    createTime: string;
}

/** 机器列表统计信息（后端机器列表 API 返回的 stat 字段，见 machine/api/machine.go Machines） */
export interface MachineListStat {
    cpuIdle: number;
    memAvailable: number;
    memTotal: number;
    fsInfos: MachineFSInfo[];
}

/** 机器统计信息 (对应 mcm.Stats) */
export interface MachineStats {
    uptime: string;
    hostname: string;
    load1: string;
    load5: string;
    load10: string;
    runningProcs: string;
    totalProcs: string;
    memInfo: MachineMemInfo;
    fSInfos: MachineFSInfo[];
    netIntf: Record<string, MachineNetIntfInfo>;
    cpu: MachineCpuInfo;
}

/** 文件系统信息 (对应 mcm.FSInfo) */
export interface MachineFSInfo {
    mountPoint: string;
    used: number;
    free: number;
}

/** 网卡接口信息 (对应 mcm.NetIntfInfo) */
export interface MachineNetIntfInfo {
    ipv4: string;
    ipv6: string;
    rx: number;
    tx: number;
}

/** 内存信息 (对应 mcm.MemInfo) */
export interface MachineMemInfo {
    total: number;
    free: number;
    buffers: number;
    available: number;
    cached: number;
    swapTotal: number;
    swapFree: number;
}

/** CPU 信息 (对应 mcm.CPUInfo) */
export interface MachineCpuInfo {
    user: number;
    nice: number;
    system: number;
    idle: number;
    iowait: number;
    irq: number;
    softIrq: number;
    steal: number;
    guest: number;
}
