import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { URL_404 } from '@/router/staticRouter';
import { MenuRoute, MenuRouteProblem, backEndRouterConverter, normalizeMenuRoutes, reportMenuRouteProblems } from '@/router/menuRouteResolver';

/** 造一个已注册到组件的叶子菜单 */
function page(path: string, routeName: string): MenuRoute {
    return { path, name: routeName, meta: {}, component: { name: routeName } };
}

/** 造一个没注册组件的菜单（父级分组，或失联的叶子） */
function hollow(path: string, routeName: string, extra: Partial<MenuRoute> = {}): MenuRoute {
    return { path, name: routeName, meta: {}, ...extra };
}

function collect(...routes: MenuRoute[]) {
    const problems: MenuRouteProblem[] = [];
    normalizeMenuRoutes(routes, problems);
    return { route: routes[0], problems };
}

describe('分组菜单必须有能落地的去处', () => {
    it('悬空 redirect 换成第一个可渲染子菜单', () => {
        // 真实故障现场：机器列表菜单已下线，父级仍写着 machine/list（相对路径 → /machine/machine/list）
        const { route, problems } = collect(
            hollow('/machine', 'Machine', {
                redirect: 'machine/list',
                children: [page('/machine/cron-job', 'CronJobList'), page('/machine/security', 'SecurityConfList')],
            })
        );

        expect(route.redirect).toBe('/machine/cron-job');
        expect(problems).toHaveLength(1);
        expect(problems[0].reason).toContain('/machine/machine/list');
    });

    it('redirect 指向真实存在的子菜单时保持原配置', () => {
        // /sys/resources 这类写法是对的，不能被"兜底"改写：管理员特意选的落地页要保留
        const { route, problems } = collect(
            hollow('/sys', 'sys', {
                redirect: '/sys/resources',
                children: [page('/sys/resources', 'ResourceList'), page('/sys/accounts', 'AccountList')],
            })
        );

        expect(route.redirect).toBe('/sys/resources');
        expect(problems).toEqual([]);
    });

    it('没配 redirect 的分组静默补齐（侧栏里分组只是展开，不算脱节）', () => {
        // 全仓有八个分组没配 redirect，把它们当错误报出来只会把真正的脱节淹掉
        const { route, problems } = collect(hollow('/dbms', 'DBMS', { children: [page('/dbms/sync', 'SyncTaskList')] }));

        expect(route.redirect).toBe('/dbms/sync');
        expect(problems).toEqual([]);
    });

    it('隐藏占位菜单不惊动开发者，但路由照样收到 404', () => {
        // 后端 seed 里有一条 code=empty、routeName=empty 的隐藏菜单（「无页面权限」占位），
        // 它本来就没有组件：不能因此每次加载都刷告警，但也不能留一条点了没反应的路由
        const hidden = hollow('/empty', 'empty', { meta: { isHide: true } });
        const { route, problems } = collect(hidden);

        expect(route.redirect).toBe(URL_404);
        expect(problems).toEqual([]);
    });

    it('要往深处找：第一个孩子是空分组时不能就此放弃', () => {
        // 菜单层级常见「分组 → 子分组 → 页面」，只看一层会把 redirect 指向同样不可渲染的子分组
        const { route } = collect(
            hollow('/ops', 'ops', {
                children: [hollow('/ops/monitor', 'Monitor', { children: [page('/ops/monitor/panels', 'Panels')] })],
            })
        );

        expect(route.redirect).toBe('/ops/monitor/panels');
    });

    it('递归修好每一层空分组', () => {
        const { route } = collect(
            hollow('/a', 'a', {
                children: [hollow('/a/b', 'b', { children: [page('/a/b/c', 'c')] })],
            })
        );

        expect(route.children?.[0].redirect).toBe('/a/b/c');
    });
});

describe('落地页的选择不能踩到隐藏页或别的分支', () => {
    it('跳往另一条分支的绝对路径是正当配置，不得被改写', () => {
        // 管理员可以把分组指到兄弟分支上；只拿自己的子树判存在性，就会把这种正确配置
        // 当成悬空并悄悄改掉。分组必须自带可渲染子页，否则走不到这个分支（那样用例是空的）
        const { route, problems } = collect(
            hollow('/a', 'a', { redirect: '/b/first', children: [page('/a/local', 'Local')] }),
            hollow('/b', 'b', { children: [page('/b/first', 'First')] })
        );

        expect(route.redirect).toBe('/b/first');
        expect(problems).toEqual([]);
    });

    it('落地页跳过隐藏子菜单（不会从分组名点进一条详情页）', () => {
        // 隐藏页不在子菜单里显示，选中它等于把一个看不见的页面当成分组的首页
        const hiddenDetail = hollow('/dbms/detail', 'DbDetail', { meta: { isHide: true }, component: { name: 'DbDetail' } });
        const { route, problems } = collect(hollow('/dbms', 'DBMS', { children: [hiddenDetail, page('/dbms/instances', 'DbList')] }));

        expect(route.redirect).toBe('/dbms/instances');
        expect(problems).toEqual([]);
    });

    it('整棵子树只剩隐藏页时照样用它，而不是拿 404 当落地页', () => {
        // 子项全隐藏只是把入口收起来了，落到那个页比 404 更接近管理员本意
        const only = hollow('/ops/hidden', 'Hidden', { meta: { isHide: true }, component: { name: 'Hidden' } });
        const { route } = collect(hollow('/ops', 'ops', { children: [only] }));

        expect(route.redirect).toBe('/ops/hidden');
    });
});

