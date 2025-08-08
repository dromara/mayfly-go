import 'nprogress/nprogress.css';
import { clearSession, getToken } from '@/common/utils/storage';
import openApi from '@/common/openApi';
import { useUserInfo } from '@/store/userInfo';
import { useRoutesList } from '@/store/routesList';
import { useKeepALiveNames } from '@/store/keepAliveNames';
import router from '.';
import { RouteRecordRaw } from 'vue-router';
import { LAYOUT_ROUTE_NAME } from './staticRouter';
import { LinkTypeEnum } from '@/common/commonEnum';

const Link = () => import('@/layout/routerView/link.vue');
const Iframe = () => import('@/layout/routerView/iframes.vue');

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

    useUserInfo().setUserInfo({});

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

type RouterConvCallbackFunc = (router: any) => void;

/**
 * 后端控制路由，后端返回路由 转换为vue route
 *
 * @description routes参数配置简介
 * @param code(path) ==> route.path -> 路由菜单访问路径
 * @param name ==> title，路由标题 相当于route.meta.title
 *
 * @param meta ==> 路由菜单元信息
 * @param meta.routeName ==> route.name -> 路由 name (对应页面组件 name, 可用作 KeepAlive 缓存标识 && 按钮权限筛选) -> 对应模块下route.ts字段key
 * @param meta.redirect ==> route.redirect -> 路由重定向地址
 * @param meta.icon ==> 菜单和面包屑对应的图标
 * @param meta.isHide ==> 是否在菜单中隐藏 (通常列表详情页需要隐藏)
 * @param meta.isFull ==> 菜单是否全屏 (示例：数据大屏页面)
 * @param meta.isAffix ==> 菜单是否固定在标签页中 (首页通常是固定项)
 * @param meta.isKeepAlive ==> 当前路由是否缓存
 * @param meta.linkType ==> 外链类型, 内嵌: 以iframe展示、外链: 新标签打开
 * @param meta.link ==> 外链地址
 * */
export function backEndRouterConverter(allModuleRoutes: any, routes: any, callbackFunc?: RouterConvCallbackFunc, parentPath = '/'): any[] {
    if (!routes) return [];

    return routes.map((item: any) => {
        if (!item.meta) return item;

        // 将json字符串的meta转为对象
        const meta = typeof item.meta === 'string' ? JSON.parse(item.meta) : item.meta;

        // 处理路径
        let path = item.code;
        if (!path.startsWith('/')) {
            path = `${parentPath}/${path}`.replace(/\/+/g, '/');
        }

        // 构建路由对象
        const routeItem: any = {
            path,
            name: meta.routeName,
            meta: {
                ...meta,
                title: item.name,
            },
        };

        // 处理外链
        if (meta.link) {
            routeItem.component = meta.linkType == LinkTypeEnum.Link.value ? Link : Iframe;
        } else {
            // 使用模块路由组件
            routeItem.component = allModuleRoutes[meta.routeName];
        }

        // 处理重定向
        if (meta.redirect) {
            routeItem.redirect = meta.redirect;
        }

        // 处理子路由
        if (item.children) {
            routeItem.children = backEndRouterConverter(allModuleRoutes, item.children, callbackFunc, path);
        }

        // 执行回调
        callbackFunc?.(routeItem);

        return routeItem;
    });
}
