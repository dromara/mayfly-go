<template>
    <div>
        <el-row>
            <el-col :span="4">
                <el-row type="flex" justify="space-between">
                    <el-col :span="24" class="el-scrollbar flex-auto">
                        <tag-tree @node-click="nodeClick" :load="loadNode">
                            <template #prefix="{ data }">
                                <span v-if="data.type == NodeType.Redis">
                                    <el-popover placement="right-start" title="redis实例信息" trigger="hover" :width="210">
                                        <template #reference>
                                            <SvgIcon name="iconfont icon-op-redis" :size="18" />
                                        </template>
                                        <template #default>
                                            <el-form class="instances-pop-form" label-width="auto" :size="'small'">
                                                <el-form-item label="名称:">{{ data.params.name }}</el-form-item>
                                                <el-form-item label="模式:">{{ data.params.mode }}</el-form-item>
                                                <el-form-item label="链接:">{{ data.params.host }}</el-form-item>
                                                <el-form-item label="备注:">{{ data.params.remark }}</el-form-item>
                                            </el-form>
                                        </template>
                                    </el-popover>
                                </span>

                                <SvgIcon v-if="data.type == NodeType.Db" name="Coin" color="#67c23a" />
                            </template>
                        </tag-tree>
                    </el-col>
                </el-row>
            </el-col>

            <el-col v-loading="state.loadingKeyTree" :span="7">
                <div class="key-list-vtree">
                    <el-row>
                        <el-col :span="2">
                            <el-input v-model="state.keySeparator" placeholder="分割符" size="small" class="ml5" />
                        </el-col>
                        <el-col :span="18">
                            <el-input @clear="clear" v-model="scanParam.match" placeholder="match 支持*模糊key" clearable size="small" class="ml10" />
                        </el-col>
                        <el-col :span="4">
                            <el-button
                                class="ml15"
                                :disabled="!scanParam.id || !scanParam.db"
                                @click="searchKey()"
                                type="success"
                                icon="search"
                                size="small"
                                plain
                            ></el-button>
                        </el-col>
                    </el-row>

                    <el-row class="mb5 mt5">
                        <el-col :span="19">
                            <el-button class="ml5" :disabled="!scanParam.id || !scanParam.db" @click="scan(true)" type="success" icon="more" size="small" plain
                                >加载更多</el-button
                            >

                            <el-button
                                v-auth="'redis:data:save'"
                                :disabled="!scanParam.id || !scanParam.db"
                                @click="showNewKeyDialog"
                                type="primary"
                                icon="plus"
                                size="small"
                                plain
                                >新增key</el-button
                            >

                            <el-button
                                :disabled="!scanParam.id || !scanParam.db"
                                @click="flushDb"
                                type="danger"
                                plain
                                v-auth="'redis:data:del'"
                                size="small"
                                icon="delete"
                                >flush</el-button
                            >
                        </el-col>
                        <el-col :span="5">
                            <span style="display: inline-block" class="mt5">keys:{{ state.dbsize }}</span>
                        </el-col>
                    </el-row>

                    <el-tree
                        :style="{ maxHeight: state.keyTreeHeight, height: state.keyTreeHeight, overflow: 'auto', border: '1px solid #e1f3d8' }"
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
                                <span :class="'ml5 ' + (data.type == 1 ? 'folder-label' : 'key-label')">
                                    {{ node.label }}
                                </span>

                                <span v-if="!node.isLeaf" class="ml5" style="font-weight: bold"> ({{ data.keyCount }}) </span>
                            </span>
                        </template>
                    </el-tree>

                    <!-- right context menu -->
                    <div ref="rightMenuRef" class="key-list-right-menu">
                        <!-- folder right menu -->
                        <div v-if="!state.rightClickNode?.isLeaf"></div>
                        <!-- key right menu -->
                        <div v-else>
                            <el-row>
                                <el-link @click="showKeyDetail(state.rightClickNode.key, true)" type="primary" icon="plus" :underline="false"
                                    >新tab打开</el-link
                                >
                            </el-row>
                            <el-row class="mt5">
                                <el-link @click="delKey(state.rightClickNode.key)" v-auth="'redis:data:del'" type="danger" icon="delete" :underline="false"
                                    >删除</el-link
                                >
                            </el-row>
                        </div>
                    </div>
                </div>
            </el-col>

            <el-col :span="13" style="border-left: 1px solid var(--el-card-border-color)">
                <div class="ml5">
                    <el-tabs @tab-remove="removeDataTab" style="width: 100%" v-model="state.activeName">
                        <el-tab-pane closable v-for="dt in state.dataTabs" :key="dt.key" :label="dt.label" :name="dt.key">
                            <key-detail :redisId="scanParam.id" :db="scanParam.db" :key-info="dt.keyInfo" @change-key="searchKey()" @del-key="delKey" />
                        </el-tab-pane>
                    </el-tabs>
                </div>
            </el-col>
        </el-row>

        <div style="text-align: center; margin-top: 10px"></div>

        <el-dialog title="新增Key" v-model="newKeyDialog.visible" width="500px" :destroy-on-close="true" :close-on-click-modal="false">
            <el-form ref="keyForm" label-width="auto">
                <el-form-item prop="key" label="键名">
                    <el-input v-model.trim="newKeyDialog.keyInfo.key" placeholder="请输入键名"></el-input>
                </el-form-item>
                <el-form-item prop="type" label="类型">
                    <el-select v-model="newKeyDialog.keyInfo.type" default-first-option style="width: 100%" placeholder="请选择类型">
                        <el-option key="string" label="string" value="string"></el-option>
                        <el-option key="hash" label="hash" value="hash"></el-option>
                        <el-option key="set" label="set" value="set"></el-option>
                        <el-option key="zset" label="zset" value="zset"></el-option>
                        <el-option key="list" label="list" value="list"></el-option>
                    </el-select>
                </el-form-item>
            </el-form>

            <template #footer>
                <div class="dialog-footer">
                    <el-button @click="cancelNewKey()">取 消</el-button>
                    <el-button v-auth="'redis:data:save'" type="primary" @click="newKey">确 定</el-button>
                </div>
            </template>
        </el-dialog>
    </div>
