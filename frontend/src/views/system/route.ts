export default {
    AccountList: () => import('@/views/system/account/AccountList.vue'),
    ResourceList: () => import('@/views/system/resource/ResourceList.vue'),
    RoleList: () => import('@/views/system/role/RoleList.vue'),
    ConfigList: () => import('@/views/system/config/ConfigList.vue'),
    SyslogList: () => import('@/views/system/syslog/SyslogList.vue'),
};
