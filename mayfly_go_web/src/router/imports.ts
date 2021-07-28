import RouterParent from '@/views/layout/routerView/parent.vue';

export const imports = {
    'RouterParent': RouterParent,
    "Home": () => import('@/views/home/index.vue'),
    'Personal': () => import('@/views/personal/index.vue'),
    "MachineList": () => import('@/views/ops/machine'),
    "ResourceList": () => import('@/views/system/resource'),
    "RoleList": () => import('@/views/system/role'),
    "AccountList": () => import('@/views/system/account'),
    "ProjectList": () => import('@/views/ops/project/ProjectList.vue'),
    "DbList": () => import('@/views/ops/db/DbList.vue'),
    "SqlExec": () => import('@/views/ops/db'),
    "RedisList": () => import('@/views/ops/redis'),
    "DataOperation": () => import('@/views/ops/redis/DataOperation.vue'),
}