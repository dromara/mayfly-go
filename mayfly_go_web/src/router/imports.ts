import RouterParent from '@/views/layout/routerView/parent.vue';

export const imports = {
    'RouterParent': RouterParent,
    "Home": () => import('@/views/home/index.vue'),
    'Personal': () => import('@/views/personal/index.vue'),
    // machine
    "MachineList": () => import('@/views/ops/machine'),
    // sys
    "ResourceList": () => import('@/views/system/resource'),
    "RoleList": () => import('@/views/system/role'),
    "AccountList": () => import('@/views/system/account'),
    // project
    "ProjectList": () => import('@/views/ops/project/ProjectList.vue'),
    // db
    "DbList": () => import('@/views/ops/db/DbList.vue'),
    "SqlExec": () => import('@/views/ops/db'),
    // redis
    "RedisList": () => import('@/views/ops/redis'),
    "DataOperation": () => import('@/views/ops/redis/DataOperation.vue'),
}