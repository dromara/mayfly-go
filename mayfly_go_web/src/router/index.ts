import { createRouter, createWebHashHistory, RouteRecordRaw } from 'vue-router';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { getSession, clearSession } from '@/common/utils/storage';
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
const viewsModules: any = import.meta.glob(['../views/**/*.{vue,tsx}', '!../views/layout/**/*.{vue,tsx}']);
const dynamicViewsModules: Record<string, Function> = Object.assign({}, { ...viewsModules });

// 添加静态路由
const router = createRouter({
    history: createWebHashHistory(),
    routes: staticRoutes,
});

// 前端控制路由：初始化方法，防止刷新时丢失
export function initAllFun() {
    NextLoading.start(); // 界面 loading 动画开始执行
    const token = getSession('token'); // 获取浏览器缓存 token 值
    if (!token) {
        // 无 token 停止执行下一步
        return false;
    }
    useUserInfo().setUserInfo({});
    router.addRoute(pathMatch); // 添加404界面
    resetRoute(); // 删除/重置路由
    // 添加动态路由
    setFilterRouteEnd().forEach((route: any) => {
        router.addRoute(route as unknown as RouteRecordRaw);
    });
    // 过滤权限菜单
    useRoutesList().setRoutesList(setFilterMenuFun(dynamicRoutes[0].children, useUserInfo().userInfo.menus));
}

// 后端控制路由：执行路由数据初始化
export async function initBackEndControlRoutesFun() {
    NextLoading.start(); // 界面 loading 动画开始执行
    const token = getSession('token'); // 获取浏览器缓存 token 值
    if (!token) {
        // 无 token 停止执行下一步
        return false;
    }
    useUserInfo().setUserInfo({});
    // 获取路由
    let menuRoute = await getBackEndControlRoutes();
    dynamicRoutes[0].children = backEndRouterConverter(menuRoute); // 处理路由（component）
    // 添加404界面
    router.addRoute(pathMatch);
    resetRoute(); // 删除/重置路由
    // 添加动态路由
    formatTwoStageRoutes(formatFlatteningRoutes(dynamicRoutes)).forEach((route: any) => {
        router.addRoute(route as unknown as RouteRecordRaw);
    });
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

// 后端控制路由，后端返回路由 转换为vue route
export function backEndRouterConverter(routes: any, parentPath: string = '/') {
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
        item.children && backEndRouterConverter(item.children, item.path);
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

// 多级嵌套数组处理成一维数组
export function formatFlatteningRoutes(arr: any) {
    if (arr.length <= 0) return false;
    for (let i = 0; i < arr.length; i++) {
        if (arr[i].children) {
            arr = arr.slice(0, i + 1).concat(arr[i].children, arr.slice(i + 1));
        }
    }
    return arr;
}

// 多级嵌套数组处理后的一维数组，再处理成 `定义动态路由` 的格式
// 只保留二级：也就是二级以上全部处理成只有二级，keep-alive 支持二级缓存
// isKeepAlive 处理 `name` 值，进行缓存。顶级关闭，全部不缓存
export function formatTwoStageRoutes(arr: any) {
    if (arr.length <= 0) return false;
    const newArr: any = [];
    const cacheList: Array<string> = [];
    arr.forEach((v: any) => {
        if (v.path === '/') {
            newArr.push({ component: v.component, name: v.name, path: v.path, redirect: v.redirect, meta: v.meta, children: [] });
        } else {
            newArr[0].children.push({ ...v });
            if (newArr[0].meta.isKeepAlive && v.meta.isKeepAlive) {
                cacheList.push(v.name);
            }
        }
    });
    useKeepALiveNames().setCacheKeepAlive(cacheList);
    return newArr;
}

// 判断路由code 是否包含当前登录用户menus字段中，menus为字符串code数组
export function hasAnth(menus: any, route: any) {
    if (route.meta && route.meta.code) {
        return menus.includes(route.meta.code);
    }
    return true;
}

// 递归过滤有权限的路由
export function setFilterMenuFun(routes: any, menus: any) {
    const menu: any = [];
    routes.forEach((route: any) => {
        const item = { ...route };
        if (hasAnth(menus, item)) {
            if (item.children) {
                item.children = setFilterMenuFun(item.children, menus);
            }
            menu.push(item);
        }
    });
    return menu;
}

// 获取当前用户的权限去比对路由表，用于动态路由的添加
export function setFilterRoute(chil: any) {
    let filterRoute: any = [];
    chil.forEach((route: any) => {
        // 如果路由需要拥有指定code才可访问，则校验该用户菜单是否存在该code
        if (route.meta.code) {
            useUserInfo().userInfo.menus.forEach((m: any) => {
                if (route.meta.code == m) {
                    filterRoute.push({ ...route });
                }
            });
        } else {
            filterRoute.push({ ...route });
        }
    });
    return filterRoute;
}

// 比对后的路由表，进行重新赋值
export function setFilterRouteEnd() {
    let filterRouteEnd: any = formatTwoStageRoutes(formatFlatteningRoutes(dynamicRoutes));
    filterRouteEnd[0].children = setFilterRoute(filterRouteEnd[0].children);
    return filterRouteEnd;
}

// 删除/重置路由
export function resetRoute() {
    useRoutesList().routesList.forEach((route: any) => {
        const { name } = route;
        router.hasRoute(name) && router.removeRoute(name);
    });
}

export async function initRouter() {
    // 初始化方法执行
    const { isRequestRoutes } = useThemeConfig(pinia).themeConfig;
    if (!isRequestRoutes) {
        // 未开启后端控制路由
        initAllFun();
    } else if (isRequestRoutes) {
        // 后端控制路由，isRequestRoutes 为 true，则开启后端控制路由
        await initBackEndControlRoutesFun();
    }
}

let SysWs: any;
let loadRouter = false;

// 路由加载前
router.beforeEach(async (to, from, next) => {
    NProgress.configure({ showSpinner: false });
    if (to.meta.title) NProgress.start();

    // 如果有标题参数，则再原标题后加上参数来区别
    if (to.meta.titleRename) {
        to.meta.title = templateResolve(to.meta.title as string, to.query);
    }

    const token = getSession('token');
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
    NextLoading.done();
});

// 导出路由
export default router;
