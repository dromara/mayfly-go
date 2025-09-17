<template>
    <div class="redis-data-op h-full">
        <el-splitter>
            <el-splitter-panel size="35%" max="50%">
                <div class="key-list-vtree h-full card !p-1">
                    <el-scrollbar>
                        <el-row :gutter="5">
                            <el-col :span="2">
                                <el-input v-model="state.keySeparator" :placeholder="$t('redis.delimiter')" size="small" />
                            </el-col>
                            <el-col :span="18">
                                <el-input
                                    @clear="clear"
                                    v-model="scanParam.match"
                                    @keyup.enter.native="searchKey()"
                                    :placeholder="$t('redis.keyMatchTips')"
                                    clearable
                                    size="small"
                                />
                            </el-col>
                            <el-col :span="4">
                                <el-button
                                    :disabled="!scanParam.id || !scanParam.db"
                                    @click="searchKey()"
                                    type="success"
                                    icon="search"
                                    size="small"
                                    plain
                                ></el-button>
                            </el-col>
                        </el-row>

                        <el-row :gutter="5" class="mb-1 mt-1">
                            <el-col :span="19">
                                <el-button :disabled="!scanParam.id || !scanParam.db" @click="scan(true)" type="success" icon="more" size="small" plain>
                                    {{ $t('redis.loadMore') }}
                                </el-button>

                                <el-button
                                    v-auth="'redis:data:save'"
                                    :disabled="!scanParam.id || !scanParam.db"
                                    @click="showNewKeyDialog"
                                    type="primary"
                                    icon="plus"
                                    size="small"
                                    plain
                                    class="!ml-0.5"
                                >
                                    {{ $t('redis.addKey') }}
                                </el-button>

                                <el-button
                                    :disabled="!scanParam.id || !scanParam.db"
                                    @click="flushDb"
                                    type="danger"
                                    plain
                                    v-auth="'redis:data:del'"
                                    size="small"
                                    icon="delete"
                                    class="!ml-0.5"
                                >
                                    flush
                                </el-button>
                            </el-col>
                            <el-col :span="5">
                                <span class="mt-1" style="display: inline-block">keys:{{ state.dbsize }}</span>
                            </el-col>
                        </el-row>

                        <el-tree
                            ref="keyTreeRef"
                            :highlight-current="true"
                            :data="keyTreeData"
                            :props="treeProps"
                            :indent="8"
                            node-key="key"
                            :auto-expand-parent="false"
                            :default-expanded-keys="Array.from(state.keyTreeExpanded)"
                            @node-click="handleKeyTreeNodeClick"
                            @node-expand="keyTreeNodeExpand"
                            @node-collapse="keyTreeNodeCollapse"
                            @node-contextmenu="rightClickNode"
                        >
                            <template #default="{ node, data }">
                                <span class="el-dropdown-link key-list-custom-node" :title="node.label">
                                    <span v-if="data.type == 1">
                                        <SvgIcon :size="15" :name="node.expanded ? 'folder-opened' : 'folder'" />
                                    </span>
                                    <span :class="'ml-1 ' + (data.type == 1 ? 'folder-label' : 'key-label')">
                                        {{ node.label }}
                                    </span>

                                    <span v-if="!node.isLeaf" class="ml-1" style="font-weight: bold"> ({{ data.keyCount }}) </span>
                                </span>
                            </template>
                        </el-tree>
                    </el-scrollbar>

                    <contextmenu :dropdown="state.contextmenu.dropdown" :items="state.contextmenu.items" ref="contextmenuRef" />
                </div>
            </el-splitter-panel>

            <el-splitter-panel>
                <div class="h-full card !p-1 key-deatil">
                    <el-tabs class="h-full" @tab-remove="removeDataTab" v-model="state.activeName">
                        <el-tab-pane class="h-full" closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label" :name="dt.key">
                            <key-detail :redis="redisInst" :key-info="dt.keyInfo" @change-key="searchKey()" @del-key="delKey" />
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-splitter-panel>
        </el-splitter>

        <el-dialog :title="$t('redis.addKey')" v-model="newKeyDialog.visible" width="500px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form ref="keyForm" label-width="auto" :rules="keyFormRules" :model="newKeyDialog.keyInfo">
                <el-form-item prop="key" label="Key" required>
                    <el-input v-model.trim="newKeyDialog.keyInfo.key"></el-input>
                </el-form-item>
                <el-form-item prop="type" :label="$t('common.type')">
                    <el-select v-model="newKeyDialog.keyInfo.type" default-first-option>
                        <el-option key="string" label="string" value="string"></el-option>
                        <el-option key="hash" label="hash" value="hash"></el-option>
                        <el-option key="set" label="set" value="set"></el-option>
                        <el-option key="zset" label="zset" value="zset"></el-option>
                        <el-option key="list" label="list" value="list"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>

            <template #footer>
                <el-button @click="cancelNewKey()">{{ $t('common.cancel') }}</el-button>
                <el-button v-auth="'redis:data:save'" type="primary" @click="newKey">{{ $t('common.confirm') }}</el-button>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { redisApi } from '../api';