describe('失联的叶子菜单不再静默空白', () => {
    it('没有注册组件的叶子落到 404 并报出问题', () => {
        // 新增菜单忘了在 route.ts 注册组件时，原行为是注册一条 component 为空的路由：
        // 点了没反应、没报错也没 404，用户只当功能没做
        const { route, problems } = collect(hollow('/flow/new-page', 'BrandNewPage'));

        expect(route.redirect).toBe(URL_404);
        expect(problems[0].reason).toContain('没有注册');
        expect(problems[0].routeName).toBe('BrandNewPage');
    });

    it('已注册组件的叶子不受影响', () => {
        const { route, problems } = collect(page('/flow/procdefs', 'ProcdefList'));

        expect(route.redirect).toBeUndefined();
        expect(problems).toEqual([]);
    });

    it('外链菜单（已有组件）不被当成失联', () => {
        const link = hollow('/doc', 'Doc', { component: { name: 'Iframe' }, meta: { link: 'https://example.com' } });
        const { problems } = collect(link);

        expect(problems).toEqual([]);
    });
});

describe('菜单转路由：路径拼接是兜底判断的地基', () => {
    it('子菜单 code 拼成完整路径，meta 字符串解析成对象', () => {
        // 归一化时靠「redirect 解析后与子路由 path 全等」判断跳转是否悬空，
        // 拼接规则一变，所有分组都会被误判成悬空（或反过来，真悬空的判不出来）
        const routes = backEndRouterConverter({ CronJobList: () => undefined }, [
            {
                code: '/machine',
                name: '机器',
                meta: JSON.stringify({ routeName: 'Machine', redirect: 'machine/list' }),
                children: [{ code: 'cron-job', name: '定时任务', meta: JSON.stringify({ routeName: 'CronJobList' }) }],
            },
        ]);

        expect(routes[0].path).toBe('/machine');
        expect(routes[0].children[0].path).toBe('/machine/cron-job');
        expect(routes[0].meta.routeName).toBe('Machine');
    });

    it('未注册的 routeName 拿到空组件（这才是要兜底的情形）', () => {
        const routes = backEndRouterConverter({ Known: () => undefined }, [{ code: '/unknown', name: 'x', meta: '{"routeName":"Other"}' }]);

        expect(routes[0].component).toBeUndefined();
    });
});

describe('脱节项必须报得出来', () => {
    afterEach(() => vi.restoreAllMocks());

    it('有脱节就打 error，且一行说清地点原因与去向', () => {
        // 不报出来就等于修了但没人知道哪里还坏着：这类失效的代价正是「没人发现」
        const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);
        const lines = reportMenuRouteProblems([
            { path: '/machine', routeName: 'Machine', reason: '跳转目标「/machine/machine/list」没有对应菜单', fallback: '/machine/cron-job' },
        ]);

        expect(spy).toHaveBeenCalledOnce();
        expect(lines).toHaveLength(1);
        expect(lines[0]).toContain('/machine/machine/list');
        expect(lines[0]).toContain('/machine/cron-job');
    });

    it('没有脱节不制造噪声', () => {
        const spy = vi.spyOn(console, 'error').mockImplementation(() => undefined);

        expect(reportMenuRouteProblems([])).toEqual([]);
        expect(spy).not.toHaveBeenCalled();
    });
});

describe('接线：路由初始化真的调了兜底与上报', () => {
    // dynamicRouter 依赖整个 router 实例（会拉起 ws 与业务 api），在测试里 import 它会踩循环初始化，
    // 所以这两步只能按源码接线断言钉住：漏掉任一步，界面仍旧是「点了没反应」而单测全绿
    const source = readFileSync(join(import.meta.dirname, '..', 'router', 'dynamicRouter.ts'), 'utf8');

    it('转换后归一化整棵菜单树', () => {
        expect(source).toMatch(/normalizeMenuRoutes\(routes, problems\)/);
    });

    it('归一化结果上报而不是自吞', () => {
        expect(source).toMatch(/reportMenuRouteProblems\(problems\)/);
    });
});