</template>

<script lang="ts" setup>
import { redisApi } from './api';
import { ref, defineAsyncComponent, toRefs, reactive, onMounted, nextTick } from 'vue';
import { ElMessage, ElMessageBox } from 'element-plus';
import { isTrue, notBlank, notNull } from '@/common/assert';
import { TagTreeNode } from '../component/tag';
import TagTree from '../component/TagTree.vue';
import { keysToTree, sortByTreeNodes, keysToList } from './utils';

const KeyDetail = defineAsyncComponent(() => import('./KeyDetail.vue'));

/**
 * 树节点类型
 */
class NodeType {
    static Redis = 1;
    static Db = 2;
}

const treeProps = {
    label: 'name',
    children: 'children',
    isLeaf: 'leaf',
};

const defaultCount = 250;

const keyTreeRef: any = ref(null);
const rightMenuRef: any = ref(null);

const state = reactive({
    tags: [],
    redisList: [] as any,
    dbList: [],
    keyTreeHeight: window.innerHeight - 147 - 30 + 'px',
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
});

const { scanParam, keyTreeData, newKeyDialog } = toRefs(state);

onMounted(async () => {
    setHeight();
    // 监听浏览器窗口大小变化,更新对应组件高度
    window.onresize = () => setHeight();
});

const setHeight = () => {
    state.keyTreeHeight = window.innerHeight - 177 + 'px';
};

/**
 * instmap;  tagPaht -> redis info[]
 */
const instMap: Map<string, any[]> = new Map();

const getInsts = async () => {
    const res = await redisApi.redisList.request({ pageNum: 1, pageSize: 1000 });
    if (!res.total) return;
    for (const redisInfo of res.list) {
        const tagPath = redisInfo.tagPath;
        let redisInsts = instMap.get(tagPath) || [];
        redisInsts.push(redisInfo);
        instMap.set(tagPath, redisInsts);
    }
};

/**
 * 加载文件树节点
 * @param {Object} node
 * @param {Object} resolve
 */
const loadNode = async (node: any) => {
    // 一级为tagPath
    if (node.level === 0) {
        await getInsts();
        const tagPaths = instMap.keys();
        const tagNodes = [];
        for (let tagPath of tagPaths) {
            tagNodes.push(new TagTreeNode(tagPath, tagPath));
        }
        return tagNodes;
    }

    const data = node.data;
    // 点击tagPath -> 加载数据库信息列表
    if (data.type === TagTreeNode.TagPath) {
        const redisInfos = instMap.get(data.key);
        return redisInfos?.map((x: any) => {
            return new TagTreeNode(`${data.key}.${x.id}`, x.name, NodeType.Redis).withParams(x);
        });
    }

    // 点击redis实例 -> 加载库列表
    if (data.type === NodeType.Redis) {
        return await getDbs(data.params);
    }

    return [];
};

