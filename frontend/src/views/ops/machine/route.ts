export default {
    MachineList: () => import('@/views/ops/machine/MachineList.vue'),
    MachineOp: () => import('@/views/ops/machine/MachineOp.vue'),
    CronJobList: () => import('@/views/ops/machine/cronjob/CronJobList.vue'),
    SecurityConfList: () => import('@/views/ops/machine/security/SecurityConfList.vue'),
};
