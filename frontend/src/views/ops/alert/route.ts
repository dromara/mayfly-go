export default {
    AlertOverview: () => import('./overview/AlertOverview.vue'),
    AlertRuleList: () => import('./rule/AlertRuleList.vue'),
    AlertEventList: () => import('./event/AlertEventList.vue'),
    AlertSilenceList: () => import('./silence/AlertSilenceList.vue'),
    AlertEscalationList: () => import('./escalation/AlertEscalationList.vue'),
    AlertInhibitionList: () => import('./inhibition/AlertInhibitionList.vue'),
    AlertNotifyPolicyList: () => import('./notify-policy/AlertNotifyPolicyList.vue'),
};
