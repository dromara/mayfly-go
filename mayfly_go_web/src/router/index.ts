import { createRouter, createWebHashHistory } from 'vue-router';
import NProgress from 'nprogress';
import 'nprogress/nprogress.css';
import { getToken } from '@/common/utils/storage';
import { templateResolve } from '@/common/utils/string';
import { NextLoading } from '@/common/utils/loading';
import { staticRoutes, URL_LOGIN, URL_401, ROUTER_WHITE_LIST, errorRoutes } from './staticRouter';
import syssocket from '@/common/syssocket';
import pinia from '@/store/index';
import { useThemeConfig } from '@/store/themeConfig';
import { useUserInfo } from '@/store/userInfo';
import { useRoutesList } from '@/store/routesList';
import { initBackendRoutes } from './dynamicRouter';

// 添加静态路由
const router = createRouter({
    history: createWebHashHistory(),
    routes: [...staticRoutes, ...errorRoutes],
});

// 前端控制路由：初始化方法，防止刷新时丢失
export function initAllFun() {
    const token = getToken(); // 获取浏览器缓存 token 值
    if (!token) {
        // 无 token 停止执行下一步
        return false;
    }
    useUserInfo().setUserInfo({});
    resetRoute(); // 删除/重置路由
    // router.addRoute(dynamicRoutes[0]);
    // // 过滤权限菜单
    // useRoutesList().setRoutesList(dynamicRoutes[0].children);
}

// 删除/重置路由
export function resetRoute() {
    useRoutesList().routesList?.forEach((route: any) => {
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
            await initBackendRoutes();
        }
    } finally {
        NextLoading.done();
    }
}

// 路由加载前
router.beforeEach(async (to, from, next) => {
    NProgress.configure({ showSpinner: false });
    NProgress.start();

    // 如果有标题参数，则再原标题后加上参数来区别
    if (to.meta.titleRename && to.meta.title) {
        to.meta.title = templateResolve(to.meta.title as string, to.query);
    }

    const token = getToken();

    const toPath = to.path;
    // 判断是访问登陆页，有token就在当前页面，没有token重置路由与用户信息到登陆页
    if (toPath === URL_LOGIN) {
        if (token) {
            return next(from.fullPath);
        }

        resetRoute();
        syssocket.destory();
        return next();
    }

    // 判断访问页面是否在路由白名单地址(静态路由)中，如果存在直接放行
    if (ROUTER_WHITE_LIST.includes(toPath)) {
        return next();
    }

    // 判断是否有token，没有重定向到 login 页面
    if (!token) {
        return next(`${URL_LOGIN}?redirect=${toPath}`);
    }

    // 终端不需要连接系统websocket消息
    if (to.path != '/machine/terminal') {
        syssocket.init();
    }

    // 不存在路由（避免刷新页面找不到路由），则重新初始化路由
    if (useRoutesList().routesList?.length == 0) {
        try {
            // 可能token过期无法获取菜单权限信息等
            await initRouter();
        } catch (e) {
            return next(`${URL_401}?redirect=${toPath}`);
        }
        return next({ path: toPath, query: to.query });
    }

    next();
});

// 路由加载后
router.afterEach(() => {
    NProgress.done();
});

/**
 * @description 路由跳转错误
 * */
router.onError((error) => {
    NProgress.done();
    console.warn('路由错误', error.message);
});

// 导出路由
export default router;
