/**
 * Flow 模块类型定义
 * 对应后端: flow/domain/entity/*
 */

// ==================== Entity ====================

/** 流程定义实体 (对应 entity.Procdef) */
export interface Procdef {
    id: number;
    name: string;
    defKey: string;
    flowDef: string;
    status: number;
    condition?: string;
    remark?: string;
    createTime: string;
    creator: string;
}

/** 流程定义查询参数 (根据资源) */
export interface ProcdefResourceParam {
    resourceType: string;
    resourceCode: string;
}

/** 流程实例实体 (对应 entity.Procinst) */
export interface Procinst {
    id: number;
    procdefId: number;
    procdefName: string;
    flowDef: string;
    vars: Record<string, unknown>;
    bizType: string;
    bizKey: string;
    bizForm: string;
    bizStatus: number;
    bizHandleRes: string;
    status: number;
    remark: string;
    endTime?: string;
    duration: number;
    createTime: string;
    creator: string;
    procinstTasks?: ProcinstTask[];
}

/** 流程任务实体 (对应 entity.ProcinstTask) */
export interface ProcinstTask {
    id: number;
    procinstId: number;
    executionId: number;
    nodeKey: string;
    nodeName: string;
    nodeType: string;
    vars: Record<string, unknown>;
    status: number;
    remark: string;
    endTime?: string;
    duration: number;
    handler?: string;
    createTime: string;
    creator: string;
    extra?: Record<string, unknown>;
}

/** 流程操作历史实体 (对应 entity.HisProcinstOp) */
export interface HisProcinstOp {
    id: number;
    procinstId: number;
    executionId: number;
    nodeKey: string;
    nodeName: string;
    nodeType: string;
    state: number;
    remark: string;
    endTime?: string;
    duration: number;
    createTime: string;
    creator: string;
    extra?: Record<string, unknown>;
}

// ==================== FlowDef ====================

/** 流程定义内容 */
export interface FlowDef {
    nodes: FlowNode[];
    edges: FlowEdge[];
}

/** 流程节点 */
export interface FlowNode {
    name: string;
    key: string;
    type: string;
    properties?: {
        tasks?: ProcinstTask[];
        opLog?: HisProcinstOp[];
        candidates?: Array<{ type: number | string; id?: number; name?: string }>;
        [key: string]: unknown;
    };
    extra?: Record<string, unknown>;
}

/** 流程连线 */
export interface FlowEdge {
    name: string;
    key: string;
    sourceNodeKey: string;
    targetNodeKey: string;
    condition: string;
    extra?: Record<string, unknown>;
}

// ==================== VO ====================

/** 流程定义详情 (含 flowDef 解析) */
export interface ProcdefDetail extends Procdef {
    flowDefObj?: FlowDef;
}

/** 流程实例详情 (含任务和操作历史) */
export interface ProcinstDetail extends Procinst {
    tasks?: ProcinstTask[];
    hisOps?: HisProcinstOp[];
    flowDefObj?: FlowDef;
}

// ==================== Flow Node ====================

/** 节点操作日志 (用于流程设计器节点样式) */
export interface NodeOpLog {
    state: number;
    extra?: {
        approvalResult?: number;
        [key: string]: unknown;
    };
}

/** 节点表单属性 (PropSetting modelValue) */
export interface NodeFormProps {
    [key: string]: unknown;
}

/** 流程业务表单 (bizForm) */
export interface FlowBizForm {
    id?: number;
    [key: string]: unknown;
}

/** 流程实例启动表单 */
export interface ProcInstStartForm {
    bizType: number | string;
    procdefId: number;
    status: number | null;
    remark: string;
    bizKey: string;
    bizForm: FlowBizForm;
}
