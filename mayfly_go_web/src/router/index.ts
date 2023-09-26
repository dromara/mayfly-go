import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { clearSession, getToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import { NextLoading } from '@/common/utils/loading';
import { dynamicRoutes, staticRoutes, pathMatch } from './route';
import openApi from '@/common/openApi';
import sockets from '@/common/sockets';
import pinia from '@/store/index';
import { useThemeConfig } from '@/store/themeConfig';
import { useUserInfo } from '@/store/userInfo';
import { useRoutesList } from '@/store/routesList';
import { useKeepALiveNames } from '@/store/keepAliveNames';

/**
 * 获取目录下的 .vue、.tsx 全部文件
 * @method import.meta.glob
 * @link 参考：https://cn.vitejs.dev/guide/features.html#json
 */
const viewsModules: any = import.meta.glob(['../views/**/*.{vue,tsx}']);
const dynamicViewsModules: Record<string, Function> = Object.assign({}, { ...viewsModules });

// 添加静态路由
const router = createRouter({
    history: createWebHashHistory(),
    routes: staticRoutes,
});

// 前端控制路由：初始化方法，防止刷新时丢失
export function initAllFun() {
    const token = getToken(); // 获取浏览器缓存 token 值
    if (!token) {
        // 无 token 停止执行下一步
        return false;
    }
    useUserInfo().setUserInfo({});
    router.addRoute(pathMatch); // 添加404界面
    resetRoute(); // 删除/重置路由
    router.addRoute(dynamicRoutes[0]);

    // 过滤权限菜单
    useRoutesList().setRoutesList(dynamicRoutes[0].children);
}

// 后端控制路由：执行路由数据初始化
export async function initBackEndControlRoutesFun() {
    const token = getToken(); // 获取浏览器缓存 token 值
    if (!token) {
        // 无 token 停止执行下一步
        return false;
    }
    useUserInfo().setUserInfo({});
    // 获取路由
    let menuRoute = await getBackEndControlRoutes();

    const cacheList: Array<string> = [];
    // 处理路由（component）
    dynamicRoutes[0].children = backEndRouterConverter(menuRoute, (router: any) => {
        // 可能为false时不存在isKeepAlive属性
        if (!router.meta.isKeepAlive) {
            router.meta.isKeepAlive = false;
        }
        if (router.meta.isKeepAlive) {
            cacheList.push(router.name);
        }
    });
    useKeepALiveNames().setCacheKeepAlive(cacheList);

    // 添加404界面
    router.addRoute(pathMatch);
    resetRoute(); // 删除/重置路由
    router.addRoute(dynamicRoutes[0] as unknown as RouteRecordRaw);

    useRoutesList().setRoutesList(dynamicRoutes[0].children);
}

// 后端控制路由，isRequestRoutes 为 true，则开启后端控制路由
export async function getBackEndControlRoutes() {
    try {
        const menuAndPermission = await openApi.getPermissions();
        // 赋值权限码，用于控制按钮等
        useUserInfo().userInfo.permissions = menuAndPermission.permissions;
        return menuAndPermission.menus;
    } catch (e: any) {
        console.error(e);
        return [];
    }
}

type RouterConvCallbackFunc = (router: any) => void;

// 后端控制路由，后端返回路由 转换为vue route
export function backEndRouterConverter(routes: any, callbackFunc: RouterConvCallbackFunc = null as any, parentPath: string = '/') {
    if (!routes) return;
    return routes.map((item: any) => {
        if (!item.meta) {
            return item;
        }
        // 将json字符串的meta转为对象
        item.meta = JSON.parse(item.meta);
        // 将meta.comoponet 解析为route.component
        if (item.meta.component) {
            item.component = dynamicImport(dynamicViewsModules, item.meta.component);
            delete item.meta['component'];
        }
        // route.path == resource.code
        let path = item.code;
        // 如果不是以 / 开头，则路径需要拼接父路径
        if (!path.startsWith('/')) {
            path = parentPath + '/' + path;
        }
        item.path = path;
        delete item['code'];

        // route.meta.title == resource.name
        item.meta.title = item.name;
        delete item['name'];

        // route.name == resource.meta.routeName
        item.name = item.meta.routeName;
        delete item.meta['routeName'];

        // route.redirect == resource.meta.redirect
        if (item.meta.redirect) {
            item.redirect = item.meta.redirect;
            delete item.meta['redirect'];
        }
        // 存在回调，则执行回调
        callbackFunc && callbackFunc(item);
        item.children && backEndRouterConverter(item.children, callbackFunc, item.path);
        return item;
    });
}

/**
 * 后端路由 component 转换函数
 * @param dynamicViewsModules 获取目录下的 .vue、.tsx 全部文件
 * @param component 当前要处理项 component
 * @returns 返回处理成函数后的 component
 */
export function dynamicImport(dynamicViewsModules: Record<string, Function>, component: string) {
    const keys = Object.keys(dynamicViewsModules);
    const matchKeys = keys.filter((key) => {
        const k = key.replace(/..\/views|../, '');
        return k.startsWith(`${component}`) || k.startsWith(`/${component}`);
    });
    if (matchKeys?.length === 1) {
        const matchKey = matchKeys[0];
        return dynamicViewsModules[matchKey];
    }
    if (matchKeys?.length > 1) {
        return false;
    }
}

// 删除/重置路由
export function resetRoute() {
    useRoutesList().routesList.forEach((route: any) => {
        const { name } = route;
        router.hasRoute(name) && router.removeRoute(name);
    });
}

export async function initRouter() {
    NextLoading.start(); // 界面 loading 动画开始执行
    try {
        // 初始化方法执行
        const { isRequestRoutes } = useThemeConfig(pinia).themeConfig;
        if (!isRequestRoutes) {
            // 未开启后端控制路由
            initAllFun();
        } else if (isRequestRoutes) {
            // 后端控制路由，isRequestRoutes 为 true，则开启后端控制路由
            await initBackEndControlRoutesFun();
        }
    } finally {
        NextLoading.done();
    }
}

let SysWs: any;
let loadRouter = false;

// 路由加载前
router.beforeEach(async (to, from, next) => {
    NProgress.configure({ showSpinner: false });
    if (to.meta.title) NProgress.start();

    // 如果有标题参数，则再原标题后加上参数来区别
    if (to.meta.titleRename && to.meta.title) {
        to.meta.title = templateResolve(to.meta.title as string, to.query);
    }

    const token = getToken();
    if ((to.path === '/login' || to.path == '/oauth2/callback') && !token) {
        next();
        NProgress.done();
        return;
    }
    if (!token) {
        next(`/login?redirect=${to.path}`);
        clearSession();
        resetRoute();
        NProgress.done();

        if (SysWs) {
            SysWs.close();
            SysWs = null;
        }
        return;
    }
    if (token && to.path === '/login') {
        next('/');
        NProgress.done();
        return;
    }

    // 终端不需要连接系统websocket消息
    if (!SysWs && to.path != '/machine/terminal') {
        SysWs = sockets.sysMsgSocket();
    }
    // 不存在路由（避免刷新页面找不到路由）并且未加载过（避免token过期，导致获取权限接口报权限不足，无限获取），则重新初始化路由
    if (useRoutesList().routesList.length == 0 && !loadRouter) {
        await initRouter();
        loadRouter = true;
        next({ path: to.path, query: to.query });
    } else {
        next();
    }
});

// 路由加载后
router.afterEach(() => {
    NProgress.done();
});

// 导出路由
export default router;
