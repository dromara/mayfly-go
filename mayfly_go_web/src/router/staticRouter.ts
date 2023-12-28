import { RouteRecordRaw } from 'vue-router';

export const URL_HOME: string = '/home';

// 登录页地址（默认）
export const URL_LOGIN: string = '/login';

export const URL_401: string = '/401';

export const URL_404: string = '/404';

export const LAYOUT_ROUTE_NAME: string = 'layout';

// 路由白名单地址（本地存在的路由 staticRouter.ts 中）
export const ROUTER_WHITE_LIST: string[] = [URL_404, URL_401, '/oauth2/callback'];

// 静态路由
export const staticRoutes: Array<RouteRecordRaw> = [
    {
        path: '/',
        redirect: URL_HOME,
    },
    {
        path: URL_LOGIN,
        name: 'login',
        component: () => import('@/views/login/index.vue'),
        meta: {
            title: '登录',
        },
    },
    {
        path: '/layout',
        name: LAYOUT_ROUTE_NAME,
        component: () => import('@/layout/index.vue'),
        redirect: URL_HOME,
        children: [],
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

// 错误页面路由
export const errorRoutes: Array<RouteRecordRaw> = [
    {
        path: URL_404,
        name: 'notFound',
        component: () => import('@/views/error/404.vue'),
        meta: {
            title: '找不到此页面',
        },
    },
    {
        path: URL_401,
        name: 'noPower',
        component: () => import('@/views/error/401.vue'),
        meta: {
            title: '没有权限',
        },
    },
    // Resolve refresh page, route warnings
    {
        path: '/:pathMatch(.*)*',
        component: () => import('@/views/error/404.vue'),
    },
];