import { ref, defineAsyncComponent, toRefs, reactive, onMounted, nextTick, Ref, watch, useTemplateRef, getCurrentInstance } from 'vue';
import { ElMessageBox } from 'element-plus';
import { isTrue, notNull } from '@/common/assert';
import { copyToClipboard } from '@/common/utils/string';
import { keysToTree, sortByTreeNodes, keysToList } from '../utils';
import { Contextmenu, ContextmenuItem } from '@/components/contextmenu';
import { RedisInst } from '../redis';
import { useI18n } from 'vue-i18n';
import { useI18nDeleteConfirm, useI18nDeleteSuccessMsg, useI18nFormValidate, useI18nOperateSuccessMsg } from '@/hooks/useI18n';
import { Rules } from '@/common/rule';
import { RedisOpComp } from '@/views/ops/redis/resource';

const KeyDetail = defineAsyncComponent(() => import('../KeyDetail.vue'));

const { t } = useI18n();

const emits = defineEmits(['init']);

const keyFormRules = {
    key: [Rules.requiredInput('Key')],
};

const cmCopyKey = new ContextmenuItem('copyValue', 'Copy')
    .withIcon('CopyDocument')
    .withHideFunc((data: any) => !data.isLeaf)
    .withOnClick(async (data: any) => await copyToClipboard(data.key));

const cmNewTabOpen = new ContextmenuItem('newTabOpenKey', 'redis.newTabOpen')
    .withIcon('plus')
    .withHideFunc((data: any) => !data.isLeaf)
    .withOnClick((data: any) => showKeyDetail(data.key, true));

const cmDelKey = new ContextmenuItem('delKey', 'common.delete')
    .withIcon('delete')
    .withPermission('redis:data:del')
    .withHideFunc((data: any) => !data.isLeaf)
    .withOnClick((data: any) => delKey(data.key));

const treeProps = {
    label: 'name',
    children: 'children',
    isLeaf: 'leaf',
};

const defaultCount = 250;

const contextmenuRef = ref();
const keyTreeRef: any = ref(null);
const keyFormRef = useTemplateRef('keyForm');

const redisInst: Ref<RedisInst> = ref(new RedisInst());

const state = reactive({
    defaultExpendKey: [] as any,
    tags: [],
    redisList: [] as any,
    dbList: [],
    keyTreeHeight: '100px',
    loadingKeyTree: false,
    keys: [] as any,
    keySeparator: ':',
    keyTreeData: [] as any,
    keyTreeExpanded: new Set(),
    activeName: '',
    dataTabs: {} as any,
    rightClickNode: {} as any,
    scanParam: {
        id: null as any,
        mode: '',
        db: null as any,
        match: null,
        count: defaultCount,
        cursor: {},
    },
    newKeyDialog: {
        visible: false,
        keyInfo: {
            type: 'string',
            timed: -1,
            key: '',
        },
    },
    dbsize: 0,
    contextmenu: {
        dropdown: {
            x: 0,
            y: 0,
        },
        items: [cmCopyKey, cmNewTabOpen, cmDelKey],
    },
});

