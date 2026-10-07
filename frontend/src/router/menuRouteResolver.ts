import { URL_401, URL_404 } from './staticRouter';
import { LinkTypeEnum } from '@/common/commonEnum';

const Link = () => import('@/layout/routerView/link.vue');
const Iframe = () => import('@/layout/routerView/iframes.vue');

type RouterConvCallbackFunc = (router: any) => void;

/** 转换后的菜单路由节点：与 vue-router 的 RouteRecordRaw 兼容，这里只声明归一化需要关心的字段 */
export type MenuRoute = {
    path: string;
    /** 后端菜单 meta.routeName，对应各模块 route.ts 里注册的组件键 */
    name?: string;
    meta: Record<string, unknown>;
    component?: unknown;
    redirect?: string;
    children?: MenuRoute[];
};

/** 一处菜单与前端路由表脱节的记录，用于集中告警 */
export type MenuRouteProblem = {
    path: string;
    routeName?: string;
    /** 为什么这条菜单落不到页面上 */
    reason: string;
    /** 归一化后实际采用的跳转目标 */
    fallback: string;
};

/**
 * 归一化整棵菜单路由（就地修改）。
 *
 * 后端菜单只声明 routeName，组件由前端各模块的 route.ts 注册表提供，两边靠字符串对齐、
 * 没有任何编译期约束：菜单改名/下线后父级 redirect 会悬空（机器列表菜单已下线，父级仍写着
 * `machine/list`，解析成 `/machine/machine/list`），新增菜单忘了注册组件会得到一条
 * `component: undefined` 的路由。两种情况的界面表现都是**一片空白**——没有报错、没有 404，
 * 用户只当功能没做，排查时也没有任何线索。这里统一兜底并把问题显式报出来。
 *
 * redirect 是否有效必须拿**全量路径**判断：分组可以被配到另一条分支上（`/a` → `/b/page`），
 * 只跟自己子树比会把这种正确配置误判成悬空并擅自改写
 */
export function normalizeMenuRoutes(routes: MenuRoute[], problems: MenuRouteProblem[]): void {
    const knownPaths = new Set<string>([URL_404, URL_401]);
    collectPaths(routes, knownPaths);
    routes.forEach((route) => normalizeMenuRoute(route, knownPaths, problems));
}

function collectPaths(routes: MenuRoute[], into: Set<string>): void {
    for (const route of routes) {
        into.add(route.path);
        collectPaths(route.children ?? [], into);
    }
}

function normalizeMenuRoute(route: MenuRoute, knownPaths: Set<string>, problems: MenuRouteProblem[]): void {
    (route.children ?? []).forEach((child) => normalizeMenuRoute(child, knownPaths, problems));

    // 有组件即为可渲染页面（外链在转换阶段已被赋值成 Link/Iframe 组件）
    if (route.component) {
        return;
    }

    const landing = firstRenderableDescendant(route);
    if (landing) {
        // 分组节点本身没有页面，必须有个能落地的去处：配置的 redirect 若指向已改名/已下线的
        // 子菜单就会落空（机器列表菜单已下线，父级仍写着 machine/list），此时改用第一个能渲染的后代
        const configured = route.redirect ? toAbsolutePath(route.path, route.redirect) : '';
        if (configured && knownPaths.has(configured)) {
            return;
        }
        // 没配 redirect 是正常形态（侧栏里分组只是展开），静默补齐即可；
        // 只有配了却落不到任何路由的情况才需要告诉开发者：菜单与路由已经对不上了
        if (configured) {
            problems.push({
                path: route.path,
                routeName: route.name,
                reason: `跳转目标「${configured}」没有对应菜单（子菜单可能已改名或下线）`,
                fallback: landing.path,
            });
        }
        route.redirect = landing.path;
        return;
    }

    // 叶子菜单：没组件、也没有子菜单可去，只能显式交给 404 页。
    // 留一条 component 为空的记录等于把「点了没反应」藏进界面里
    if (!route.redirect) {
        // 隐藏菜单不会被人点（如后端占位的「无页面权限」节点），它没组件不值得惊动开发者，
        // 但路由本身照样要收向 404：一条 component 永远为空的路由迟早会被直接访问到
        if (!route.meta?.isHide) {
            problems.push({
                path: route.path,
                routeName: route.name,
                reason: '前端没有注册该 routeName 对应的页面组件',
                fallback: URL_404,
            });
        }
        route.redirect = URL_404;
    }
}

/**
 * 把脱节项一次性报给开发者：菜单与路由对不上时界面只是空白，没有任何线索，
 * 不报出来就永远没人发现（机器列表菜单下线后父级跳转悬空就是这么存在了很久的）
 *
 * @returns 已格式化的告警行，便于测试与调用方复用
 */
export function reportMenuRouteProblems(problems: MenuRouteProblem[]): string[] {
    if (problems.length === 0) {
        return [];
    }
    const lines = problems.map((p) => `${p.path}（routeName=${p.routeName ?? '-'}）${p.reason}，已降级跳转到 ${p.fallback}`);
    console.error('菜单与前端路由存在脱节:', lines);
    return lines;
}

/**
 * 分组的落地页：取第一个「能渲染且会在子菜单里显示」的后代。
 *
 * 隐藏菜单（详情页、占位页）不会出现在子菜单里，把它当落地页会让人从一个分组名点进一条详情页；
 * 整棵子树只剩隐藏页面时才退而求其次——总比落到空白页或 404 更接近管理员的本意
 */
function firstRenderableDescendant(route: MenuRoute): MenuRoute | undefined {
    return findRenderable(route, true) ?? findRenderable(route, false);
}

function findRenderable(route: MenuRoute, onlyVisible: boolean): MenuRoute | undefined {
    for (const child of route.children ?? []) {
        if (child.component && !(onlyVisible && child.meta?.isHide)) {
            return child;
        }
        // 只看一层孩子不够：菜单层级是「分组 → 分组 → 页面」，父级的第一个孩子常常同样是空分组
        const nested = findRenderable(child, onlyVisible);
        if (nested) {
            return nested;
        }
    }
    return undefined;
}

/**
 * 菜单里的 redirect 两种写法都有（`/sys/resources` 与 `machine/list`），
 * 后者按 vue-router 的规则相对父路径解析，不归一化就没法判断它是否真的存在
 */
function toAbsolutePath(base: string, target: string): string {
    if (target.startsWith('/')) {
        return target;
    }
    return `${base}/${target}`.replace(/\/{2,}/g, '/');
}

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
