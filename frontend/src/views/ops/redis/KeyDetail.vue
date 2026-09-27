<template>
    <div class="flex h-full min-h-0 flex-col gap-3">
        <KeyMetaHeader
            :key-name="props.keyName"
            :meta="store.meta.value"
            :view="store.view.value"
            :descriptor="store.descriptor.value"
            :size="store.total.value"
            :view-options="viewOptions"
            :missing="missing"
            @refresh="onRefresh"
            @switch-view="store.switchView"
            @rename="onKeyDialog('rename')"
            @copy="onKeyDialog('copy')"
            @ttl="onKeyDialog('ttl')"
            @del="emit('del', props.keyName)"
        />

        <!-- 数据区直接展示：命令控制台是实例级的独立入口，不挂在每个 key 的详情里 -->
        <div class="card flex min-h-0 flex-1 flex-col p-2!">
            <div v-if="store.meta.value && !store.descriptor.value && !store.creating.value" class="p-3">
                <el-alert :title="$t('redis.unsupportedType', { type: store.meta.value?.type })" type="warning" :closable="false" show-icon>
                    <div class="mt-1 text-xs">
                        {{ $t('redis.useConsole') }}
                        <el-button link type="primary" size="small" @click="emit('openConsole')">{{ $t('redis.tabConsole') }}</el-button>
                    </div>
                </el-alert>
            </div>
            <!-- key 已被删除或过期：不能退化成「新建 key」的编辑态，否则用户以为还在改原 key -->
            <div v-else-if="missing" class="flex h-full items-center justify-center">
                <el-empty :description="$t('redis.keyGone')">
                    <el-button type="primary" icon="refresh" @click="emit('refreshTree')">{{ $t('common.refresh') }}</el-button>
                </el-empty>
            </div>
            <KeyMemberArea v-else :store="store" :ttl="props.ttl" />
        </div>

        <AutoFormDialog
            v-model:visible="keyDialog.visible"
            :title="keyDialog.title"
            :items="keyDialog.items"
            :data="keyDialog.data"
            :confirm-api="onKeyDialogConfirm"
            width="520px"
        />
    </div>
</template>

<script lang="ts" setup>
import { AutoFormDialog, defineFormItems, type AutoFormItem } from '@/components/auto-form';
import { Rules } from '@/common/rule';
import { computed, onMounted, reactive, watch } from 'vue';
import { useI18n } from 'vue-i18n';
import { redisApi } from './api';
import KeyMemberArea from './keyview/KeyMemberArea.vue';
import KeyMetaHeader from './keyview/KeyMetaHeader.vue';
import { useKeyView } from './keyview/useKeyView';
import { viewsOfType } from './keyview/descriptor';
import { PERM_DATA_DEL, PERM_DATA_SAVE } from './keyview/permission';
import type { RedisInst } from './redis';

const props = defineProps<{
    redis: RedisInst;
    /** 正在查看的 key */
    keyName: string;
    /** 新增态下用户选定的数据类型（即初始视角） */
    createView?: string;
    /** 新增 key 时随首个成员一起设置的过期时间（秒） */
    ttl?: number;
    /** 实例开放的库号：复制目标库只能选这些（资源树也只渲染这些库） */
    dbs?: number[];
}>();

const emit = defineEmits<{
    /** 删除 key：由持有 tab 的列表页负责刷新树与关闭 tab */
    del: [key: string];
    /** 重命名成功：列表页需要刷新 key 树并把 tab 标题切到新名 */
    renamed: [newKey: string];
    /** 首个成员写入成功（key 从无到有）：列表页需要刷新 key 树 */
    created: [];
    /** 请求刷新 key 树（打开的 key 已不存在时给用户一条出路） */
    refreshTree: [];
    /** 请求打开命令控制台 tab（不支持可视化的类型要靠它操作） */
    openConsole: [];
}>();

const { t } = useI18n();

const store = useKeyView(
    () => ({ id: props.redis.id, db: props.redis.db }),
    () => props.keyName
);

const viewOptions = computed(() => viewsOfType(store.descriptors.value, store.meta.value?.type ?? ''));

