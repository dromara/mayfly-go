import { computed, ref } from 'vue';
import { redisApi } from '../api';
import type { RedisKeyMember, RedisKeyMeta, RedisViewCaps, RedisViewDescriptor } from '../types';
import { EMPTY_CAPS, findView, hasMoreMembers } from './descriptor';

/** 单次请求的成员数，与后端默认页大小对齐 */
const PAGE_SIZE = 50;

/** 操作目标：某个实例的某个库 */
export interface KeyTarget {
    id: number;
    db: number;
}

/**
 * key 数据视图的状态与读写入口。
 *
 * 组件只负责渲染，视角解析、分页游标、写后刷新等逻辑全部在这里；
 * 而「该类型有哪些列、支持哪些操作」由后端描述符决定，因此本文件不含任何类型分支
 */
export function useKeyView(target: () => KeyTarget, currentKey: () => string) {
    const meta = ref<RedisKeyMeta>();
    const descriptors = ref<RedisViewDescriptor[]>([]);
    const view = ref('');
    const members = ref<RedisKeyMember[]>([]);
    const total = ref(0);
    const cursor = ref('');
    const keyword = ref('');
    const loading = ref(false);
    const loadedCount = ref(0);
    /** 本次打开期间该 key 是否真实存在过：用来区分「新建流程里还没写入首个成员」与「原本存在后来被删除/过期」 */
    const everExisted = ref(false);

    const descriptor = computed(() => findView(descriptors.value, view.value));

    /**
     * 新增态：本次会话从没存在过的 key，成员区只提供「写入首个成员」的表单，不出现改删入口。
     * 必须等元信息到位后才判定，否则已有 key 的加载瞬间会被误判成新增态
     */
    const creating = computed(() => !!meta.value && !meta.value.exists && !everExisted.value);
    /**
     * 已失效：曾经存在过、现在读不到了（被删除或自然过期）。
     * 不能退化成新增态，否则用户以为还在改原 key，实际写入的是凭空新建
     */
    const missing = computed(() => !!meta.value && !meta.value.exists && everExisted.value);
    const caps = computed<RedisViewCaps>(() => descriptor.value?.caps ?? meta.value?.caps ?? EMPTY_CAPS);
    const hasMore = computed(() => hasMoreMembers(caps.value, cursor.value, loadedCount.value, total.value, !!keyword.value));

    /** 视角描述符与库无关，同一次会话取一次即可 */
    async function loadDescriptors() {
        if (descriptors.value.length) {
            return;
        }
        descriptors.value = (await redisApi.views.request(target())) ?? [];
    }

    /**
     * 读取 key 元信息，preferView 为空时沿用后端解析出的默认（或自动探测到的）视角。
     * key 不存在（新增态）时保留调用方选定的视角，不能把用户刚选的类型冲掉
     */
    async function loadMeta(preferView?: string) {
        const { id, db } = target();
        const res = await redisApi.keyMeta.request({ id, db, key: currentKey(), view: preferView ?? view.value });
        meta.value = res;
        if (res?.exists) {
            everExisted.value = true;
        }
        view.value = res?.exists ? res.view : preferView || view.value;
        // 元信息一到，总数先跟它对齐：下面的成员读取会用回包的 total 覆盖为最新值
        total.value = res?.size ?? 0;
    }

    /** 进入新增态：按选定视角等待首个成员写入 */
    async function startCreate(viewName: string) {
        view.value = viewName;
        members.value = [];
        total.value = 0;
        cursor.value = '';
        loadedCount.value = 0;
        keyword.value = '';
        await loadMeta(viewName);
    }

    async function loadMembers(reset = false) {
        const key = currentKey();
        if (!key) {
            members.value = [];
            return;
        }

        if (reset) {
            cursor.value = '';
            loadedCount.value = 0;
        }

        // 没有任何视角接管该类型（如模块自带的 ReJSON、vectorset）时不发请求：
        // 后端只会回「不支持的数据类型」，页面此刻展示的是「暂不支持可视化编辑 + 去控制台」的引导，
        // 再叠一个红色报错只会让人以为连接坏了
        if (!descriptor.value) {
            members.value = [];
            total.value = 0;
            cursor.value = '';
            loadedCount.value = 0;
            return;
        }

        loading.value = true;
        try {
            const page = await redisApi.keyValues.request({
                ...target(),
                key,
                view: view.value,
                cursor: cursor.value,
                // 排名分页以下一页起点为已读取条数，游标分页由 cursor 决定
                offset: loadedCount.value,
                size: PAGE_SIZE,
                keyword: keyword.value,
            });
            const rows = page?.members ?? [];
            members.value = reset ? rows : [...members.value, ...rows];
            total.value = page?.total ?? 0;
            loadedCount.value = members.value.length;
            cursor.value = page?.cursor ?? '';
            // 药丸上的成员数与列表必须同源：字段在页面停留期间自然过期时，
            // 只有成员回包的 total 是新的，元信息里的 size 仍是旧值
            if (meta.value) {
                meta.value.size = total.value;
            }
        } finally {
            loading.value = false;
        }
    }

    /** 写操作后回到第一页：中间页在增删后语义不稳定，宁可从头看 */
    async function reload() {
        await loadMeta();
        await loadMembers(true);
    }

    /** 首成员写入成功后 key 即存在，需要把成员区从新增态切回正常读取 */
    async function saveMember(op: string, args: Record<string, string>, row?: RedisKeyMember | null, ttl?: number) {
        await redisApi.putKeyValue.request({ ...target(), key: currentKey(), view: view.value, op, member: row ?? null, args, ttl });
        await reload();
    }

    /** 设置 key 过期时间（新增 key 时可随首个成员一并设置） */
    async function saveTtl(ttl: number) {
        await redisApi.setKeyTtl.request({ ...target(), key: currentKey(), ttl });
    }

    async function deleteMembers(rows: RedisKeyMember[]) {
        await redisApi.putKeyValue.request({ ...target(), key: currentKey(), view: view.value, op: 'delete', members: rows });
        await reload();
    }

    async function runOp(op: string, args: Record<string, string>) {
        const result = await redisApi.runKeyOp.request({ ...target(), key: currentKey(), view: view.value, op, args });
        await reload();
        return result;
    }

    async function switchView(next: string) {
        if (!next || next === view.value) {
            return;
        }
        keyword.value = '';
        await loadMeta(next);
        await loadMembers(true);
    }

    async function search(keywordValue: string) {
        keyword.value = keywordValue;
        await loadMembers(true);
    }

    return {
        meta,
        creating,
        missing,
        loadedCount,
        descriptors,
        view,
        descriptor,
        caps,
        members,
        total,
        cursor,
        keyword,
        loading,
        hasMore,
        loadDescriptors,
        loadMeta,
        startCreate,
        saveTtl,
        loadMembers,
        reload,
        saveMember,
        deleteMembers,
        runOp,
        switchView,
        search,
    };
}

/** 成员区组件与 key 详情共用的数据面契约（组件通过它读写，不各自持有请求逻辑） */
export type KeyViewStore = ReturnType<typeof useKeyView>;
