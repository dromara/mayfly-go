/**
 * Flow 模块类型定义
 * 对应后端: flow/domain/entity/*
 */
import type { ResourceTag } from '@/views/ops/tag/types';
import type { RuleNode, TriggerPolicy } from '@/components/policy-builder';

export type {
    PolicySchema,
    PolicyScenario,
    PolicyCheck,
    PolicyCheckParam,
    PolicyField,
    PolicyOperator,
    CheckConfig,
    CustomCondition,
    RuleNode,
    Severity,
    Decision,
    Finding,
    SimulateResult,
    TriggerPolicy,
} from '@/components/policy-builder';

// ==================== Entity ====================

/** 流程定义实体 (对应 entity.Procdef) */
export interface Procdef {
    id: number;
    name: string;
    defKey: string;
    flowDef: string;
    status: number;
    /** 触发策略，决定哪些资源操作需要走该审批流程 */
    triggerPolicy?: TriggerPolicy;
    remark?: string;
    createTime: string;
    creator: string;
}

/**
 * 流程定义列表行 / 详情 (对应 vo.Procdef = entity.Procdef + tagentity.RelateTags)
 *
 * 仅 GET /flow/procdefs（列表）与 /flow/procdefs/detail/{id} 经 FillTagInfo 填充 tags；
 * GET /flow/procdefs/{resourceType}/{resourceCode} 返回裸实体，仍用 Procdef。
 */
export type ProcdefVO = Procdef & { tags?: ResourceTag[] };

/** 流程定义查询参数 (根据资源) */
export interface ProcdefResourceParam {
    resourceType: string;
    resourceCode: string;
}

/** 流程实例（对应接口返回的 vo.ProcinstVO 与列表行） */
export interface Procinst {
    id: number;
    procdefId: number;
    procdefName: string;
    flowDef: string;
    /** 流程变量（列表与详情接口均返回） */
    vars: Record<string, unknown>;
    bizType: string;
    bizKey: string;
    bizForm: string;
    bizStatus: number;
    bizHandleRes: string;
    /** 当前任务 key */
    taskKey?: string;
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

/** 流程连线。跳转条件存于 extra.condition，与其它节点/连线属性同一取法 */
export interface FlowEdge {
    name: string;
    key: string;
    sourceNodeKey: string;
    targetNodeKey: string;
    extra?: { condition?: RuleNode | string | null; [key: string]: unknown };
}

/** 可复用条件组（对应 entity.RuleSegment） */
export interface RuleSegment {
    id: number;
    /** 引用标识：条件树里的 segment 节点按它引用，创建后不可修改 */
    ref: string;
    name: string;
    bizType: string;
    remark?: string;
    ruleNode?: RuleNode;
    createTime: string;
    creator: string;
}

/** 触发策略变更记录（对应 entity.ProcdefPolicyHis） */
export interface ProcdefPolicyHis {
    id: number;
    procdefId: number;
    procdefName: string;
    /** 1 新建 2 修改 3 删除 */
    action: number;
    before?: TriggerPolicy;
    after?: TriggerPolicy;
    createTime: string;
    creator: string;
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

/**
 * 提单抽屉的预填内容。
 *
 * 由拦截处（SQL 控制台、Redis 命令控制台）在打开抽屉时直接传入：
 * 这些内容是在点击那一刻才算出来的，走 props 要等父组件渲染完才传得进来
 */
export interface TicketPrefill {
    /** 业务场景，取值为后端注册的 bizType */
    bizType?: string;
    /** 预填的业务表单（库、SQL、命令、资源编码等） */
    bizForm?: Record<string, unknown>;
    /** 预填的工单备注 */
    remark?: string;
}