/** 打开的 key 已被删除或过期：判据由 store 统一给出，它能区分「新增流程里还没写入」与「曾经存在后来消失」 */
const missing = computed(() => store.missing.value);

// key 从「不存在」到「存在」只发生在首个成员写入成功时，此时通知列表页刷新树
watch(
    () => store.meta.value?.exists,
    (exists, before) => {
        if (exists && before === false) {
            emit('created');
        }
    }
);

async function load() {
    await store.loadDescriptors();
    if (props.createView) {
        await store.startCreate(props.createView);
        return;
    }
    await store.loadMeta();
    await store.loadMembers(true);
}

onMounted(load);

watch(() => [props.keyName, props.createView], load);

const onRefresh = async () => {
    await store.loadMeta();
    await store.loadMembers(true);
};

/** key 级操作的弹层状态与字段声明 */
type KeyDialogKind = 'rename' | 'copy' | 'ttl';

const keyDialog = reactive({
    visible: false,
    kind: 'rename' as KeyDialogKind,
    title: '',
    data: null as Record<string, unknown> | null,
    items: [] as AutoFormItem[],
});

const renameItems = defineFormItems<{ newKey: string }>([{ prop: 'newKey', label: 'Key', required: true, rules: [Rules.requiredInput('redis.keyName')] }]);

const copyItems = defineFormItems<{ newKey: string; db: number; replace: boolean }>([
    { prop: 'newKey', label: 'redis.copyTargetKey', required: true, rules: [Rules.requiredInput('redis.copyTargetKey')] },
    {
        // 目标库做成候选而不是自由输入：填一个实例没开放的库，写进去的 key 在资源树里看不见也删不掉
        prop: 'db',
        label: 'redis.copyTargetDb',
        type: 'select',
        options: async () => (props.dbs ?? []).map((db) => ({ value: db, label: `db${db}` })),
        description: 'redis.copyDbTips',
    },
    { prop: 'replace', label: 'redis.copyReplace', type: 'switch' },
]);

const ttlItems = defineFormItems<{ ttl: number }>([{ prop: 'ttl', label: 'redis.ttl', type: 'number', description: 'redis.ttlTips' }]);

const onKeyDialog = (kind: KeyDialogKind) => {
    keyDialog.kind = kind;
    keyDialog.title = t(`redis.${kind}Key`);
    keyDialog.items = kind === 'rename' ? renameItems : kind === 'copy' ? copyItems : ttlItems;
    keyDialog.data =
        kind === 'ttl'
            ? { ttl: store.meta.value?.ttl && store.meta.value.ttl > 0 ? store.meta.value.ttl : -1 }
            : kind === 'copy'
              ? { newKey: `${props.keyName}_copy`, db: props.redis.db, replace: false }
              : { newKey: props.keyName };
    keyDialog.visible = true;
};

async function onKeyDialogConfirm(form: Record<string, unknown>) {
    const target = { id: props.redis.id, db: props.redis.db, key: props.keyName };
    if (keyDialog.kind === 'ttl') {
        await redisApi.setKeyTtl.request({ ...target, ttl: Number(form.ttl ?? -1) });
    } else if (keyDialog.kind === 'rename') {
        await redisApi.renameKey.request({ ...target, newKey: String(form.newKey) });
        // 交给列表页换 tab：本组件随即被卸载，自己再刷一次只会用旧 key 名发出必然失败的在途请求
        emit('renamed', String(form.newKey));
        return;
    } else {
        const targetDb = form.db === undefined || form.db === null || form.db === '' ? undefined : Number(form.db);
        await redisApi.copyKey.request({ ...target, newKey: String(form.newKey), targetDb, replace: Boolean(form.replace) });
    }

    // 成功提示由弹层宿主统一发出，这里只做数据刷新，避免双份 toast
    await onRefresh();
    // 设置过期与复制都会改变 key 列表的节点（TTL 角标 / 多一个 key），
    // 只刷详情会让列表读数停在旧值，重命名路径已由列表页在 onRenamed 里刷新
    emit('refreshTree');
}
</script>
