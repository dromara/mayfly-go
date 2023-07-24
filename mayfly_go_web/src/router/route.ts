import { RouteRecordRaw } from 'vue-router';
import Layout from '@/views/layout/index.vue';

// 定义动态路由
export const dynamicRoutes = [
    {
        path: '/',
        name: '/',
        component: Layout,
        redirect: '/home',
        meta: {
            isKeepAlive: true,
        },
        children: [],
        //     children: [
        //         {
        //             path: '/home',
        //             name: 'home',
        //             component: () => import('@/views/home/index.vue'),
        //             meta: {
        //                 title: '首页',
        //                 // iframe链接
        //                 link: '',
        //                 // 是否在菜单栏显示，默认显示
        //                 isHide: false,
        //                 isKeepAlive: true,
        //                 // tag标签是否不可删除
        //                 isAffix: true,
        //                 // 是否为iframe
        //                 isIframe: false,
        //                 icon: 'el-icon-s-home',
        //             },
        //         },
        //         {
        //             path: '/sys',
        //             name: 'Resource',
        //             redirect: '/sys/resources',
        //             meta: {
        //                 title: '系统管理',
        //                 // 资源code，用于校验用户是否拥有该资源权限
        //                 code: 'sys',
        //                 // isKeepAlive: true,
        //                 icon: 'el-icon-monitor',
        //             },
        //             children: [
        //                 {
        //                     path: 'sys/resources',
        //                     name: 'ResourceList',
        //                     component: () => import('@/views/system/resource'),
        //                     meta: {
        //                         title: '资源管理',
        //                         code: 'resource:list',
        //                         isKeepAlive: true,
        //                         icon: 'el-icon-menu',
        //                     },
        //                 },
        //                 {
        //                     path: 'sys/roles',
        //                     name: 'RoleList',
        //                     component: () => import('@/views/system/role'),
        //                     meta: {
        //                         title: '角色管理',
        //                         code: 'role:list',
        //                         isKeepAlive: true,
        //                         icon: 'el-icon-menu',
        //                     },
        //                 },
        //                 {
        //                     path: 'sys/accounts',
        //                     name: 'ResourceList',
        //                     component: () => import('@/views/system/account'),
        //                     meta: {
        //                         title: '账号管理',
        //                         code: 'account:list',
        //                         isKeepAlive: true,
        //                         icon: 'el-icon-menu',
        //                     },
        //                 },
        //             ],
        //         },
        //         {
        //             path: '/machine',
        //             name: 'Machine',
        //             redirect: '/machine/list',
        //             meta: {
        //                 title: '机器管理',
        //                 // 资源code，用于校验用户是否拥有该资源权限
        //                 code: 'machine',
        //                 // isKeepAlive: true,
        //                 icon: 'el-icon-monitor',
        //             },
        //             children: [
        //                 {
        //                     path: '/list',
        //                     name: 'MachineList',
        //                     component: () => import('@/views/ops/machine'),
        //                     meta: {
        //                         title: '机器列表',
        //                         code: 'machine:list',
        //                         isKeepAlive: true,
        //                         icon: 'el-icon-menu',
        //                     },
        //                 },
        //             ],
        //         },
        //         {
        //             path: '/personal',
        //             name: 'personal',
        //             component: () => import('@/views/personal/index.vue'),
        //             meta: {
        //                 title: '个人中心',
        //                 isKeepAlive: true,
        //                 icon: 'el-icon-user',
        //             },
        //         },
        //     ],
    },
];

// 定义静态路由
export const staticRoutes: Array<RouteRecordRaw> = [
    {
        path: '/login',
        name: 'login',
        component: () => import('@/views/login/index.vue'),
        meta: {
            title: '登录',
        },
    },
    {
        path: '/404',
        name: 'notFound',
        component: () => import('@/views/error/404.vue'),
        meta: {
            title: '找不到此页面',
        },
    },
    {
        path: '/401',
        name: 'noPower',
        component: () => import('@/views/error/401.vue'),
        meta: {
            title: '没有权限',
        },
    },
    {
        path: '/oauth2/callback',
        name: 'oauth2Callback',
        component: () => import('@/views/oauth/Oauth2Callback.vue'),
        meta: {
            title: 'oauth2回调',
        },
    },
    {
        path: '/machine/terminal',
        name: 'machineTerminal',
        component: () => import('@/views/ops/machine/SshTerminalPage.vue'),
        meta: {
            // 将路径 'xxx?name=名字' 里的name字段值替换到title里
            title: '终端 | {name}',
            // 是否根据query对标题名进行参数替换，即最终显示为‘终端_机器名’
            titleRename: true,
        },
    },
];

// 定义404界面
export const pathMatch = {
    path: '/:path(.*)*',
    redirect: '/404',
};
