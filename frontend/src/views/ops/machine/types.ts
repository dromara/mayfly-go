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
    /** 后端 model.ExtraData 透传的自由 map；机器侧实际仅携带 ciphers/keyExchanges 两项字符串 */
    extra?: Record<string, string>;
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
    timeoutSeconds: number;
    retryTimes: number;
    notifyType: number;
    notifyTmplCode: string;
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
    timeoutSeconds?: number;
    retryTimes?: number;
    notifyType?: number;
    notifyTmplCode?: string;
    remark?: string;
    codePaths?: string[];
}

/** 机器命令配置表单 (对应 form.MachineCmdConfForm) */
export interface MachineCmdConfForm {
    id?: number;
    name: string;
    cmds: string[];
    status?: number;
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
    /** 后端 model.ExtraData 透传的自由 map；机器侧实际仅携带 ciphers/keyExchanges 两项字符串 */
    extra?: Record<string, string>;
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

/** 机器文件信息 (对应 vo.MachineFileInfo，只描述后端返回的事实) */
export interface MachineFileInfo {
    name: string;
    path: string;
    size: number;
    type: string;
    mode: string;
    modTime: string;
    uid: number;
    gid: number;
}

/**
 * 文件列表的行视图模型：在后端事实之上叠加渲染与交互状态。
 *
 * 字段全部必填，由 `lsFile` 一次性构造，因此消费方不需要处理「没初始化」的分支；
 * 把它们与 VO 字段分开的意义在于：后端契约变动不会连带渲染态，渲染态也不会被误当成接口字段。
 */
export interface FileRowVM extends MachineFileInfo {
    /** 是否目录（由 type 推导，模板不再比较字面量） */
    isFolder: boolean;
    /** 文件图标名 */
    icon: string;
    /** 处于行内重命名编辑态 */
    nameEdit: boolean;
    /** 目录大小读数，空串表示尚未计算 */
    dirSize: string;
    loadingDirSize: boolean;
    /** stat 原文读数，空串表示尚未拉取 */
    stat: string;
    loadingStat: boolean;
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
    remark: string;
    createTime: string;
    creator: string;
}

/** 主机公钥信任记录 (对应 entity.MachineHostKey)，由连接时自动采集（TOFU） */
export interface MachineHostKeyVO {
    id: number;
    /** 信任键：连接改写前的原始目标地址 ip:port */
    hostAddr: string;
    keyType: string;
    fingerprint: string;
    remark: string;
    createTime: string;
    creator: string;
}

/** 单台机器的批量命令执行结果（单台失败/超时不影响他台） */
export interface BatchCmdResult {
    machineId: number;
    name: string;
    ip: string;
    port: number;
    authCertName: string;
    username: string;

    success: boolean;
    output: string;
    error: string;
    /** 命令策略「仅提醒」命中提示（不是失败） */
    policyNotice: string;
    costMs: number;
    timeout: boolean;
}

/** 单台机器的批量文件分发结果 (对应 application.BatchFileResult) */
export interface BatchFileResult {
    machineId: number;
    name: string;
    ip: string;
    port: number;
    authCertName: string;
    success: boolean;
    /** 分发字节数 */
    bytes: number;
    error: string;
    costMs: number;
}

/** 机器指标历史采样点 (对应 entity.MachineMetric) */
export interface MachineMetric {
    id: number;
    machineId: number;
    /** 采集时间（后端 time.Time 序列化为 RFC3339 字符串） */
    collectTime: string;
    cpuUsage: number;
    memUsage: number;
    diskUsage: number;
    load1: number;
    load5: number;
    load10: number;
    /** 网络累计接收字节（速率由前端相邻点差分） */
    netRx: number;
    /** 网络累计发送字节 */
    netTx: number;
    /** 1在线 0离线 */
    status: number;
}

/** 单条阈值命中明细（由告警侧按已启用的告警规则判定后回填） */
export interface MachineHealthHit {
    ruleId: number;
    ruleName: string;
    /** cpu_rate / mem_rate / disk_usage / status */
    metric: string;
    /** gt / gte / lt / lte / eq / neq */
    compare: string;
    threshold: number;
    current: number;
    /** 命中规则优先级：0 P0 .. 3 P3 */
    priority: number;
}

/** 全机器健康总览单项 (对应 application.MachineHealth) */
export interface MachineHealth {
    machineId: number;
    name: string;
    ip: string;
    port: number;
    code: string;
    /** 1在线 0离线 */
    status: number;
    cpuUsage: number;
    memUsage: number;
    diskUsage: number;
    collectTime: string | null;
    /** 命中的最差规则优先级，-1 表示无命中 */
    priority: number;
    /** false 表示有规则覆盖但取不到当前值，本次未能判定（不等于正常） */
    triaged: boolean;
    hits: MachineHealthHit[] | null;
}

/** 磁盘目录占用节点 (对应 application.DiskNode) */
export interface DiskNode {
    path: string;
    /** 字节 */
    size: number;
}

/** 磁盘分析结果 (对应 application.DiskAnalysisResult) */
export interface DiskAnalysisResult {
    path: string;
    depth: number;
    nodes: DiskNode[];
}

/** 机器进程行（`ps` 输出按空白切割后映射而得，字段与表格列一一对应；数值均已在解析时格式化为展示字符串） */
export interface MachineProcess {
    /** ps USER 列 */
    user: string;
    /** ps PID 列 */
    pid: string;
    /** ps %CPU 列 */
    cpu: string;
    /** ps %MEM 列 */
    mem: string;
    /** ps VSZ 列（已折算为 MB 字符串） */
    vsz: string;
    /** ps RSS 列（已折算为 MB 字符串） */
    rss: string;
    /** ps STAT 列 */
    stat: string;
    /** ps START 列 */
    start: string;
    /** ps TIME 列 */
    time: string;
    /** ps COMMAND 列（由第 10 列起重新拼接，兼容命令自带空白） */
    command: string;
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