const { scanParam, keyTreeData, newKeyDialog } = toRefs(state);

onMounted(async () => {
    emits('init', { name: RedisOpComp.name, ref: getCurrentInstance()?.exposed });
});

const scan = async (appendKey = false) => {
    isTrue(state.scanParam.id != null, 'redis.redisSelectErr');

    const match: string = state.scanParam.match || '';
    if (!match) {
        state.scanParam.count = defaultCount;
    } else if (match.indexOf('*') != -1) {
        const dbsize = state.dbsize;
        // 如果为模糊搜索，并且搜索的key模式大于指定字符数，则将count设大点scan
        if (match.length > 10) {
            state.scanParam.count = dbsize > 100000 ? Math.floor(dbsize / 10) : 1000;
        } else {
            state.scanParam.count = defaultCount;
        }
    }

    const scanParam = { ...state.scanParam };
    // 集群模式count设小点，因为后端会从所有master节点scan一遍然后合并结果,默认假设redis集群有3个master
    if (scanParam.mode == 'cluster') {
        scanParam.count = Math.floor(state.scanParam.count / 3);
    }

    try {
        state.loadingKeyTree = true;
        const res = await redisApi.scan.request(scanParam);
        // 追加key，则将新key合并至原keys（加载更多）
        if (appendKey) {
            state.keys = [...state.keys, ...res.keys];
        } else {
            state.keys = res.keys;
        }
        setKeyList(state.keys);
        state.dbsize = res.dbSize;
        state.scanParam.cursor = res.cursor;
    } finally {
        state.loadingKeyTree = false;
    }
};

const setKeyList = (keys: any) => {
    state.keyTreeData = state.keySeparator ? keysToTree(keys, state.keySeparator, state.keyTreeExpanded) : keysToList(keys);
    nextTick(() => {
        // key长度小于指定数量，则展开所有节点
        if (keys.length <= 20) {
            expandAllKeyNode(state.keyTreeData);
        }

        sortByTreeNodes(keyTreeRef.value.root.childNodes);
    });
};

// 展开所有节点
const expandAllKeyNode = (nodes: any) => {
    for (let node of nodes) {
        if (!node.children) {
            continue;
        }
        state.keyTreeExpanded.add(node.key);
        for (let i = 0; i < node.children.length; i++) {
            expandAllKeyNode(node.children);
        }
    }
};

const handleKeyTreeNodeClick = async (data: any) => {
    // 关闭可能存在的右击菜单
    contextmenuRef.value.closeContextmenu();
    // 目录则不做处理
    if (data.type == 1) {
        return;
    }

    showKeyDetail(data.key);
};

const showKeyDetail = async (key: any, newTab = false) => {
    let keyInfo;
    if (typeof key == 'object') {
        keyInfo = key;
    } else {
        if (state.dataTabs[key]) {
            state.activeName = key;
            return;
        }
        const res = await redisApi.keyInfo.request({ id: state.scanParam.id, db: state.scanParam.db, key: key });
        keyInfo = {
            key: key,
            type: res.type,
            timed: res.ttl,
        };
    }

    let label = keyInfo.key;
    if (label.length > 40) {
        label = label.slice(0, 40) + '...';
    }
    const dataTab = {
        key: keyInfo.key,
        label,
        keyInfo,
    };

    if (!newTab) {
        delete state.dataTabs[state.activeName];
    }

    state.dataTabs[keyInfo.key] = dataTab;
    state.activeName = keyInfo.key;
};

const removeDataTab = (targetName: string) => {
    const tabNames = Object.keys(state.dataTabs);
    let activeName = state.activeName;
    tabNames.forEach((name, index) => {
        if (name === targetName) {
            const nextTab = tabNames[index + 1] || tabNames[index - 1];
            if (nextTab) {
                activeName = nextTab;
            }
        }
    });
    state.activeName = activeName;
    delete state.dataTabs[targetName];
};

