import RouterParent from '@/views/layout/routerView/parent.vue';

export const imports = {
    'RouterParent': RouterParent,
    "Home": () => import('@/views/home/index.vue'),
    'Personal': () => import('@/views/personal/index.vue'),
    "MachineList": () => import('@/views/ops/machine'),
    "ResourceList": () => import('@/views/system/resource'),
    "RoleList": () => import('@/views/system/role'),
    "AccountList": () => import('@/views/system/account'),
    "SelectData": () => import('@/views/ops/db'),
}