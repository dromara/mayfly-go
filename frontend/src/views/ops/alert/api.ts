import Api from '@/common/Api';
import type { PageResult } from '@/types/common';
import type {
    AlertRuleVO,
    AlertRuleQuery,
    AlertRuleForm,
    AlertEventVO,
    AlertEventQuery,
    AlertSilenceVO,
    AlertSilenceQuery,
    AlertSilenceForm,
    AlertEscalationVO,
    AlertEscalationQuery,
    AlertEscalationForm,
    AlertInhibitionVO,
    AlertInhibitionQuery,
    AlertInhibitionForm,
    AlertNotifyPolicyVO,
    AlertNotifyPolicyQuery,
    AlertNotifyPolicyForm,
    AlertOverviewData,
    ResourceMetrics,
} from './types';

/** 告警规则 API */
export const alertRuleApi = {
    list: Api.newGet<PageResult<AlertRuleVO>, AlertRuleQuery>('/alert-rules'),
    save: Api.newPost<void, AlertRuleForm>('/alert-rules'),
    changeStatus: Api.newPut<void>('/alert-rules/{id}/{status}'),
    del: Api.newDelete<void>('/alert-rules/{id}'),
    /** 按资源类型获取可用指标元信息，用于条件编辑器的指标下拉与阈值范围约束 */
    metrics: Api.newGet<ResourceMetrics[]>('/alert-rules/metrics'),
};

/** 告警事件 API */
export const alertEventApi = {
    list: Api.newGet<PageResult<AlertEventVO>, AlertEventQuery>('/alert-events'),
    ack: Api.newPut<void>('/alert-events/{id}/ack'),
    close: Api.newPut<void>('/alert-events/{id}/close'),
    del: Api.newDelete<void>('/alert-events/{id}'),
};

/** 静默规则 API */
export const alertSilenceApi = {
    list: Api.newGet<PageResult<AlertSilenceVO>, AlertSilenceQuery>('/alert-silences'),
    save: Api.newPost<void, AlertSilenceForm>('/alert-silences'),
    changeStatus: Api.newPut<void>('/alert-silences/{id}/{status}'),
    del: Api.newDelete<void>('/alert-silences/{id}'),
};

/** 升级策略 API */
export const alertEscalationApi = {
    list: Api.newGet<PageResult<AlertEscalationVO>, AlertEscalationQuery>('/alert-escalations'),
    save: Api.newPost<void, AlertEscalationForm>('/alert-escalations'),
    changeStatus: Api.newPut<void>('/alert-escalations/{id}/{status}'),
    del: Api.newDelete<void>('/alert-escalations/{id}'),
};

/** 抑制规则 API */
export const alertInhibitionApi = {
    list: Api.newGet<PageResult<AlertInhibitionVO>, AlertInhibitionQuery>('/alert-inhibitions'),
    save: Api.newPost<void, AlertInhibitionForm>('/alert-inhibitions'),
    changeStatus: Api.newPut<void>('/alert-inhibitions/{id}/{status}'),
    del: Api.newDelete<void>('/alert-inhibitions/{id}'),
};

/** 通知策略 API */
export const alertNotifyPolicyApi = {
    list: Api.newGet<PageResult<AlertNotifyPolicyVO>, AlertNotifyPolicyQuery>('/alert-notify-policies'),
    save: Api.newPost<void, AlertNotifyPolicyForm>('/alert-notify-policies'),
    changeStatus: Api.newPut<void>('/alert-notify-policies/{id}/{status}'),
    del: Api.newDelete<void>('/alert-notify-policies/{id}'),
};

/** 告警概览 API */
export const alertOverviewApi = {
    overview: Api.newGet<AlertOverviewData>('/alert-events/overview'),
};