const nodeClick = (data: any) => {
    // 点击库事件
    if (data.type === NodeType.Db) {
        resetScanParam();
        state.scanParam.id = data.params.id;
        state.scanParam.db = data.params.db;
        scan();
    }
};

/**
 * 获取所有库信息
 * @param redisInfo redis信息
 */
const getDbs = async (redisInfo: any) => {
    let dbs: TagTreeNode[] = redisInfo.db.split(',').map((x: string) => {
        return new TagTreeNode(x, `db${x}`, NodeType.Db).withIsLeaf(true).withParams({
            id: redisInfo.id,
            db: x,
            name: `db${x}`,
            keys: 0,
        });
    });

    if (redisInfo.mode == 'cluster') {
        return dbs;
    }

    const res = await redisApi.redisInfo.request({ id: redisInfo.id, host: redisInfo.host, section: 'Keyspace' });
    for (let db in res.Keyspace) {
        for (let d of dbs) {
            if (db == d.params.name) {
                d.params.keys = res.Keyspace[db]?.split(',')[0]?.split('=')[1] || 0;
            }
        }
    }
    // 替换label
    dbs.forEach((e: any) => {
        e.label = `${e.params.name} [${e.params.keys}]`;
    });
    return dbs;
};

const scan = async (appendKey = false) => {
    isTrue(state.scanParam.id != null, '请先选择redis');
    notBlank(state.scanParam.db, '请先选择库');

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
    hideAllMenus();
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

const keyTreeNodeExpand = (data: any, node: any, component: any) => {
    state.keyTreeExpanded.add(data.key);
    // async sort nodes
    if (!node.customSorted) {
        node.customSorted = true;
        sortByTreeNodes(node.childNodes);
    }
};

const keyTreeNodeCollapse = (data: any, node: any, component: any) => {
    state.keyTreeExpanded.delete(data.key);
};

const rightClickNode = (event: any, data: any, node: any) => {
    hideAllMenus();

    keyTreeRef.value.setCurrentKey(node.key);
    state.rightClickNode = node;

    // nextTick for dom render
    nextTick(() => {
        let top = event.clientY;
        const menu = rightMenuRef.value;
        menu.style.display = 'block';

        // position in bottom
        if (document.body.clientHeight - top < menu.clientHeight) {
            top -= menu.clientHeight;
        }

        menu.style.left = `${event.clientX}px`;
        menu.style.top = `${top}px`;

        document.addEventListener('click', hideAllMenus, { once: true });
    });
};

const hideAllMenus = () => {
    let menus: any = document.querySelectorAll('.key-list-right-menu');

    if (menus.length === 0) {
        return;
    }

    state.rightClickNode = null;
    for (const menu of menus) {
        menu.style.display = 'none';
    }
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
    notNull(state.scanParam.id, '请先选择redis');
    notNull(state.scanParam.db, '请选择要操作的库');
    resetNewKeyInfo();
    state.newKeyDialog.visible = true;
};

const flushDb = () => {
    ElMessageBox.confirm(`确定清空[${state.scanParam.db}]库的所有key?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    })
        .then(() => {
            redisApi.flushDb
                .request({
                    id: state.scanParam.id,
                    db: state.scanParam.db,
                })
                .then(() => {
                    ElMessage.success('清除成功！');
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
    const keyInfo = state.newKeyDialog.keyInfo;
    const keyType = keyInfo.type;
    const key = keyInfo.key;
    notBlank(key, '键名不能为空');

    if (keyType == 'string') {
        await redisApi.setString.request({
            id: state.scanParam.id,
            db: state.scanParam.db,
            key: key,
            value: '',
        });
    }

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

const delKey = (key: string) => {
    ElMessageBox.confirm(`确定删除[ ${key} ] 该key?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
    })
        .then(async () => {
            await redisApi.delKey.request({
                key,
                id: state.scanParam.id,
                db: state.scanParam.db,
            });
            ElMessage.success('删除成功！');
            searchKey();

            removeDataTab(key);
        })
        .catch(() => {});
};
</script>

<style lang="scss">
.instances-pop-form {
    .el-form-item {
        margin-bottom: unset;
    }
}

.key-list-vtree {
    height: calc(100vh - 250px);
}

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

/* right menu style start */
.key-list-right-menu {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    padding: 5px;
    z-index: 99999;
    overflow: hidden;
    border-radius: 3px;
    border: 2px solid lightgrey;
    background: #fafafa;
}
.dark-mode .key-list-right-menu {
    background: #263238;
}
</style>
