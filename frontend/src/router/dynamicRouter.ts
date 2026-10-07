import 'nprogress/nprogress.css';
import { clearSession, getToken } from '@/common/utils/storage';
import openApi from '@/common/openApi';
import { useUserInfo } from '@/store/userInfo';
import { useRoutesList } from '@/store/routesList';
import { useKeepALiveNames } from '@/store/keepAliveNames';
import router from '.';
import { RouteRecordRaw } from 'vue-router';
import { LAYOUT_ROUTE_NAME } from './staticRouter';
import { MenuRouteProblem, backEndRouterConverter, normalizeMenuRoutes, reportMenuRouteProblems } from './menuRouteResolver';

/**
 * 获取目录下的 route.ts 全部文件
 * @method import.meta.glob
 * @link 参考：https://cn.vitejs.dev/guide/features.html#json
 */
const routeModules: Record<string, any> = import.meta.glob(['../views/**/route.{ts,js}'], { eager: true });

// 后端控制路由：执行路由数据初始化
export async function initBackendRoutes() {
    // 合并所有模块路由
    const allModuleRoutes = Object.values(routeModules).reduce((acc: any, module: any) => {
        return { ...acc, ...module.default };
    }, {});

    const token = getToken();
    if (!token) {
        return false;
    }

    useUserInfo().setUserInfo({} as any);

    try {
        // 获取路由和权限
        const menuAndPermission = await openApi.getPermissions();
        useUserInfo().userInfo.permissions = menuAndPermission.permissions;
        const menuRoute = menuAndPermission.menus;

        const cacheList: string[] = [];

        // 处理路由（component）
        const routes = backEndRouterConverter(allModuleRoutes, menuRoute, (router: any) => {
            // 确保 isKeepAlive 属性存在
            router.meta.isKeepAlive = router.meta.isKeepAlive ?? false;
            if (router.meta.isKeepAlive) {
                cacheList.push(router.name as string);
            }
        });

        // 菜单与前端路由表是两套字符串标识，对不上时不会报错只会空白，这里统一兜底并显式报出脱节项
        const problems: MenuRouteProblem[] = [];
        normalizeMenuRoutes(routes, problems);
        reportMenuRouteProblems(problems);

        // 添加路由
        routes.forEach((item: any) => {
            if (item.meta.isFull) {
                router.addRoute(item as RouteRecordRaw);
            } else {
                router.addRoute(LAYOUT_ROUTE_NAME, item as RouteRecordRaw);
            }
        });

        useKeepALiveNames().setCacheKeepAlive(cacheList);
        useRoutesList().setRoutesList(routes);
    } catch (e: any) {
        console.error('获取菜单权限信息失败', e);
        clearSession();
        throw e;
    }
}