const keyTreeNodeExpand = (data: any, node: any) => {
    state.keyTreeExpanded.add(data.key);
    // async sort nodes
    if (!node.customSorted) {
        node.customSorted = true;
        sortByTreeNodes(node.childNodes);
    }
};

const keyTreeNodeCollapse = (data: any) => {
    state.keyTreeExpanded.delete(data.key);
};

const rightClickNode = (event: any, data: any, node: any) => {
    const { clientX, clientY } = event;
    state.contextmenu.dropdown.x = clientX;
    state.contextmenu.dropdown.y = clientY;
    contextmenuRef.value.openContextmenu(node);
    keyTreeRef.value.setCurrentKey(node.key);
};

const searchKey = async () => {
    state.scanParam.cursor = {};
    await scan(false);
};

const clear = () => {
    resetScanParam();
    if (state.scanParam.id) {
        scan();
    }
};

const resetScanParam = () => {
    state.scanParam.match = null;
    state.scanParam.cursor = {};
    state.keyTreeExpanded.clear();
    state.dataTabs = {};
    state.activeName = '';
};

const showNewKeyDialog = () => {
    notNull(state.scanParam.id, t('redis.redisSelectErr'));
    resetNewKeyInfo();
    state.newKeyDialog.visible = true;
};

const flushDb = () => {
    ElMessageBox.confirm(t('redis.flushDbTips', { db: state.scanParam.db }), t('common.hint'), {
        confirmButtonText: t('common.confirm'),
        cancelButtonText: t('common.cancel'),
        type: 'warning',
    })
        .then(() => {
            // FLUSHDB [ASYNC | SYNC]
            redisInst.value.runCmd(['FLUSHDB']).then(() => {
                useI18nOperateSuccessMsg();
                searchKey();
            });
        })
        .catch(() => {});
};

const cancelNewKey = () => {
    resetNewKeyInfo();
    state.newKeyDialog.visible = false;
};

const newKey = async () => {
    await useI18nFormValidate(keyFormRef);
    const keyInfo = state.newKeyDialog.keyInfo;
    const key = keyInfo.key;

    showKeyDetail(
        {
            ...keyInfo,
        },
        true
    );
    state.newKeyDialog.visible = false;

    // 添加新增的key至key tree
    state.keys.push(key);
    setKeyList(state.keys);
};

const resetNewKeyInfo = () => {
    state.newKeyDialog.keyInfo.key = '';
    state.newKeyDialog.keyInfo.type = 'string';
    state.newKeyDialog.keyInfo.timed = -1;
};

const delKey = async (key: string) => {
    await useI18nDeleteConfirm(key);
    // DEL key [key ...]
    await redisInst.value.runCmd(['DEL', key]);
    useI18nDeleteSuccessMsg();
    searchKey();

    removeDataTab(key);
};

const onDbClick = async (dbInfo: any) => {
    if (state.scanParam.db == dbInfo.db) {
        return;
    }
    resetScanParam();

    state.scanParam.id = dbInfo.id;
    state.scanParam.mode = dbInfo.mode;
    state.scanParam.db = dbInfo.db;

    redisInst.value.id = dbInfo.id;
    redisInst.value.db = Number.parseInt(dbInfo.db);

    scan();
};

defineExpose({
    onDbClick,
});
</script>

<style lang="scss" scoped>
.key-deatil {
    .el-tabs__header {
        background-color: var(--el-color-white);
        border-bottom: 1px solid var(--el-border-color);
    }

    ::v-deep(.el-tabs__item) {
        padding: 0 10px;
        height: 29px;
    }

    ::v-deep(.el-tabs__nav-next) {
        line-height: 29px;
    }
    ::v-deep(.el-tabs__nav-prev) {
        line-height: 29px;
    }
}

.redis-data-op {
    .key-list-vtree .folder-label {
        font-weight: bold;
    }

    .key-list-vtree .key-label {
        color: #67c23a;
    }

    .key-list-vtree .key-list-custom-node {
        width: 100%;
        overflow: hidden;
        text-overflow: ellipsis;
        /*note the following 2 items should be same value, may not consist with itemSize*/
        height: 22px;
        line-height: 22px;
    }
}
</style>
